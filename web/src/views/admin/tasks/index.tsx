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
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { tasksApi, type Task } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';

const statusColor: Record<string, string> = {
  pending: 'yellow',
  running: 'blue',
  completed: 'green',
  failed: 'red',
  cancelled: 'gray',
};

export default function Tasks() {
  const { t } = useTranslation('modules/tasks');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [cancelTarget, setCancelTarget] = useState<string | null>(null);
  const [isCancelling, setIsCancelling] = useState(false);

  const loadTasks = useCallback(async () => {
    try {
      const data = await tasksApi.list({ page: 1, page_size: 100 });
      setTasks(data.list);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    }
  }, [toast, t]);

  useEffect(() => {
    loadTasks().finally(() => setLoading(false));
  }, [loadTasks]);

  const handleCancel = async () => {
    if (!cancelTarget) return;
    setIsCancelling(true);
    try {
      await tasksApi.cancel(cancelTarget);
      toast({ title: t('message.cancelled'), status: 'success' });
      loadTasks();
    } catch (err) {
      toast({ title: t('message.cancelFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsCancelling(false);
      setCancelTarget(null);
    }
  };

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>{t('table.columns.id')}</Th>
              <Th>{t('table.columns.type')}</Th>
              <Th>{t('table.columns.status')}</Th>
              <Th>{t('table.columns.error')}</Th>
              <Th>{t('table.columns.createdAt')}</Th>
              <Th>{t('table.columns.updatedAt')}</Th>
              <Th>{t('table.columns.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {tasks.map((task) => (
              <Tr key={task.id}>
                <Td>{task.id}</Td>
                <Td fontWeight="600">{task.type}</Td>
                <Td>
                  <Badge colorScheme={statusColor[task.status] || 'gray'}>{t(`table.status.${task.status}`)}</Badge>
                </Td>
                <Td maxW="200px" isTruncated color="red.400">{task.error || '-'}</Td>
                <Td whiteSpace="nowrap">{formatDateTime(task.created_at)}</Td>
                <Td whiteSpace="nowrap">{formatDateTime(task.updated_at)}</Td>
                <Td>
                  {(task.status === 'pending' || task.status === 'running') && (
                    <HStack spacing={2}>
                      <IconButton
                        aria-label={t('actions.cancel')}
                        icon={<CloseIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => setCancelTarget(task.id)}
                      />
                    </HStack>
                  )}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
      <ConfirmDialog
        isOpen={cancelTarget !== null}
        onClose={() => setCancelTarget(null)}
        onConfirm={handleCancel}
        title={t('actions.cancel')}
        message={t('message.cancelConfirm')}
        confirmText={t('message.confirmCancel')}
        isLoading={isCancelling}
      />
    </Box>
  );
}
