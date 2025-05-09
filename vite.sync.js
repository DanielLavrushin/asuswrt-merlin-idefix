/* eslint-disable @typescript-eslint/no-var-requires */
/* eslint-disable no-console */
const Client = require('ssh2-sftp-client');
require('dotenv').config();

async function uploadFiles() {
  const sftp = new Client();

  try {
    await sftp.connect({
      host: process.env.SFTP_ROUTER,
      port: 22,
      username: process.env.SFTP_USERNAME,
      password: process.env.SFTP_PASSWORD,
      readyTimeout: 3000
    });

    console.log('Connected via SFTP.');

    // Ensure directories exist
    await sftp.mkdir('/opt/share/idefix', true);

    // Upload files
    await sftp.fastPut('dist/index.asp', '/opt/share/idefix/index.asp');
    await sftp.fastPut('dist/app.js', '/opt/share/idefix/app.js');
    await sftp.fastPut('dist/idefix', '/jffs/scripts/idefix');

    // Set executable permissions
    await sftp.chmod('/jffs/scripts/idefix', '755');

    console.log('Files uploaded and permissions set successfully.');
  } catch (err) {
    console.error('Error uploading files via SFTP:', err);
  } finally {
    sftp.end();
  }
}

uploadFiles();
