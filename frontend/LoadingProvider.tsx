import React, { createContext, useContext, useState, ReactNode } from 'react';
import { Backdrop, CircularProgress, LinearProgress, Stack, Typography, Fade } from '@mui/material';

type LoadingState = { open: boolean; progress?: number; message?: string };
type LoadingApi = {
  show: (msg?: string, progress?: number) => void;
  update: (msg?: string, progress?: number) => void;
  hide: () => void;
};

const Ctx = createContext<LoadingApi | null>(null);
export const useLoading = () => {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error('useLoading must be inside <LoadingProvider>');
  return ctx;
};

export const LoadingProvider = ({ children }: { children: ReactNode }) => {
  const [state, set] = useState<LoadingState>({ open: false });

  const api: LoadingApi = {
    show: (m, p) => set({ open: true, message: m, progress: p }),
    update: (m, p) => set((s) => ({ ...s, message: m ?? s.message, progress: p ?? s.progress })),
    hide: () => set({ open: false })
  };

  return (
    <Ctx.Provider value={api}>
      {children}

      <Backdrop open={state.open} sx={{ zIndex: 9998, color: 'white', backdropFilter: 'blur(3px)' }}>
        <Fade in={state.open} timeout={300}>
          <Stack spacing={2} sx={{ minWidth: 240 }} alignItems="center">
            <CircularProgress size={48} thickness={4} />
            {state.message && <Typography variant="caption">{state.message}</Typography>}
          </Stack>
        </Fade>
      </Backdrop>
    </Ctx.Provider>
  );
};
