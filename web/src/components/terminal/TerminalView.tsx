import { useEffect, useRef, useState } from 'react';
import { Box, Flex, Button, Text, Spinner, useColorModeValue } from '@chakra-ui/react';
import { useTerminal } from 'hooks/useTerminal';

interface TerminalViewProps {
  nodeId: string;
  sessionId?: string;
  onSessionChange?: (sessionId: string) => void;
  onClose?: () => void;
}

export default function TerminalView({ nodeId, sessionId: initialSessionId, onSessionChange, onClose }: TerminalViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [terminalReady, setTerminalReady] = useState(false);
  const bgColor = useColorModeValue('white', 'gray.800');

  const {
    status,
    sessionId,
    initTerminal,
    connect,
    disconnect,
  } = useTerminal({
    nodeId,
    sessionId: initialSessionId,
    onSessionChange,
  });

  // Initialize terminal renderer once
  useEffect(() => {
    if (!containerRef.current) return;
    const cleanup = initTerminal(containerRef.current);
    setTerminalReady(true);
    return () => {
      cleanup?.();
      setTerminalReady(false);
    };
  }, [initTerminal]);

  // Auto-connect when terminal is ready
  useEffect(() => {
    if (terminalReady) {
      connect(initialSessionId);
    }
  }, [terminalReady]); // eslint-disable-line react-hooks/exhaustive-deps

  const statusColor = {
    disconnected: 'gray',
    connecting: 'blue',
    connected: 'green',
    paused: 'orange',
    closed: 'red',
  }[status];

  const statusText = {
    disconnected: '未连接',
    connecting: '连接中...',
    connected: '已连接',
    paused: '已暂停',
    closed: '已关闭',
  }[status];

  return (
    <Box>
      <Flex mb={2} align="center" justify="space-between">
        <Flex align="center" gap={2}>
          <Box w={2} h={2} borderRadius="full" bg={`${statusColor}.400`} />
          <Text fontSize="sm" color={`${statusColor}.400`}>{statusText}</Text>
          {sessionId && (
            <Text fontSize="xs" color="gray.400">({sessionId.slice(0, 8)})</Text>
          )}
        </Flex>
        <Flex gap={2}>
          {status === 'paused' ? (
            <Button size="xs" colorScheme="blue" onClick={() => connect()} >
              恢复连接
            </Button>
          ) : status === 'connected' ? (
            <Button size="xs" variant="outline" onClick={disconnect}>
              断开
            </Button>
          ) : null}
          <Button size="xs" variant="ghost" onClick={onClose}>关闭</Button>
        </Flex>
      </Flex>
      <Box
        ref={containerRef}
        h="400px"
        bg="#1a1b2e"
        borderRadius="md"
        overflow="hidden"
        position="relative"
      >
        {status === 'connecting' && (
          <Flex position="absolute" inset={0} align="center" justify="center" bg="rgba(26,27,46,0.8)" zIndex={1}>
            <Spinner color="blue.400" />
          </Flex>
        )}
      </Box>
    </Box>
  );
}
