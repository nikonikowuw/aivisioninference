import { Box, Text, Button, Flex, HStack, VStack, Spinner, useToast } from '@chakra-ui/react';
import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import TerminalView from 'components/terminal/TerminalView';
import { terminalApi } from 'services/terminal';
import type { EdgeNode } from 'services/edgeNode';

interface TerminalTabProps {
  node: EdgeNode;
}

export default function TerminalTab({ node }: TerminalTabProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const toast = useToast();
  const navigate = useNavigate();
  const [showTerminal, setShowTerminal] = useState(false);
  const [sessions, setSessions] = useState<{ id: string; started_at: string }[]>([]);
  const [showSessions, setShowSessions] = useState(false);
  const [resumingSession, setResumingSession] = useState<string | null>(null);

  const canConnect = node.status === 'online';

  const handleOpen = useCallback(() => {
    setShowTerminal(true);
  }, []);

  const handleClose = useCallback(() => {
    setShowTerminal(false);
  }, []);

  const handleListSessions = useCallback(async () => {
    try {
      const data = await terminalApi.listSessions(node.id, { status: 'paused' });
      const list = Array.isArray(data) ? data : [];
      setSessions(list);
      setShowSessions(!showSessions);
    } catch {
      toast({ title: '获取会话列表失败', status: 'error' });
    }
  }, [node.id, showSessions, toast]);

  const handleResume = useCallback(async (sessionId: string) => {
    setResumingSession(sessionId);
    setShowTerminal(true);
  }, []);

  if (!canConnect && !showTerminal) {
    return (
      <Box py={8} textAlign="center">
        <Text color="gray.400" mb={4}>
          {node.status !== 'online'
            ? '节点不在线，无法打开终端'
            : '节点未配置 SSH 连接'}
        </Text>
        {node.status === 'online' && (
          <Text fontSize="sm" color="gray.500">
            请在节点编辑页配置 SSH 私钥和端口
          </Text>
        )}
      </Box>
    );
  }

  if (showTerminal) {
    return (
      <TerminalView
        nodeId={node.id}
        sessionId={resumingSession || undefined}
        onClose={() => {
          setShowTerminal(false);
          setResumingSession(null);
        }}
        onSessionChange={(sid) => {
          setResumingSession(null);
        }}
      />
    );
  }

  return (
    <Box py={4}>
      <VStack spacing={4} align="stretch">
        <Button colorScheme="blue" onClick={handleOpen}>
          打开新终端
        </Button>
        <Button variant="outline" onClick={handleListSessions}>
          {showSessions ? '收起会话列表' : '恢复已有会话'}
        </Button>
        {showSessions && (
          <Box>
            {sessions.length === 0 ? (
              <Text fontSize="sm" color="gray.400">没有可恢复的会话</Text>
            ) : (
              <VStack spacing={2} align="stretch">
                {sessions.map((s) => (
                  <Flex
                    key={s.id}
                    align="center"
                    justify="space-between"
                    px={3}
                    py={2}
                    borderRadius="md"
                    cursor="pointer"
                    _hover={{ bg: 'whiteAlpha.200' }}
                    onClick={() => handleResume(s.id)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') handleResume(s.id); }}
                  >
                    <HStack spacing={2}>
                      {resumingSession === s.id && <Spinner size="sm" />}
                      <Text fontSize="sm">
                        会话 {s.id.slice(0, 8)} — {new Date(s.started_at).toLocaleString()}
                      </Text>
                    </HStack>
                    <Button
                      size="xs"
                      variant="link"
                      colorScheme="blue"
                      onClick={(e) => {
                        e.stopPropagation();
                        navigate(`/admin/devices/edge-nodes/${node.id}/playback/${s.id}`);
                      }}
                    >
                      查看回放
                    </Button>
                  </Flex>
                ))}
              </VStack>
            )}
          </Box>
        )}
      </VStack>
    </Box>
  );
}
