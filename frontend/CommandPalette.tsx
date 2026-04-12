import { useState, useEffect, useRef, useMemo } from 'react';
import { Box, IconButton, InputAdornment, Paper, TextField, Tooltip, Typography } from '@mui/material';
import KeyboardCommandKeyIcon from '@mui/icons-material/KeyboardCommandKey';
import defaultCommands from './quick-commands.json';

export interface QuickCommand {
  label: string;
  cmd: string;
  category: string;
}

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  onSelect: (cmd: string) => void;
}

export default function CommandPalette({ open, onClose, onSelect }: Readonly<CommandPaletteProps>) {
  const [filter, setFilter] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const [selectedIdx, setSelectedIdx] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    if (!filter) return defaultCommands;
    const q = filter.toLowerCase();
    return defaultCommands.filter((c) => c.label.toLowerCase().includes(q) || c.cmd.toLowerCase().includes(q) || c.category.toLowerCase().includes(q));
  }, [filter]);

  useEffect(() => {
    if (open) {
      setFilter('');
      setSelectedIdx(0);
      setTimeout(() => inputRef.current?.focus(), 50);
    }
  }, [open]);

  useEffect(() => {
    setSelectedIdx(0);
  }, [filter]);

  // scroll selected item into view
  useEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const item = list.querySelector(`[data-idx="${selectedIdx}"]`) as HTMLElement;
    item?.scrollIntoView({ block: 'nearest' });
  }, [selectedIdx]);

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIdx((i: number) => Math.min(i + 1, filtered.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIdx((i: number) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter' && filtered.length > 0) {
      e.preventDefault();
      onSelect(filtered[selectedIdx].cmd);
      onClose();
    }
  };

  if (!open) return null;

  // group by category
  type IndexedCommand = QuickCommand & { globalIdx: number };
  const groups: { category: string; items: IndexedCommand[] }[] = [];
  const catMap = new Map<string, IndexedCommand[]>();
  filtered.forEach((c: QuickCommand, i: number) => {
    if (!catMap.has(c.category)) catMap.set(c.category, []);
    catMap.get(c.category)?.push({ ...c, globalIdx: i });
  });
  catMap.forEach((items, category) => groups.push({ category, items }));

  return (
    <>
      <Box onClick={onClose} sx={{ position: 'fixed', inset: 0, zIndex: 1400 }} />
      <Paper
        elevation={8}
        onKeyDown={handleKey}
        sx={{
          position: 'fixed',
          top: '15%',
          left: '50%',
          transform: 'translateX(-50%)',
          width: 480,
          maxHeight: '60vh',
          zIndex: 1401,
          bgcolor: '#1e2428',
          border: '1px solid #3a4449',
          borderRadius: 2,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden'
        }}
      >
        <Box sx={{ p: 1.5, borderBottom: '1px solid #3a4449' }}>
          <TextField
            inputRef={inputRef}
            fullWidth
            size="small"
            placeholder="Search commands..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            slotProps={{
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <KeyboardCommandKeyIcon sx={{ fontSize: 18, color: '#888' }} />
                  </InputAdornment>
                ),
                sx: {
                  bgcolor: '#2a3035',
                  color: '#ddd',
                  fontSize: 14,
                  '& fieldset': { border: 'none' }
                }
              }
            }}
          />
        </Box>
        <Box ref={listRef} sx={{ overflow: 'auto', maxHeight: '50vh', py: 0.5 }}>
          {groups.length === 0 && <Typography sx={{ p: 2, color: '#666', textAlign: 'center', fontSize: 13 }}>No matching commands</Typography>}
          {groups.map((g) => (
            <Box key={g.category}>
              <Typography sx={{ px: 1.5, pt: 1, pb: 0.5, fontSize: 11, color: '#FFCC00', fontWeight: 700, textTransform: 'uppercase', letterSpacing: 0.5 }}>{g.category}</Typography>
              {g.items.map((c) => (
                <Box
                  key={c.globalIdx}
                  data-idx={c.globalIdx}
                  onClick={() => {
                    onSelect(c.cmd);
                    onClose();
                  }}
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    px: 1.5,
                    py: 0.75,
                    cursor: 'pointer',
                    bgcolor: c.globalIdx === selectedIdx ? 'rgba(255,204,0,0.1)' : 'transparent',
                    '&:hover': { bgcolor: 'rgba(255,204,0,0.08)' }
                  }}
                >
                  <Typography sx={{ fontSize: 13, color: '#ccc' }}>{c.label}</Typography>
                  <Typography sx={{ fontSize: 11, color: '#666', fontFamily: 'Consolas, monospace', ml: 2, flexShrink: 0, maxWidth: 240, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{c.cmd}</Typography>
                </Box>
              ))}
            </Box>
          ))}
        </Box>
        <Box sx={{ borderTop: '1px solid #3a4449', px: 1.5, py: 0.75, display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
          <Typography sx={{ fontSize: 11, color: '#555' }}>
            <kbd style={{ padding: '1px 4px', border: '1px solid #444', borderRadius: 3, fontSize: 10 }}>↑↓</kbd> navigate <kbd style={{ padding: '1px 4px', border: '1px solid #444', borderRadius: 3, fontSize: 10 }}>Enter</kbd> run <kbd style={{ padding: '1px 4px', border: '1px solid #444', borderRadius: 3, fontSize: 10 }}>Esc</kbd> close
          </Typography>
        </Box>
      </Paper>
    </>
  );
}

export function CommandPaletteButton({ onClick }: Readonly<{ onClick: () => void }>) {
  return (
    <Tooltip title="Quick Commands (Ctrl+K)" placement="bottom">
      <IconButton
        size="small"
        onClick={onClick}
        sx={{
          color: '#FFCC00',
          ml: 0.5,
          width: 28,
          height: 28,
          '&:hover': { color: '#fff', bgcolor: 'rgba(255,204,0,0.15)' }
        }}
      >
        <KeyboardCommandKeyIcon sx={{ fontSize: 18 }} />
      </IconButton>
    </Tooltip>
  );
}
