import { Box } from '@mui/material';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import TerminalTabs from './TerminalTabs';
import './App.css';
import idefixBg from './assets/idefix.png?inline';
import Version from './Version';
import engine from './modules/Engine';

const idefixHaterCookie = 'i-hate-dogs';
const MIN_SHELL_HEIGHT = 320;
const SHELL_BOTTOM_GAP = 16;

function App() {
  const [showIdefix, setShowIdefix] = useState(false);
  const shellRef = useRef<HTMLDivElement | null>(null);
  const footerRef = useRef<HTMLDivElement | null>(null);
  const [shellHeight, setShellHeight] = useState(MIN_SHELL_HEIGHT);

  useLayoutEffect(() => {
    const measure = () => {
      const shell = shellRef.current;
      if (!shell) return;
      const top = shell.getBoundingClientRect().top + window.scrollY;
      const footer = footerRef.current?.offsetHeight ?? 0;
      const pageFooter = document.getElementById('footer')?.offsetHeight ?? 0;
      const available = window.innerHeight - top - footer - pageFooter - SHELL_BOTTOM_GAP;
      setShellHeight(Math.max(MIN_SHELL_HEIGHT, Math.round(available)));
    };

    measure();
    window.addEventListener('resize', measure);

    let observer: ResizeObserver | undefined;
    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(measure);
      observer.observe(document.body);
      if (footerRef.current) observer.observe(footerRef.current);
    }

    return () => {
      window.removeEventListener('resize', measure);
      observer?.disconnect();
    };
  }, [showIdefix]);

  useEffect(() => {
    const cookie = engine.getCookie(idefixHaterCookie);
    if (cookie && cookie === 'true') {
      setShowIdefix(false);
    } else {
      setShowIdefix(true);
    }
  }, []);

  return (
    <>
      <Box
        sx={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column'
        }}
      >
        <Box className="formfonttitle" sx={{ p: 0, pt: 4, pl: 1 }}>
          🐾 IDEFIX Terminal
        </Box>
        <Box sx={{ m: 1, mt: 0, mb: 1.5 }} className="splitLine"></Box>
        <Box ref={shellRef} sx={{ height: shellHeight, minHeight: 0 }}>
          <TerminalTabs />
        </Box>
        <Box
          ref={footerRef}
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-end'
          }}
        >
          {showIdefix && (
            <Box
              component="img"
              src={idefixBg}
              alt=""
              onClick={() => {
                if (
                  confirm(`Idefix is a fictional character from the Asterix comic series, known for his loyalty and bravery. He is a small dog who accompanies the main characters on their adventures same as your journey with SSH Terminal. 
Do you really want to hide this cute dog?`)
                ) {
                  engine.setCookie(idefixHaterCookie, 'true');
                  setShowIdefix(false);
                }
              }}
              sx={{
                cursor: 'pointer',
                width: 140,
                opacity: 0.85
              }}
            />
          )}
          <Box sx={{ ml: 'auto' }}>
            <Version />
          </Box>
        </Box>
      </Box>
    </>
  );
}

export default App;
