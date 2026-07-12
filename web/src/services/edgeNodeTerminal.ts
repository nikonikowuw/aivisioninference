import { getAccessToken } from './api';

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

export interface TerminalMessage {
  type: string;
  session_id?: string;
  data?: string;
  cols?: number;
  rows?: number;
  error?: string;
}

export interface TerminalCallbacks {
  onOutput: (data: string) => void;
  onError: (error: string) => void;
  onSessionOpen?: (sessionId: string) => void;
  onSessionClose?: () => void;
}

export class EdgeNodeTerminalClient {
  private ws: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private nodeId: string;
  private callbacks: TerminalCallbacks;
  private connected = false;
  private maxReconnectAttempts = 3;
  private reconnectAttempts = 0;

  constructor(nodeId: string, callbacks: TerminalCallbacks) {
    this.nodeId = nodeId;
    this.callbacks = callbacks;
  }

  /** Connect to the terminal WebSocket */
  connect(): void {
    if (this.ws && this.connected) return;

    const token = getAccessToken();
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const apiUrl = BASE_URL || `${protocol}://${window.location.host}/api/v1`;
    const wsUrl = `${apiUrl.replace(/^http/, 'ws')}/edge-nodes/${this.nodeId}/terminal`;

    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      this.connected = true;
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const msg: TerminalMessage = JSON.parse(event.data);
        switch (msg.type) {
          case 'session':
            if (msg.session_id && this.callbacks.onSessionOpen) {
              this.callbacks.onSessionOpen(msg.session_id);
            }
            break;
          case 'output':
            if (msg.data && this.callbacks.onOutput) {
              this.callbacks.onOutput(msg.data);
            }
            break;
          case 'error':
            if (msg.error && this.callbacks.onError) {
              this.callbacks.onError(msg.error);
            }
            break;
          case 'pong':
            // Keep-alive response
            break;
        }
      } catch (e) {
        console.error('[TerminalClient] Failed to parse message:', e);
      }
    };

    this.ws.onclose = () => {
      this.connected = false;
      if (this.callbacks.onSessionClose) {
        this.callbacks.onSessionClose();
      }
      this.attemptReconnect();
    };

    this.ws.onerror = (e) => {
      console.error('[TerminalClient] WebSocket error:', e);
    };
  }

  /** Send terminal input to the server */
  sendInput(data: string): void {
    if (!this.ws || !this.connected) return;
    const msg: TerminalMessage = {
      type: 'input',
      data,
    };
    this.ws.send(JSON.stringify(msg));
  }

  /** Send resize event */
  sendResize(cols: number, rows: number): void {
    if (!this.ws || !this.connected) return;
    const msg: TerminalMessage = {
      type: 'resize',
      cols,
      rows,
    };
    this.ws.send(JSON.stringify(msg));
  }

  /** Send keep-alive ping */
  sendPing(): void {
    if (!this.ws || !this.connected) return;
    this.ws.send(JSON.stringify({ type: 'ping' }));
  }

  /** Disconnect and clean up */
  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.reconnectAttempts = this.maxReconnectAttempts; // prevent reconnect
    if (this.ws) {
      this.ws.send(JSON.stringify({ type: 'close' }));
      this.ws.close();
      this.ws = null;
    }
    this.connected = false;
  }

  /** Check if connected */
  isConnected(): boolean {
    return this.connected;
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;
    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 10000);
    this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, delay);
  }
}
