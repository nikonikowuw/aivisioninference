import { useEffect, useRef, useCallback, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';

interface TerminalMessage {
  type: string;
  data?: string;
  cols?: number;
  rows?: number;
  reason?: string;
  session_id?: string;
}

interface UseTerminalOptions {
  nodeId: string;
  sessionId?: string;
  onStatusChange?: (status: 'disconnected' | 'connecting' | 'connected' | 'paused' | 'closed', reason?: string) => void;
  onSessionChange?: (sessionId: string) => void;
}

export function useTerminal({ nodeId, sessionId: initialSessionId, onStatusChange, onSessionChange }: UseTerminalOptions) {
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [status, setStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'paused' | 'closed'>('disconnected');
  const [sessionId, setSessionId] = useState<string | undefined>(initialSessionId);

  const updateStatus = useCallback((newStatus: 'disconnected' | 'connecting' | 'connected' | 'paused' | 'closed', reason?: string) => {
    setStatus(newStatus);
    onStatusChange?.(newStatus, reason);
  }, [onStatusChange]);

  const getWebSocketURL = useCallback((sid?: string) => {
    const token = (window as any).__ACCESS_TOKEN__ || '';
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    let url = `${protocol}//${window.location.host}/api/v1/ws/terminal?token=${encodeURIComponent(token)}&node_id=${encodeURIComponent(nodeId)}&cols=80&rows=24`;
    if (sid) url += `&session_id=${encodeURIComponent(sid)}`;
    return url;
  }, [nodeId]);

  const connect = useCallback((sid?: string) => {
    const term = terminalRef.current;
    if (!term) return;

    const targetSessionId = sid || sessionId;
    updateStatus('connecting');

    const ws = new WebSocket(getWebSocketURL(targetSessionId));
    wsRef.current = ws;

    ws.onopen = () => {
      updateStatus('connected');
      term.reset();
      fitAddonRef.current?.fit();
    };

    ws.onmessage = (event) => {
      try {
        const msg: TerminalMessage = JSON.parse(event.data);
        switch (msg.type) {
          case 'output':
            if (msg.data) term.write(msg.data);
            break;
          case 'paused':
            updateStatus('paused', msg.reason);
            break;
          case 'resumed':
            updateStatus('connected');
            break;
          case 'closed':
            updateStatus('closed', msg.reason);
            break;
          case 'resize':
            if (msg.cols && msg.rows) term.resize(msg.cols, msg.rows);
            break;
          case 'pong':
            break;
        }
      } catch { /* ignore */ }
    };

    ws.onclose = () => {
      updateStatus('paused', 'disconnected');
    };

    ws.onerror = () => {
      ws.close();
    };

    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'input', data }));
      }
    });
  }, [nodeId, sessionId, getWebSocketURL, updateStatus]);

  const disconnect = useCallback(() => {
    wsRef.current?.close();
    wsRef.current = null;
    updateStatus('disconnected');
  }, [updateStatus]);

  const sendResize = useCallback((cols: number, rows: number) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'resize', cols, rows }));
    }
  }, []);

  const initTerminal = useCallback((container: HTMLDivElement) => {
    const term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'block',
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1a1b2e',
        foreground: '#cdd6f4',
        cursor: '#f5e0dc',
        selectionBackground: '#585b70',
        black: '#45475a',
        red: '#f38ba8',
        green: '#a6e3a1',
        yellow: '#f9e2af',
        blue: '#89b4fa',
        magenta: '#f5c2e7',
        cyan: '#94e2d5',
        white: '#bac2de',
        brightBlack: '#585b70',
        brightRed: '#f38ba8',
        brightGreen: '#a6e3a1',
        brightYellow: '#f9e2af',
        brightBlue: '#89b4fa',
        brightMagenta: '#f5c2e7',
        brightCyan: '#94e2d5',
        brightWhite: '#a6adc8',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    fitAddonRef.current = fitAddon;

    term.open(container);
    fitAddon.fit();

    terminalRef.current = term;

    // Handle resize
    const handleResize = () => {
      fitAddon.fit();
      if (term.cols && term.rows) {
        sendResize(term.cols, term.rows);
      }
    };
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      term.dispose();
      terminalRef.current = null;
    };
  }, [sendResize]);

  return {
    status,
    sessionId,
    terminalRef,
    initTerminal,
    connect,
    disconnect,
    sendResize,
  };
}
