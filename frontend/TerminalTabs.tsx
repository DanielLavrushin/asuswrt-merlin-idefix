import { useState, useCallback } from 'react';
import { Box, IconButton } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import CloseIcon from '@mui/icons-material/Close';
import TerminalIcon from '@mui/icons-material/Terminal';
import IdefixTerminal from './IdefixTerminal';

interface TabSession {
  id: string;
  num: number;
}

const MAX_TABS = 6;

function genId() {
  return Math.random().toString(36).slice(2) + Date.now().toString(36);
}

export default function TerminalTabs() {
  const [tabs, setTabs] = useState<TabSession[]>([{ id: genId(), num: 1 }]);
  const [activeId, setActiveId] = useState(tabs[0].id);

  const findNextNum = (existing: TabSession[]) => {
    const used = new Set(existing.map((t) => t.num));
    let n = 1;
    while (used.has(n)) n++;
    return n;
  };

  const addTab = useCallback(() => {
    if (tabs.length >= MAX_TABS) return;
    const newTab = { id: genId(), num: findNextNum(tabs) };
    setTabs((prev) => [...prev, newTab]);
    setActiveId(newTab.id);
  }, [tabs]);

  const closeTab = useCallback(
    (id: string) => {
      setTabs((prev) => {
        const next = prev.filter((t) => t.id !== id);
        if (next.length === 0) {
          const fresh = { id: genId(), num: 1 };
          setActiveId(fresh.id);
          return [fresh];
        }
        if (activeId === id) {
          const idx = prev.findIndex((t) => t.id === id);
          const newActive = next[Math.min(idx, next.length - 1)];
          setActiveId(newActive.id);
        }
        return next;
      });
    },
    [activeId]
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Tab bar */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          bgcolor: '#4d595d',
          borderBottom: '1px solid #3a4449',
          px: 0.5,
          gap: 0.5,
          pt: 2
        }}
      >
        {tabs.map((t) => {
          const isActive = activeId === t.id;
          return (
            <Box
              key={t.id}
              onClick={() => setActiveId(t.id)}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 0.75,
                px: 1.5,
                py: 1.15,
                cursor: 'pointer',
                borderRadius: '4px 4px 0 0',
                bgcolor: '#2f3a3e',
                color: isActive ? '#fff' : '#888',
                fontWeight: isActive ? 800 : 500,
                borderTop: isActive ? '2px solid #FFCC00' : '2px solid transparent',
                transition: 'all 0.15s ease',
                '&:hover': {
                  bgcolor: isActive ? '#2f3a3e' : '#252b30',
                  color: '#ccc',
                  py: 1.15
                },
                '&:hover .close-btn': {
                  opacity: 1
                }
              }}
            >
              <TerminalIcon sx={{ fontSize: 16, opacity: 0.7 }} />
              <span style={{ fontSize: 13, whiteSpace: 'nowrap' }}>Shell {t.num}</span>
              <CloseIcon
                className="close-btn"
                sx={{
                  fontSize: 14,
                  opacity: isActive ? 0.6 : 0,
                  ml: 0.5,
                  borderRadius: '4px',
                  transition: 'all 0.15s ease',
                  '&:hover': {
                    opacity: 1,
                    bgcolor: 'rgba(255,255,255,0.1)'
                  }
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  closeTab(t.id);
                }}
              />
            </Box>
          );
        })}

        <IconButton
          size="small"
          onClick={addTab}
          disabled={tabs.length >= MAX_TABS}
          sx={{
            color: '#666',
            ml: 0.5,
            width: 28,
            height: 28,
            '&:hover': { color: '#aaa', bgcolor: 'rgba(255,255,255,0.05)' },
            '&.Mui-disabled': { color: '#444' }
          }}
        >
          <AddIcon sx={{ fontSize: 18 }} />
        </IconButton>
      </Box>

      {/* Terminal panels */}
      <Box sx={{ flex: 1, position: 'relative', bgcolor: '#1e1e1e' }}>
        {tabs.map((t) => {
          const isActive = activeId === t.id;
          return (
            <Box
              key={t.id}
              sx={{
                position: 'absolute',
                inset: 0,
                display: 'flex',
                visibility: isActive ? 'visible' : 'hidden',
                zIndex: isActive ? 1 : 0,
                pointerEvents: isActive ? 'auto' : 'none'
              }}
            >
              <IdefixTerminal />
            </Box>
          );
        })}
      </Box>
    </Box>
  );
}
