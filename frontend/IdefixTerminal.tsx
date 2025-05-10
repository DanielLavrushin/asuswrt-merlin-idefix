import React, { useEffect, useLayoutEffect, useRef, useState, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { AttachAddon } from '@xterm/addon-attach';
import '@xterm/xterm/css/xterm.css';
import { Backdrop, Box, Button, CircularProgress, IconButton, Stack, Typography } from '@mui/material';
import './IdefixTerminal.css';
import engine, { EngineToken, SubmitActions } from './modules/Engine';
import CloseFullscreenIcon from '@mui/icons-material/CloseFullscreen';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';

export interface TerminalProps {
  onStatusChange?: (s: 'connected' | 'reconnecting' | 'offline') => void;
}

const protocol = 'idefix';
const cols = 0;
const rows = 50;

export const IdefixTerminal: React.FC<TerminalProps> = ({ onStatusChange }) => {
  const terminalRef = useRef<HTMLDivElement | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const attachAddonRef = useRef<AttachAddon | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const [loading, setLoading] = useState<boolean>(false);
  const [token, setToken] = useState<EngineToken | undefined>();
  const report = (s: 'connected' | 'reconnecting' | 'offline') => onStatusChange?.(s);
  const [wide, setWide] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const toggleWide = () => {
    setWide((p) => !p);
    requestAnimationFrame(fitAndResize);
  };

  const [connected, setConnected] = useState<boolean>(true);

  const buildEndpoint = (secure: boolean) => `${secure ? 'wss' : 'ws'}://${window.location.hostname}:8787/ws`;

  const fitAndResize = () => {
    const term = termRef.current;
    const fit = fitAddonRef.current;
    const sock = socketRef.current;
    if (!term || !fit) return;

    fit.fit();

    if (sock && sock.readyState === WebSocket.OPEN) {
      const msg = JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows
      });
      sock.send(msg);
    }
  };

  const fetchToken = async () => {
    setLoading(true);
    const token = generateToken();
    await engine.submit(SubmitActions.generateToken, { client_token: token });
    await engine.getServerToken();
    setToken(engine.token);
    setLoading(false);
  };

  useEffect(() => {
    setConnected(false);

    fetchToken();
  }, []);

  const generateToken = () => {
    const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let token = '';
    for (let i = 0; i < 16; i++) {
      token += characters.charAt(Math.floor(Math.random() * characters.length));
    }
    return token;
  };

  const connectSocket = useCallback(
    async (shouldStartServer?: boolean) => {
      if (!engine.token) return;
      if (!termRef.current) return;
      if (shouldStartServer) {
        report('reconnecting');
        setLoading(true);
        await engine.submit(SubmitActions.restart);
        await engine.delay(3000);
      }
      attachAddonRef.current?.dispose();
      socketRef.current?.close();

      const base = buildEndpoint(window.location.protocol === 'https:');

      const url = new URL(base, window.location.href);
      url.searchParams.set('c', engine.token?.cl || '');
      url.searchParams.set('t', engine.token?.ts?.toFixed() || '');
      url.searchParams.set('s', engine.token?.sig || '');

      const socket = new WebSocket(url, protocol);
      socket.binaryType = 'arraybuffer';

      socket.addEventListener('open', () => {
        setConnected(true);
        report('connected');
      });
      socket.addEventListener('close', async (evt) => {
        if (token?.ts && Date.now() - token.ts * 1000 > 110 * 1000) {
          await fetchToken();
          connectSocket();
        } else {
          setConnected(false);
          report('offline');
        }
      });
      socket.addEventListener('error', () => {
        setConnected(false);
        report('offline');
      });

      const attachAddon = new AttachAddon(socket, { bidirectional: true });
      termRef.current.loadAddon(attachAddon);

      socketRef.current = socket;
      attachAddonRef.current = attachAddon;
      setLoading(false);
    },
    [protocol]
  );

  useLayoutEffect(() => {
    if (!terminalRef.current) return;
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: 'Consolas, monospace',
      fontSize: 14,
      cols,
      rows
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(terminalRef.current);
    requestAnimationFrame(() => fitAndResize());

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    connectSocket();

    const handleResize = () => fitAndResize();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      socketRef.current?.close();
      term.dispose();
    };
  }, [token]);

  const showOverlay = !connected || loading;

  return (
    <>
      {wide && (
        <Box
          onClick={toggleWide}
          sx={{
            position: 'fixed',
            inset: 0,
            zIndex: 1299,
            backdropFilter: 'blur(3px) brightness(.6)'
          }}
        />
      )}

      <Stack
        ref={wrapperRef}
        sx={{
          position: wide ? 'fixed' : 'relative',
          inset: wide ? 0 : 'auto',
          m: wide ? 'auto' : 0,
          width: wide ? '70vw' : '100%',
          height: wide ? '80vh' : 'unset',
          zIndex: wide ? 1300 : 'auto',
          transition: 'all .25s cubic-bezier(.4,0,.2,1)'
        }}
      >
        <IconButton
          aria-label={wide ? 'Shrink' : 'Enlarge'}
          onClick={toggleWide}
          size="small"
          sx={{
            position: 'absolute',
            top: 4,
            right: 4,
            bgcolor: 'rgba(0,0,0,.35)',
            color: 'white',
            '&:hover': { bgcolor: 'rgba(0,0,0,.5)' },
            zIndex: 1301
          }}
        >
          {wide ? <CloseFullscreenIcon fontSize="small" /> : <OpenInFullIcon fontSize="small" />}
        </IconButton>

        <Box ref={terminalRef} sx={{ flex: 1, p: 0.2 }} />
        {showOverlay && (
          <Backdrop
            open={!connected}
            sx={{
              position: 'absolute',
              zIndex: 1304,
              inset: 0,
              color: 'white',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              backdropFilter: 'blur(3px) brightness(0.6)'
            }}
            transitionDuration={300}
          >
            {loading && (
              <Stack spacing={2} alignItems="center">
                <CircularProgress size={40} thickness={4} />
                <Typography variant="body2">Connecting…</Typography>
              </Stack>
            )}
            {!loading && (
              <Button variant="contained" size="small" onClick={() => connectSocket(true)} sx={{ alignSelf: 'center', mt: 1 }}>
                Reconnect
              </Button>
            )}
          </Backdrop>
        )}
      </Stack>
    </>
  );
};

export default IdefixTerminal;
