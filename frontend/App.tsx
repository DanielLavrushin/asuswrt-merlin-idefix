import { Box, ChipProps } from '@mui/material';
import { useEffect, useState } from 'react';
import IdefixTerminal from './IdefixTerminal';
import './App.css';
import idefixBg from './assets/idefix.png?inline';
import Version from './Version';
import engine from './modules/Engine';

const idefixHaterCookie = 'i-hate-dogs';
function App() {
  const [status, setStatus] = useState<'connected' | 'reconnecting' | 'offline'>('offline');
  const [showIdefix, setShowIdefix] = useState(false);

  const statusColor: ChipProps['color'] = {
    connected: 'success',
    reconnecting: 'warning',
    offline: 'error'
  }[status] as ChipProps['color'];

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
        <Box sx={{ flex: 1, height: '100%' }}>
          <IdefixTerminal onStatusChange={setStatus} />
        </Box>
        <Box
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
        {/* <Chip color={statusColor} size="small" label={status} sx={{ textTransform: 'capitalize', top: 2, right: 0 }} /> */}
      </Box>
    </>
  );
}

export default App;
