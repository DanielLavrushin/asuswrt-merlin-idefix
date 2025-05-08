import React, { useEffect, useLayoutEffect, useRef, useState, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { AttachAddon } from '@xterm/addon-attach';
import '@xterm/xterm/css/xterm.css';
import { Backdrop, Box, Button, CircularProgress, Stack, Typography } from '@mui/material';
import './IdefixTerminal.css';
import engine, { SubmitActions } from './modules/Engine';

export interface TerminalProps {
  endpoint?: string;
  protocol?: string;
  cols?: number;
  rows?: number;
}

const secure = window.location.protocol === 'https:';

export const IdefixTerminal: React.FC<TerminalProps> = ({ endpoint = `${secure ? 'wss' : 'ws'}://${window.location.hostname}:8787/ws`, protocol = 'idefix', cols = 0, rows = 30 }) => {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const attachAddonRef = useRef<AttachAddon | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const [loading, setLoading] = useState<boolean>(false);

  const [connected, setConnected] = useState<boolean>(true);

  const connectSocket = useCallback(
    async (shouldStartServer?: boolean) => {
      if (!termRef.current) return;
      if (shouldStartServer) {
        setLoading(true);
        await engine.submit(SubmitActions.restart);
        await engine.delay(2000);
      }
      attachAddonRef.current?.dispose();
      socketRef.current?.close();

      const socket = new WebSocket(endpoint, protocol);
      socket.binaryType = 'arraybuffer';

      socket.addEventListener('open', () => setConnected(true));
      socket.addEventListener('close', () => setConnected(false));
      socket.addEventListener('error', () => setConnected(false));

      const attachAddon = new AttachAddon(socket, { bidirectional: true });
      termRef.current.loadAddon(attachAddon);

      socketRef.current = socket;
      attachAddonRef.current = attachAddon;
      setLoading(false);
    },
    [endpoint, protocol]
  );

  useLayoutEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      fontFamily: 'Consolas, monospace',
      fontSize: 14,
      cols,
      rows
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(containerRef.current);
    requestAnimationFrame(() => fitAddon.fit());

    termRef.current = term;
    fitAddonRef.current = fitAddon;

    connectSocket(); // initial connect

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      socketRef.current?.close();
      term.dispose();
    };
  }, [cols, rows, connectSocket]);

  return (
    <Stack sx={{ position: 'relative' }}>
      <Box ref={containerRef} sx={{ flex: 1, p: 0.2 }} />

      <Backdrop
        open={!connected}
        sx={{
          position: 'absolute',
          zIndex: 1,
          color: 'white',
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
    </Stack>
  );
};

export default IdefixTerminal;
