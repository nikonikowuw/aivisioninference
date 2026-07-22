import { useEffect, useRef, useState, useCallback } from 'react';
import { Box, Flex, Text, Tag, Button } from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { EdgeNodeTerminalClient } from 'services/edgeNodeTerminal';

interface TerminalProps {
  nodeId: string;
}

export default function Terminal({ nodeId }: TerminalProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const terminalRef = useRef<HTMLDivElement>(null);
  const clientRef = useRef<EdgeNodeTerminalClient | null>(null);
  const xtermRef = useRef<any>(null);
  const fitAddonRef = useRef<any>(null);
  const [connected, setConnected] = useState(false);
  const [statusText, setStatusText] = useState(t('terminal.connecting'));
  const [reconnectKey, setReconnectKey] = useState(0);

  const writeToTerminal = useCallback((data: string) => {
    if (xtermRef.current) {
      xtermRef.current.write(data);
    }
  }, []);

  useEffect(() => {
    if (!nodeId) return;

    let disposed = false;

    const initTerminal = async () => {
      try {
        const { Terminal } = await import('@xterm/xterm');
        const { FitAddon } = await import('@xterm/addon-fit');

        await import('@xterm/xterm/css/xterm.css');

        if (disposed || !terminalRef.current) return;

        const term = new Terminal({
          cursorBlink: true,
          cursorStyle: 'block',
          fontSize: 14,
          fontFamily: 'Menlo, Monaco, "Courier New", monospace',
          theme: {
            background: '#1a1b2e',
            foreground: '#e0e0e0',
            cursor: '#ffffff',
            selectionBackground: '#4a4a6a',
            black: '#2e2e2e',
            red: '#eb6a6a',
            green: '#6aeb6a',
            yellow: '#ebeb6a',
            blue: '#6a6aeb',
            magenta: '#eb6aeb',
            cyan: '#6aebeb',
            white: '#e0e0e0',
            brightBlack: '#555555',
            brightRed: '#ff6a6a',
            brightGreen: '#6aff6a',
            brightYellow: '#ffff6a',
            brightBlue: '#6a6aff',
            brightMagenta: '#ff6aff',
            brightCyan: '#6affff',
            brightWhite: '#ffffff',
          },
          allowTransparency: true,
          rows: 24,
          cols: 80,
        });

        xtermRef.current = term;

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);
        fitAddonRef.current = fitAddon;

        term.open(terminalRef.current);
        fitAddon.fit();

        // Create terminal client
        const client = new EdgeNodeTerminalClient(nodeId, {
          onOutput: (data) => {
            if (!disposed) term.write(data);
          },
          onError: (error) => {
            if (!disposed) {
              term.writeln(`\r\n\x1b[31m[Error] ${error}\x1b[0m`);
            }
          },
          onSessionOpen: () => {
            if (!disposed) {
              setConnected(true);
              setStatusText(t('terminal.connected'));
              term.write(`\r\n\x1b[32m${t('terminal.sessionOpened')}\x1b[0m\r\n`);
            }
          },
          onSessionClose: () => {
            if (!disposed) {
              setConnected(false);
              setStatusText(t('terminal.disconnected'));
              term.write(`\r\n\x1b[33m${t('terminal.sessionClosed')}\x1b[0m\r\n`);
            }
          },
        });

        clientRef.current = client;

        // Handle user input
        term.onData((data: string) => {
          client.sendInput(data);
        });

        // Connect
        client.connect();

        // Handle resize
        const handleResize = () => {
          try {
            fitAddon.fit();
            const dims = fitAddon.proposeDimensions();
            if (dims && client.isConnected()) {
              client.sendResize(dims.cols, dims.rows);
            }
          } catch {}
        };

        window.addEventListener('resize', handleResize);

        // Ping every 30 seconds
        const pingInterval = setInterval(() => {
          if (client.isConnected()) {
            client.sendPing();
          }
        }, 30000);

        // Store cleanup
        return () => {
          window.removeEventListener('resize', handleResize);
          clearInterval(pingInterval);
        };
      } catch (e) {
        console.error('Failed to initialize terminal:', e);
        if (!disposed) {
          setStatusText(t('terminal.initFailed'));
        }
      }
    };

    const cleanupPromise = initTerminal();

    return () => {
      disposed = true;
      cleanupPromise.then((cleanup) => {
        if (cleanup) cleanup();
        if (clientRef.current) {
          clientRef.current.disconnect();
          clientRef.current = null;
        }
        if (xtermRef.current) {
          xtermRef.current.dispose();
          xtermRef.current = null;
        }
        fitAddonRef.current = null;
      });
    };
  }, [nodeId, t, reconnectKey]);

  const handleReconnect = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
    }
    if (xtermRef.current) {
      xtermRef.current.clear();
    }
    setStatusText(t('terminal.reconnecting'));
    // Reconnect by re-running the useEffect via a key counter.
    setReconnectKey((k) => k + 1);
  }, [t]);

  return (
    <Box>
      <Flex align="center" mb="10px" gap={2}>
        <Tag colorScheme={connected ? 'green' : 'gray'} size="sm">
          {connected ? t('terminal.connected') : t('terminal.disconnected')}
        </Tag>
        <Text fontSize="sm" color="gray.500">
          {statusText}
        </Text>
        <Box flex={1} />
        {!connected && (
          <Button size="xs" onClick={handleReconnect}>
            {t('terminal.reconnect')}
          </Button>
        )}
      </Flex>
      <Box
        ref={terminalRef}
        bg="#1a1b2e"
        borderRadius="md"
        overflow="hidden"
        sx={{
          '.xterm': {
            padding: '8px',
            height: '100%',
          },
          '.xterm-viewport': {
            scrollbarWidth: 'thin',
          },
        }}
        height="500px"
      />
    </Box>
  );
}
