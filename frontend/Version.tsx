// Version.tsx
import { Box, Chip, Dialog, DialogTitle, DialogContent, DialogActions, Button, Typography, useTheme } from '@mui/material';
import { useEffect, useState, useCallback, useRef } from 'react';
import axios from 'axios';
import vClean from 'version-clean';
import vCompare from 'version-compare';
import { marked } from 'marked';
import engine, { SubmitActions } from './modules/Engine';
import React from 'react';

const COOKIE_NAME = 'idefix_dontupdate';
const GITHUB_LATEST_API = 'https://api.github.com/repos/daniellavrushin/asuswrt-merlin-idefix/releases/latest';

export default function VersionBadge() {
  const current = window.idefix?.custom_settings?.idefix_version ?? '0.0.0';
  const [latest, setLatest] = useState<string | null>(null);
  const [changelog, setChangelog] = useState<string>('');
  const [open, setOpen] = useState(false);
  const theme = useTheme();

  const hasUpdate = latest !== null && vCompare(latest, current) === 1 && engine.getCookie(COOKIE_NAME) !== latest;

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

  const handleUpdate = useCallback(async () => {
    setOpen(false);
    await engine.executeWithLoadingProgress(async () => {
      await engine.submit(SubmitActions.update);
    });
  }, []);

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
        {hasUpdate && <Chip label="!" color="info" size="small" sx={{ fontWeight: 700, p: 0, height: 18 }} />}
        <span style={{ fontWeight: 500 }}>v{current}</span>
      </Box>

      {/* ------ modal ------ */}
      <Dialog open={open} maxWidth="sm" fullWidth onClose={() => setOpen(false)}>
        <DialogTitle>New version available</DialogTitle>

        <DialogContent dividers sx={{ typography: 'body2' }}>
          <Typography gutterBottom>
            Current version:&nbsp;<strong>{current}</strong>
          </Typography>

          {latest ? (
            <>
              <Typography gutterBottom>
                Latest version on GitHub:&nbsp;
                <strong>{latest}</strong>
              </Typography>
              <Box
                sx={{
                  mt: 2,
                  p: 1,
                  bgcolor: theme.palette.mode === 'dark' ? '#2f3a3e' : '#f5f5f5',
                  border: '1px solid',
                  borderColor: theme.palette.mode === 'dark' ? '#222' : 'divider',
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

        <DialogActions>
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
