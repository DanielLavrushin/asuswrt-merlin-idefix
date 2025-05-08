import { Box, CssBaseline } from "@mui/material";
import React from "react";
import { useEffect } from "react";
import IdefixTerminal from "./IdefixTerminal";

function App() {
  useEffect(() => {}, []);

  return (
    <>
      <Box
        sx={{
          height: "100%",
          width: "100%",
          display: "flex",
        }}
      >
        <IdefixTerminal />
      </Box>
    </>
  );
}

export default App;
