import {
  Box,
  Flex,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  useColorModeValue,
  IconButton,
  useToast,
  HStack,
  Spinner,
  Center,
  Badge,
} from '@chakra-ui/react';
import { CloseIcon } from '@chakra-ui/icons';
import { useEffect, useState, useCallback } from 'react';
import { tasksApi, type Task } from 'services/api';

const statusColor: Record<string, string> = {
  pending: 'yellow',
  running: 'blue',
  completed: 'green',
  failed: 'red',
  cancelled: 'gray',
};

export default function Tasks() {
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  const loadTasks = useCallback(async () => {
    try {
      const data = await tasksApi.list({ page: 1, page_size: 100 });
      setTasks(data.list);
    } catch {}
  }, []);

  useEffect(() => {
    loadTasks().finally(() => setLoading(false));
  }, [loadTasks]);

  const handleCancel = async (id: string) => {
    if (!confirm('确定取消该任务？')) return;
    try {
      await tasksApi.cancel(id);
      toast({ title: '已取消', status: 'success' });
      loadTasks();
    } catch (err) {
      toast({ title: '取消失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>任务管理</Text>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr><Th>ID</Th><Th>类型</Th><Th>状态</Th><Th>错误</Th><Th>创建时间</Th><Th>更新时间</Th><Th>操作</Th></Tr>
          </Thead>
          <Tbody>
            {tasks.map((t) => (
              <Tr key={t.id}>
                <Td>{t.id}</Td>
                <Td fontWeight="600">{t.type}</Td>
                <Td>
                  <Badge colorScheme={statusColor[t.status] || 'gray'}>{t.status}</Badge>
                </Td>
                <Td maxW="200px" isTruncated color="red.400">{t.error || '-'}</Td>
                <Td whiteSpace="nowrap">{new Date(t.created_at).toLocaleString('zh-CN')}</Td>
                <Td whiteSpace="nowrap">{new Date(t.updated_at).toLocaleString('zh-CN')}</Td>
                <Td>
                  {(t.status === 'pending' || t.status === 'running') && (
                    <HStack spacing={2}>
                      <IconButton
                        aria-label="取消任务"
                        icon={<CloseIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => handleCancel(t.id)}
                      />
                    </HStack>
                  )}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </Box>
  );
}
