import React, { useEffect, useLayoutEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import "@xterm/xterm/css/xterm.css";

export interface TerminalProps {
  endpoint?: string;
  protocol?: string;
  cols?: number;
  rows?: number;
}

const secure = window.location.protocol === "https:";

export const IdefixTerminal: React.FC<TerminalProps> = ({
  endpoint = `${secure ? "wss" : "ws"}://${window.location.hostname}:8787/ws`,
  protocol = "idefix",
  cols = 85,
  rows = 50,
}) => {
  const containerRef = useRef<HTMLDivElement | null>(null);

  useLayoutEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      fontFamily: "Consolas, monospace",
      fontSize: 14,
      cols,
      rows,
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    term.open(containerRef.current);
    requestAnimationFrame(() => fitAddon.fit());

    const socket = new WebSocket(endpoint, protocol);
    socket.binaryType = "arraybuffer";
    const attachAddon = new AttachAddon(socket, {
      bidirectional: true,
    });
    term.loadAddon(attachAddon);

    const handleResize = () => fitAddon.fit();
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      socket.readyState === WebSocket.OPEN && socket.close();
      term.dispose();
    };
  }, [endpoint, protocol, cols, rows]);

  return (
    <div
      ref={containerRef}
      style={{ width: "100%", height: "100%", overflow: "hidden" }}
    />
  );
};

export default IdefixTerminal;
