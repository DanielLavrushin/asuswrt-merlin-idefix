/* eslint-disable @typescript-eslint/no-var-requires */
/* eslint-disable no-console */
const Client = require('ssh2-sftp-client');
const fs = require('node:fs');
require('dotenv').config();

const ADDON_SCRIPT = '/jffs/addons/idefix/idefix.sh';
const SERVER_PATH = '/opt/share/idefix/idefix-server';
const SERVER_PROC = 'idefix-server';

// always (default): after the upload the server is running, whatever its
// previous state - restarted if it was up, started if it was not
// auto: only restart it if it was already running
// never: don't touch it (the new binary lands but the old one keeps running)
const RESTART_MODES = ['always', 'auto', 'never'];
const REQUESTED_MODE = process.env.IDEFIX_RESTART || 'always';
const RESTART_MODE = RESTART_MODES.includes(REQUESTED_MODE) ? REQUESTED_MODE : 'always';

if (RESTART_MODE !== REQUESTED_MODE) {
  console.warn(`Unknown IDEFIX_RESTART="${REQUESTED_MODE}", falling back to "always".`);
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

function exec(sftp, command) {
  return new Promise((resolve, reject) => {
    sftp.client.exec(command, (err, stream) => {
      if (err) return reject(err);
      let stdout = '';
      let stderr = '';
      stream
        .on('close', (code) => resolve({ code: code ?? 0, stdout, stderr }))
        .on('data', (d) => (stdout += d.toString()))
        .stderr.on('data', (d) => (stderr += d.toString()));
    });
  });
}

async function serverPid(sftp) {
  const { stdout } = await exec(sftp, `/bin/pidof ${SERVER_PROC} 2>/dev/null || true`);
  return stdout.trim();
}

async function stopServer(sftp) {
  console.log('Stopping idefix-server...');
  await exec(sftp, `sh ${ADDON_SCRIPT} stop >/tmp/idefix-sync.log 2>&1 || true`);

  for (let i = 0; i < 10; i++) {
    if (!(await serverPid(sftp))) {
      console.log('idefix-server stopped.');
      return true;
    }
    await sleep(500);
  }

  console.warn('idefix-server did not stop gracefully, forcing kill...');
  await exec(sftp, `killall -9 ${SERVER_PROC} 2>/dev/null || true`);
  await sleep(500);
  return !(await serverPid(sftp));
}

async function startServer(sftp) {
  console.log('Starting idefix-server...');
  // redirect output to a file so the exec channel closes instead of hanging
  // on the backgrounded server inheriting stdout
  await exec(sftp, `sh ${ADDON_SCRIPT} start >/tmp/idefix-sync.log 2>&1`);

  for (let i = 0; i < 10; i++) {
    const pid = await serverPid(sftp);
    if (pid) {
      console.log(`idefix-server started with PID: ${pid}`);
      return true;
    }
    await sleep(500);
  }

  const { stdout } = await exec(sftp, 'cat /tmp/idefix-sync.log 2>/dev/null || true');
  console.error('idefix-server failed to start.', stdout.trim());
  return false;
}

async function pushFiles(sftp, serverBin) {
  // Ensure directories exist
  await sftp.mkdir('/opt/share/idefix', true);
  await sftp.mkdir('/jffs/addons/idefix', true);

  // Upload files
  await sftp.fastPut('dist/index.asp', '/opt/share/idefix/index.asp');
  await sftp.fastPut('dist/app.js', '/opt/share/idefix/app.js');
  await sftp.fastPut('dist/idefix', '/jffs/addons/idefix/idefix.sh');

  // Set executable permissions
  await sftp.chmod('/jffs/addons/idefix/idefix.sh', '755');

  if (serverBin) {
    // upload next to the target and rename into place: an atomic swap that
    // also dodges ETXTBSY if the old process is still shutting down
    const staged = `${SERVER_PATH}.new`;
    await sftp.fastPut(serverBin, staged);
    await sftp.chmod(staged, '755');
    const { code, stderr } = await exec(sftp, `mv -f ${staged} ${SERVER_PATH}`);
    if (code !== 0) throw new Error(`Failed to replace idefix-server: ${stderr.trim()}`);
    console.log('idefix-server uploaded.');
  }

  console.log('Files uploaded and permissions set successfully.');
}

// don't leave the router with a dead server because the upload blew up
async function recoverAfterError(sftp, stopped) {
  if (!stopped) return;
  try {
    await startServer(sftp);
  } catch (err) {
    console.error('Failed to restart idefix-server after error:', err);
  }
}

async function uploadFiles() {
  const sftp = new Client();
  let stopped = false;

  try {
    await sftp.connect({
      host: process.env.SFTP_ROUTER,
      port: 22,
      username: process.env.SFTP_USERNAME,
      password: process.env.SFTP_PASSWORD,
      readyTimeout: 3000
    });

    console.log('Connected via SFTP.');

    const goArch = process.env.IDEFIX_GOARCH || 'arm64';
    const binPath = `dist/server/${goArch}/idefix-server`;
    const serverBin = fs.existsSync(binPath) ? binPath : null;

    // upload first: the staged rename means the running binary is never in the
    // way, so downtime is just the restart instead of the whole transfer
    await pushFiles(sftp, serverBin);

    const wasRunning = Boolean(await serverPid(sftp));
    const shouldRun =
      Boolean(serverBin) && RESTART_MODE !== 'never' && (RESTART_MODE === 'always' || wasRunning);

    if (!shouldRun) {
      console.log(`idefix-server left as-is (running: ${wasRunning}).`);
      return;
    }

    // only stop what is actually up - starting from stopped is the normal
    // case here, not an error
    if (wasRunning) {
      stopped = await stopServer(sftp);
    }

    if (!(await startServer(sftp))) process.exitCode = 1;
  } catch (err) {
    console.error('Error uploading files via SFTP:', err);
    process.exitCode = 1;
    await recoverAfterError(sftp, stopped);
  } finally {
    sftp.end();
  }
}

uploadFiles();
