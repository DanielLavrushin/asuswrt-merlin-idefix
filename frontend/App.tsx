import { AppBar, Box, Chip, ChipProps, Typography } from '@mui/material';
import React, { useState } from 'react';
import IdefixTerminal from './IdefixTerminal';
import './App.css';
import idefixBg from './assets/idefix.png?inline';
import Version from './Version';
function App() {
  const [status, setStatus] = useState<'connected' | 'reconnecting' | 'offline'>('offline');

  const statusColor: ChipProps['color'] = {
    connected: 'success',
    reconnecting: 'warning',
    offline: 'error'
  }[status] as ChipProps['color'];

  return (
    <>
      <Box
        sx={{
          height: '100vh',
          width: '100%',
          display: 'flex',
          flexDirection: 'column'
        }}
      >
        <Box className="formfonttitle" sx={{ p: 0, pt: 4, pl: 1 }}>
          IDEFIX Terminal
        </Box>
        <Box sx={{ m: 1, mt: 0, mb: 1.5 }} className="splitLine"></Box>

        <IdefixTerminal onStatusChange={setStatus} />
        <Box
          component="img"
          src={idefixBg}
          alt=""
          sx={{
            position: 'absolute',
            bottom: 0,
            left: 0,
            width: 140,
            pointerEvents: 'none',
            userSelect: 'none',
            opacity: 0.85
          }}
        />
        <Version />
        {/* <Chip color={statusColor} size="small" label={status} sx={{ textTransform: 'capitalize', top: 2, right: 0 }} /> */}
      </Box>
    </>
  );
}

export default App;
