// Version.tsx
import { Box, Chip, Dialog, DialogTitle, DialogContent, DialogActions, Button, IconButton, Typography } from '@mui/material';
import { useEffect, useState, useCallback, useRef } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import axios from 'axios';
import vClean from 'version-clean';
import vCompare from 'version-compare';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import engine, { SubmitActions } from './modules/Engine';
import { useLoadingBridge } from './modules/LoadingBridge';

const COOKIE_NAME = 'idefix_dontupdate';
const GITHUB_LATEST_API = 'https://api.github.com/repos/daniellavrushin/asuswrt-merlin-idefix/releases/latest';

const AMBER = '#FFCC00';
const PANEL = '#1e2428';
const CHROME = '#2f3a3e';
const HAIRLINE = '#3a4449';
const MONO = 'Consolas, "Courier New", monospace';

const HTTP_URL = /^https?:\/\//i;

const SANITIZE_OPTIONS = {
  ALLOWED_TAGS: [
    'p', 'br', 'hr', 'span', 'div',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'strong', 'b', 'em', 'i', 'del', 's',
    'code', 'pre', 'blockquote',
    'ul', 'ol', 'li',
    'a', 'img',
    'table', 'thead', 'tbody', 'tr', 'th', 'td'
  ],
  ALLOWED_ATTR: ['href', 'title', 'src', 'alt'],
  ALLOWED_URI_REGEXP: HTTP_URL,
  ALLOW_DATA_ATTR: false,
  ALLOW_ARIA_ATTR: false
};

DOMPurify.addHook('afterSanitizeAttributes', node => {
  if (node.hasAttribute('src') && !HTTP_URL.test(node.getAttribute('src') ?? '')) {
    node.removeAttribute('src');
  }
  if (node.tagName === 'A' && node.hasAttribute('href')) {
    node.setAttribute('target', '_blank');
    node.setAttribute('rel', 'noopener noreferrer');
  }
});

