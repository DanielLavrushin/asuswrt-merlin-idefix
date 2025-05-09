// Version.tsx
import { Box, Chip, Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography, useTheme } from '@mui/material';
import { useEffect, useState, useCallback, useRef } from 'react';
import axios from 'axios';
import vClean from 'version-clean';
import vCompare from 'version-compare';
import { marked } from 'marked';
import engine, { SubmitActions } from './modules/Engine';
import { useLoadingBridge } from './modules/LoadingBridge';

const COOKIE_NAME = 'idefix_dontupdate';
const GITHUB_LATEST_API = 'https://api.github.com/repos/daniellavrushin/asuswrt-merlin-idefix/releases/latest';

export default function VersionBadge() {
  const current = window.idefix?.custom_settings?.idefix_version ?? '0.0.0';
  const [latest, setLatest] = useState<string | null>(null);
  const [changelog, setChangelog] = useState<string>('');
  const [open, setOpen] = useState(false);
  const theme = useTheme();

  const hasUpdate = latest !== null && vCompare(latest, current) === 1;

  useEffect(() => {
    (async () => {
      try {
        const { data } = await axios.get(GITHUB_LATEST_API, { timeout: 5000 });
        const tag = vClean(data.tag_name);
        setLatest(tag);

        setChangelog(await marked.parse(data.body || ''));
        if (tag && vCompare(tag, current) === 1 && engine.getCookie(COOKIE_NAME) !== tag) {
          setOpen(true);
        }
      } catch {}
    })();
  }, [current]);

  const loading = useLoadingBridge();

  const handleUpdate = useCallback(async () => {
    setOpen(false);
    await engine.executeWithLoadingProgress(() => engine.submit(SubmitActions.update), loading);
  }, [loading]);

  const handleSkip = () => {
    if (latest) engine.setCookie(COOKIE_NAME, latest);
    setOpen(false);
  };

  return (
    <>
      <Box
        sx={{
          position: 'absolute',
          bottom: 8,
          right: 8,
          display: 'flex',
          alignItems: 'center',
          gap: 0.5,
          px: 1,
          py: 0.5,
          fontSize: 12,
          cursor: 'pointer',
          userSelect: 'none'
        }}
        onClick={() => setOpen(true)}
      >
        {hasUpdate && <Chip label="!" size="small" sx={{ fontWeight: 700, p: 0, height: 18, backgroundColor: '#fc0' }} />}
        <span style={{ fontWeight: 500, color: '#fc0' }}>v{current}</span>
      </Box>

      <Dialog open={open} fullWidth onClose={() => setOpen(false)}>
        <DialogTitle sx={{ bgcolor: '#2F3A3E', color: 'white' }}>{hasUpdate ? 'New version available!' : 'Version info'}</DialogTitle>
        <DialogContent dividers sx={{ typography: 'body2', bgcolor: '#4D595D', color: 'white' }}>
          <Typography gutterBottom>
            Current version: <strong>{current}</strong>
          </Typography>

          {latest ? (
            <>
              <Typography gutterBottom>
                Latest version on GitHub: <strong>{latest}</strong>
              </Typography>
              <Box
                sx={{
                  mt: 2,
                  p: 1,

                  border: '1px solid',
                  color: 'white',
                  borderRadius: 1,
                  maxHeight: 240,
                  overflow: 'auto',
                  '& h2': { mt: 1, mb: 0.5 },
                  '& ul': { pl: 2, mb: 1 }
                }}
                dangerouslySetInnerHTML={{ __html: changelog }}
              />
            </>
          ) : (
            <Typography>No release info available.</Typography>
          )}
        </DialogContent>

        <DialogActions sx={{ bgcolor: '#4D595D', color: 'white' }}>
          {hasUpdate ? (
            <>
              <Button onClick={handleSkip}>Skip&nbsp;v{latest}</Button>
              <Button variant="contained" onClick={handleUpdate}>
                Update&nbsp;now
              </Button>
            </>
          ) : (
            <Button onClick={() => setOpen(false)}>Close</Button>
          )}
        </DialogActions>
      </Dialog>
    </>
  );
}
