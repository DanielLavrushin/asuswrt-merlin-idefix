import { Box, CssBaseline, Typography } from '@mui/material';
import React from 'react';
import { useEffect } from 'react';
import IdefixTerminal from './IdefixTerminal';
import './App.css';
import idefixBg from './assets/idefix.png?inline';

function App() {
  useEffect(() => {}, []);

  return (
    <Box
      sx={{
        height: '100%',
        width: '100%',
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      <Typography variant="h5" sx={{ p: 2 }}>
        Idefix Terminal
      </Typography>
      <IdefixTerminal />
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
    </Box>
  );
}

export default App;
