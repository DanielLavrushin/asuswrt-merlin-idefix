import { Box, CssBaseline, Typography } from "@mui/material";
import React from "react";
import { useEffect } from "react";
import IdefixTerminal from "./IdefixTerminal";
import "./App.css";

function App() {
  useEffect(() => {}, []);

  return (
    <>
      <Box
        sx={{
          height: "100%",
          width: "100%",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <Typography variant="h5" sx={{ p: 2 }}>
          Idefix Terminal
        </Typography>
        <IdefixTerminal />
      </Box>
    </>
  );
}

export default App;