const stripLeadingHeading = (markdown: string) => markdown.replace(/^\s*#{1,6}[^\n]*(\n+|$)/, '');

const renderChangelog = (markdown: string) => DOMPurify.sanitize(marked.parse(stripLeadingHeading(markdown), { async: false }), SANITIZE_OPTIONS);

const formatDate = (iso: string) => {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
};

const notesSx = {
  fontSize: 13,
  lineHeight: 1.65,
  color: '#c3c9cc',
  '& > *:first-of-type': { mt: 0 },
  '& > *:last-child': { mb: 0 },
  '& h1, & h2, & h3, & h4': {
    fontSize: 12,
    fontWeight: 700,
    letterSpacing: 0.6,
    textTransform: 'uppercase',
    color: '#8f989c',
    mt: 2.5,
    mb: 1
  },
  '& p': { mt: 0, mb: 1.25 },
  '& ul, & ol': { pl: 2.5, mt: 0, mb: 1.25 },
  '& li': { mb: 0.75, '&::marker': { color: AMBER } },
  '& strong, & b': { color: '#fff', fontWeight: 600 },
  '& a': { color: AMBER, textDecoration: 'none', '&:hover': { textDecoration: 'underline' } },
  '& code': {
    fontFamily: MONO,
    fontSize: 12,
    color: '#e0c98a',
    bgcolor: 'rgba(255,204,0,0.07)',
    borderRadius: '3px',
    px: 0.6,
    py: '1px'
  },
  '& pre': {
    fontFamily: MONO,
    fontSize: 12,
    bgcolor: '#161b1e',
    border: `1px solid ${HAIRLINE}`,
    borderRadius: 1,
    p: 1.25,
    overflowX: 'auto',
    '& code': { bgcolor: 'transparent', px: 0, color: '#c3c9cc' }
  },
  '& blockquote': {
    m: 0,
    mb: 1.25,
    pl: 1.5,
    borderLeft: `2px solid ${AMBER}`,
    color: '#9aa3a7',
    fontStyle: 'italic'
  },
  '& hr': { border: 0, borderTop: `1px solid ${HAIRLINE}`, my: 2 },
  '& img': { maxWidth: '100%', borderRadius: 1 },
  '& table': { borderCollapse: 'collapse', width: '100%', mb: 1.25, fontSize: 12 },
  '& th, & td': { border: `1px solid ${HAIRLINE}`, px: 1, py: 0.5, textAlign: 'left' },
  '& th': { color: '#8f989c', fontWeight: 700 }
};

const scrollerSx = {
  maxHeight: 260,
  overflowY: 'auto',
  pr: 1.5,
  scrollbarWidth: 'thin',
  scrollbarColor: `${HAIRLINE} transparent`,
  '&::-webkit-scrollbar': { width: 8 },
  '&::-webkit-scrollbar-track': { bgcolor: 'transparent' },
  '&::-webkit-scrollbar-thumb': {
    bgcolor: HAIRLINE,
    borderRadius: 4,
    '&:hover': { bgcolor: '#4d5b61' }
  }
};

function VersionReadout({ current, latest, hasUpdate }: Readonly<{ current: string; latest: string | null; hasUpdate: boolean }>) {
  const label = { fontSize: 9, fontWeight: 700, letterSpacing: 1.1, textTransform: 'uppercase' as const, color: '#6b7478', mb: 0.5 };

  return (
    <Box sx={{ display: 'flex', alignItems: 'flex-end', gap: 2.5, px: 3, py: 2.5 }}>
      <Box>
        <Typography sx={label}>Installed</Typography>
        <Typography sx={{ fontFamily: MONO, fontSize: hasUpdate ? 20 : 30, lineHeight: 1, color: hasUpdate ? '#7d878b' : '#fff' }}>{current}</Typography>
      </Box>

      {hasUpdate && (
        <>
          <Box sx={{ fontFamily: MONO, fontSize: 20, lineHeight: 1, color: '#5c686d', pb: '1px' }}>&rarr;</Box>
          <Box>
            <Typography sx={{ ...label, color: AMBER }}>Available</Typography>
            <Typography sx={{ fontFamily: MONO, fontSize: 30, fontWeight: 600, lineHeight: 1, color: AMBER }}>{latest}</Typography>
          </Box>
        </>
      )}

      {!hasUpdate && latest && (
        <Typography sx={{ fontSize: 12, color: '#6b7478', pb: '3px' }}>
          Up to date&nbsp;&mdash;&nbsp;nothing newer on GitHub.
        </Typography>
      )}
    </Box>
  );
}

export default function VersionBadge() {
  const current = window.idefix?.custom_settings?.idefix_version ?? '0.0.0';
  const [latest, setLatest] = useState<string | null>(null);
  const [changelog, setChangelog] = useState<string>('');
  const [published, setPublished] = useState<string>('');
  const [checked, setChecked] = useState(false);
  const [open, setOpen] = useState(false);
  const [moreBelow, setMoreBelow] = useState(false);
  const scrollerRef = useRef<HTMLDivElement | null>(null);

  const hasUpdate = latest !== null && vCompare(latest, current) === 1;

  useEffect(() => {
    (async () => {
      try {
        const { data } = await axios.get(GITHUB_LATEST_API, { timeout: 5000 });
        const tag = vClean(data.tag_name);
        setLatest(tag);
        setPublished(data.published_at ? formatDate(data.published_at) : '');
        setChangelog(renderChangelog(data.body || ''));
        if (tag && vCompare(tag, current) === 1 && engine.getCookie(COOKIE_NAME) !== tag) {
          setOpen(true);
        }
      } catch {
      } finally {
        setChecked(true);
      }
    })();
  }, [current]);

  const syncFade = useCallback(() => {
    const el = scrollerRef.current;
    if (!el) return;
    setMoreBelow(el.scrollHeight - el.scrollTop - el.clientHeight > 4);
  }, []);

  useEffect(() => {
    if (open) requestAnimationFrame(syncFade);
  }, [open, changelog, syncFade]);

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
        onClick={() => setOpen(true)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 0.75,
          px: 1,
          py: 0.5,
          borderRadius: 1,
          fontSize: 12,
          cursor: 'pointer',
          userSelect: 'none',
          transition: 'background-color 0.15s ease',
          '&:hover': { bgcolor: 'rgba(255,204,0,0.12)' }
        }}
      >
        {hasUpdate && <Chip label="!" size="small" sx={{ fontWeight: 700, p: 0, height: 18, backgroundColor: AMBER }} />}
        <Box component="span" sx={{ fontFamily: MONO, fontWeight: 500, color: AMBER }}>
          v{current}
        </Box>
      </Box>

      <Dialog
        open={open}
        fullWidth
        maxWidth="sm"
        onClose={() => setOpen(false)}
        sx={{ zIndex: 1400 }}
        slotProps={{ paper: { sx: { bgcolor: PANEL, border: `1px solid ${HAIRLINE}`, borderRadius: 2, backgroundImage: 'none' } } }}
      >
        <DialogTitle
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            bgcolor: CHROME,
            borderBottom: `1px solid ${HAIRLINE}`,
            color: '#fff',
            fontSize: 13,
            fontWeight: 700,
            letterSpacing: 0.8,
            textTransform: 'uppercase',
            py: 1.5
          }}
        >
          Idefix version
          <IconButton
            size="small"
            onClick={() => setOpen(false)}
            sx={{ color: '#8f989c', '&:hover': { color: '#fff', bgcolor: 'rgba(255,255,255,0.08)' } }}
          >
            <CloseIcon sx={{ fontSize: 18 }} />
          </IconButton>
        </DialogTitle>

        <DialogContent sx={{ p: 0, bgcolor: PANEL }}>
          <VersionReadout current={current} latest={latest} hasUpdate={hasUpdate} />

          {changelog && (
            <Box sx={{ borderTop: `1px solid ${HAIRLINE}`, px: 3, pt: 2, pb: 2.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1, mb: 1.5 }}>
                <Typography sx={{ fontSize: 10, fontWeight: 700, letterSpacing: 1.1, textTransform: 'uppercase', color: AMBER }}>Release notes</Typography>
                {published && <Typography sx={{ fontSize: 11, color: '#6b7478' }}>{published}</Typography>}
              </Box>

              <Box sx={{ position: 'relative' }}>
                <Box ref={scrollerRef} onScroll={syncFade} sx={{ ...scrollerSx, ...notesSx }} dangerouslySetInnerHTML={{ __html: changelog }} />
                <Box
                  sx={{
                    position: 'absolute',
                    left: 0,
                    right: 0,
                    bottom: 0,
                    height: 36,
                    pointerEvents: 'none',
                    opacity: moreBelow ? 1 : 0,
                    transition: 'opacity 0.2s ease',
                    background: `linear-gradient(to bottom, rgba(30,36,40,0) 0%, rgba(30,36,40,0.85) 65%, ${PANEL} 100%)`
                  }}
                />
              </Box>
            </Box>
          )}

          {checked && !latest && (
            <Box sx={{ borderTop: `1px solid ${HAIRLINE}`, px: 3, py: 2.5 }}>
              <Typography sx={{ fontSize: 13, color: '#8f989c' }}>Could not reach GitHub to check for updates. Try again once the router is back online.</Typography>
            </Box>
          )}
        </DialogContent>

        <DialogActions sx={{ bgcolor: PANEL, borderTop: `1px solid ${HAIRLINE}`, px: 3, py: 2, gap: 1 }}>
          {hasUpdate ? (
            <>
              <Button
                onClick={handleSkip}
                sx={{
                  color: '#8f989c',
                  fontSize: 12,
                  textTransform: 'none',
                  '&:hover': { color: '#fff', bgcolor: 'rgba(255,255,255,0.06)' }
                }}
              >
                Skip v{latest}
              </Button>
              <Button
                variant="contained"
                onClick={handleUpdate}
                sx={{
                  bgcolor: AMBER,
                  color: '#1e2428',
                  fontSize: 12,
                  fontWeight: 700,
                  textTransform: 'none',
                  boxShadow: 'none',
                  px: 2.5,
                  '&:hover': { bgcolor: '#ffd633', boxShadow: 'none' }
                }}
              >
                Update now
              </Button>
            </>
          ) : (
            <Button
              onClick={() => setOpen(false)}
              sx={{
                color: '#8f989c',
                fontSize: 12,
                textTransform: 'none',
                '&:hover': { color: '#fff', bgcolor: 'rgba(255,255,255,0.06)' }
              }}
            >
              Close
            </Button>
          )}
        </DialogActions>
      </Dialog>
    </>
  );
}
