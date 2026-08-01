import './IdefixTerminal.css';
import '@xterm/xterm/css/xterm.css';
import React, { useEffect, useLayoutEffect, useRef, useState, useCallback, useImperativeHandle, forwardRef } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { AttachAddon } from '@xterm/addon-attach';

import { Backdrop, Box, Button, CircularProgress, IconButton, Stack, Tooltip, Typography } from '@mui/material';
import engine, { randomId, SubmitActions } from './modules/Engine';
import CloseFullscreenIcon from '@mui/icons-material/CloseFullscreen';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';
import SaveAltIcon from '@mui/icons-material/SaveAlt';

export interface TerminalHandle {
  sendCommand: (cmd: string) => void;
  getBufferText: () => string;
  clear: () => void;
}

export interface TerminalProps {
  onStatusChange?: (s: 'connected' | 'reconnecting' | 'offline') => void;
}

type Phase = 'connected' | 'connecting' | 'offline';

const protocol = 'idefix';
const cols = 0;
const rows = 0;

const RECONNECT_DELAYS = [250, 500, 1000, 2000, 4000, 8000];

export const IdefixTerminal = forwardRef<TerminalHandle, TerminalProps>(({ onStatusChange = () => {} }, ref) => {
  const terminalRef = useRef<HTMLDivElement | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const attachAddonRef = useRef<AttachAddon | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const sessionIdRef = useRef<string>('');
  if (!sessionIdRef.current) sessionIdRef.current = randomId();
  const retireSocketRef = useRef<(() => void) | null>(null);
  const retryTimerRef = useRef<number | null>(null);
  const attemptRef = useRef(0);
  const connectingRef = useRef(false);
  const disposedRef = useRef(false);
  const connectRef = useRef<(restartServer?: boolean) => Promise<void>>(() => Promise.resolve());
  const lastSizeRef = useRef({ cols: 0, rows: 0 });

  const [phase, setPhase] = useState<Phase>('connecting');
  const [wide, setWide] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const toggleWide = () => setWide((p) => !p);

  const statusCbRef = useRef(onStatusChange);
  statusCbRef.current = onStatusChange;

  const goPhase = useCallback((p: Phase) => {
    setPhase(p);
    statusCbRef.current?.(p === 'connecting' ? 'reconnecting' : p);
  }, []);

  useImperativeHandle(ref, () => ({
    sendCommand(cmd: string) {
      const sock = socketRef.current;
      if (sock && sock.readyState === WebSocket.OPEN) {
        sock.send(new TextEncoder().encode(cmd + '\n'));
      }
    },
    getBufferText() {
      const term = termRef.current;
      if (!term) return '';
      const buf = term.buffer.active;
      const lines: string[] = [];
      for (let i = 0; i < buf.length; i++) {
        const line = buf.getLine(i);
        if (line) lines.push(line.translateToString(true));
      }
      while (lines.length > 0 && lines[lines.length - 1].trim() === '') lines.pop();
      return lines.join('\n');
    },
    clear() {
      termRef.current?.clear();
    }
  }));

  const exportBuffer = () => {
    const term = termRef.current;
    if (!term) return;
    const buf = term.buffer.active;
    const lines: string[] = [];
    for (let i = 0; i < buf.length; i++) {
      const line = buf.getLine(i);
      if (line) lines.push(line.translateToString(true));
    }
    while (lines.length > 0 && lines[lines.length - 1].trim() === '') lines.pop();
    const text = lines.join('\n') + '\n';
    const blob = new Blob([text], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `idefix-session-${new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)}.log`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  };

  const buildEndpoint = (secure: boolean) => `${secure ? 'wss' : 'ws'}://${window.location.hostname}:8787/ws`;

  const fitAndResize = () => {
    const term = termRef.current;
    const fit = fitAddonRef.current;
    const sock = socketRef.current;
    const host = terminalRef.current;
    if (!term || !fit || !host) return;
    if (host.clientWidth === 0 || host.clientHeight === 0) return;

    fit.fit();
    if (term.cols === lastSizeRef.current.cols && term.rows === lastSizeRef.current.rows) return;
    lastSizeRef.current = { cols: term.cols, rows: term.rows };

    if (sock && sock.readyState === WebSocket.OPEN) {
      const msg = JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows
      });
      sock.send(msg);
    }
  };

  useEffect(() => {
    fitAndResize();
  }, [wide]);

  useEffect(() => {
    const node = terminalRef.current;
    if (!node || typeof ResizeObserver === 'undefined') return;

    let frame = 0;
    const observer = new ResizeObserver(() => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(fitAndResize);
    });

    observer.observe(node);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
    };
  }, []);

  const socketAlive = () => {
    const sock = socketRef.current;
    return !!sock && (sock.readyState === WebSocket.OPEN || sock.readyState === WebSocket.CONNECTING);
  };

  const clearRetry = () => {
    if (retryTimerRef.current !== null) {
      window.clearTimeout(retryTimerRef.current);
      retryTimerRef.current = null;
    }
  };

  const scheduleReconnect = useCallback(() => {
    if (disposedRef.current || socketAlive() || retryTimerRef.current !== null) return;

    if (attemptRef.current >= RECONNECT_DELAYS.length) {
      goPhase('offline');
      return;
    }

    const delay = RECONNECT_DELAYS[attemptRef.current];
    attemptRef.current += 1;
    goPhase('connecting');
    retryTimerRef.current = window.setTimeout(() => {
      retryTimerRef.current = null;
      void connectRef.current();
    }, delay);
  }, [goPhase]);

  const connectSocket = useCallback(
    async (shouldStartServer = false) => {
      if (disposedRef.current || connectingRef.current || !termRef.current) return;
      if (!shouldStartServer && socketAlive()) return;

      connectingRef.current = true;
      clearRetry();
      goPhase('connecting');

      try {
        if (shouldStartServer) {
          attemptRef.current = 0;
          await engine.submit(SubmitActions.restart);
          await engine.delay(4000);
          if (disposedRef.current) return;
        }

        const token = await engine.refreshToken();
        if (disposedRef.current) return;

        if (!token?.sig || !token.cl || !token.ts || !token.n) {
          scheduleReconnect();
          return;
        }

        retireSocketRef.current?.();

        const url = new URL(buildEndpoint(window.location.protocol === 'https:'), window.location.href);
        url.searchParams.set('sid', sessionIdRef.current);

        const credential = `${protocol}.auth.${token.cl}.${token.ts.toFixed()}.${token.n}.${token.sig}`;
        const socket = new WebSocket(url, [protocol, credential]);
        socket.binaryType = 'arraybuffer';

        let retired = false;
        const dropped = () => {
          if (retired || disposedRef.current || socketRef.current !== socket) return;
          scheduleReconnect();
        };

        socket.addEventListener('open', () => {
          if (retired) return;
          attemptRef.current = 0;
          termRef.current?.reset();
          goPhase('connected');
          lastSizeRef.current = { cols: 0, rows: 0 };
          fitAndResize();
        });

        socket.addEventListener('close', dropped);
        socket.addEventListener('error', dropped);

        const attachAddon = new AttachAddon(socket, { bidirectional: true });
        termRef.current.loadAddon(attachAddon);

        retireSocketRef.current = () => {
          retired = true;
          attachAddon.dispose();
          if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
            socket.close(1000, 'replaced');
          }
          if (socketRef.current === socket) socketRef.current = null;
          if (attachAddonRef.current === attachAddon) attachAddonRef.current = null;
        };

        socketRef.current = socket;
        attachAddonRef.current = attachAddon;
      } catch (err) {
        console.error('idefix: connect failed', err);
        scheduleReconnect();
      } finally {
        connectingRef.current = false;
      }
    },
    [goPhase, scheduleReconnect]
  );

  connectRef.current = connectSocket;

  useLayoutEffect(() => {
    if (!terminalRef.current) return;
    disposedRef.current = false;

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

    void connectRef.current();

    const handleResize = () => fitAndResize();

    const resume = () => {
      if (disposedRef.current || document.visibilityState === 'hidden' || socketAlive()) return;
      attemptRef.current = 0;
      clearRetry();
      void connectRef.current();
    };
    const handlePageShow = (e: PageTransitionEvent) => {
      if (e.persisted) resume();
    };

    window.addEventListener('resize', handleResize);
    window.addEventListener('pageshow', handlePageShow);
    window.addEventListener('focus', resume);
    window.addEventListener('online', resume);
    document.addEventListener('visibilitychange', resume);

    return () => {
      disposedRef.current = true;
      clearRetry();
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('pageshow', handlePageShow);
      window.removeEventListener('focus', resume);
      window.removeEventListener('online', resume);
      document.removeEventListener('visibilitychange', resume);

      const sock = socketRef.current;
      if (sock?.readyState === WebSocket.OPEN) {
        try {
          sock.send(JSON.stringify({ type: 'bye' }));
        } catch {}
      }
      retireSocketRef.current?.();
      retireSocketRef.current = null;
      term.dispose();
      termRef.current = null;
    };
  }, []);

  const showOverlay = phase !== 'connected';

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
          flex: wide ? undefined : 1,
          position: wide ? 'fixed' : 'relative',
          inset: wide ? 0 : 'auto',
          m: wide ? 'auto' : 0,
          width: wide ? '70vw' : '100%',
          height: wide ? '80vh' : '100%',
          zIndex: wide ? 1300 : 'auto',
          minHeight: 0,
          overflow: 'hidden'
        }}
      >
        <Stack direction="row" spacing={0.5} sx={{ position: 'absolute', top: 6, right: 4, zIndex: 1301 }}>
          <Tooltip title="Export session" placement="bottom">
            <IconButton
              aria-label="Export session"
              onClick={exportBuffer}
              size="small"
              sx={{
                bgcolor: 'rgba(0,0,0,.35)',
                color: 'white',
                '&:hover': { bgcolor: 'rgba(0,0,0,.5)' }
              }}
            >
              <SaveAltIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title={wide ? 'Shrink' : 'Enlarge'} placement="bottom">
            <IconButton
              aria-label={wide ? 'Shrink' : 'Enlarge'}
              onClick={toggleWide}
              size="small"
              sx={{
                bgcolor: 'rgba(0,0,0,.35)',
                color: 'white',
                '&:hover': { bgcolor: 'rgba(0,0,0,.5)' }
              }}
            >
              {wide ? <CloseFullscreenIcon fontSize="small" /> : <OpenInFullIcon fontSize="small" />}
            </IconButton>
          </Tooltip>
        </Stack>

        <Box ref={terminalRef} sx={{ flex: 1, minHeight: 0, overflow: 'hidden', p: 0.2 }} />
        {showOverlay && (
          <Backdrop
            open={showOverlay}
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
            {phase === 'connecting' && (
              <Stack spacing={2} alignItems="center">
                <CircularProgress size={40} thickness={4} />
                <Typography variant="body2">Connecting…</Typography>
              </Stack>
            )}
            {phase === 'offline' && (
              <Button variant="contained" size="small" onClick={() => void connectSocket(true)} sx={{ alignSelf: 'center', mt: 1 }}>
                Reconnect
              </Button>
            )}
          </Backdrop>
        )}
      </Stack>
    </>
  );
});

IdefixTerminal.displayName = 'IdefixTerminal';
export default IdefixTerminal;
