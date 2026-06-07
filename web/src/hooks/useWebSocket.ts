import { useEffect, useRef } from 'react';
import { useToast } from '@chakra-ui/react';
import { getAccessToken } from 'services/api';

interface UseWebSocketOptions {
  onMessage: (msg: any) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: () => void;
  urlPath?: string;
}

export function useWebSocket({ onMessage, onOpen, onClose, onError, urlPath = '/api/v1/ws' }: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const unmountedRef = useRef(false);
  const retryTimerRef = useRef<ReturnType<typeof setTimeout>>();
  const retryDelayRef = useRef(1000);
  const maxRetryDelay = 30000;

  useEffect(() => {
    unmountedRef.current = false;

    const connect = () => {
      if (unmountedRef.current) return;
      const token = getAccessToken();
      if (!token) return;
      
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}${urlPath}?token=${encodeURIComponent(token)}`;
      
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        retryDelayRef.current = 1000;
        if (onOpen) onOpen();
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          onMessage(msg);
        } catch { /* ignore non-JSON */ }
      };

      ws.onclose = () => {
        if (onClose) onClose();
        if (!unmountedRef.current) {
          retryTimerRef.current = setTimeout(connect, retryDelayRef.current);
          retryDelayRef.current = Math.min(retryDelayRef.current * 2, maxRetryDelay);
        }
      };

      ws.onerror = () => {
        if (onError) onError();
        ws.close();
      };
    };

    connect();

    return () => {
      unmountedRef.current = true;
      if (retryTimerRef.current) clearTimeout(retryTimerRef.current);
      if (wsRef.current) wsRef.current.close();
    };
  }, [onMessage, onOpen, onClose, onError, urlPath]);

  return { ws: wsRef.current };
}
