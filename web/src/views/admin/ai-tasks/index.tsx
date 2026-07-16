import {
  Box,
  Button,
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
  HStack,
  Center,
  Spinner,
  Badge,
  useToast,
  Tooltip,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon, RepeatIcon, WarningIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { aiVisionTasksApi, aiTimeSchedulesApi, type AIVisionTask, type AITimeSchedule } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { useWebSocket } from 'hooks/useWebSocket';
import { WS_TOPIC } from 'constants/websocket';
import TaskFormModal from './components/TaskFormModal';

const statusColor: Record<string, string> = {
  draft: 'gray',
  ready: 'cyan',
  running: 'green',
  suspended: 'yellow',
  stopped: 'red',
  error: 'red',
};

export default function AIVisionTasks() {
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { t, i18n } = useTranslation(['modules/ai-tasks', 'common']);

  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [restartingID, setRestartingID] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<AIVisionTask | null>(null);
  const [scheduleMap, setScheduleMap] = useState<Record<string, AITimeSchedule>>({});

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const formatErrorReason = useCallback((reason?: string) => {
    if (!reason) return '-';
    if (reason.startsWith('errors.')) {
      return i18n.exists(reason, { ns: 'modules/ai-tasks' }) ? t(reason) : t('errors.unknown');
    }
    return reason;
  }, [i18n, t]);

  // 加载时间配置列表用于名称映射
  useEffect(() => {
    aiTimeSchedulesApi.listAll().then(res => {
      const map: Record<string, AITimeSchedule> = {};
      res.forEach(s => { map[s.id] = s; });
      setScheduleMap(map);
    }).catch(() => {});
  }, [searchTrigger]);

  const fetchTasks = useCallback(
    (p: number, ps: number) =>
      aiVisionTasksApi.list({ page: p, page_size: ps, ...filters }),
    [filters],
  );

  const {
    list: tasks,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
  } = usePagination<AIVisionTask>(fetchTasks);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  // WebSocket real-time task status updates
  const handleWsMessage = useCallback((msg: any) => {
    if (msg.type === WS_TOPIC.TASK_STATUS) {
      load({ page });
    }
  }, [load, page]);

  useWebSocket({ onMessage: handleWsMessage });

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await aiVisionTasksApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      load();
    } catch (err: any) {
      toast({ title: t('message.deleteFailed'), description: err?.message || '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  const handleRestart = async (task: AIVisionTask) => {
    setRestartingID(task.id);
    try {
      await aiVisionTasksApi.restart(task.id);
      toast({ title: t('message.restartSuccess'), status: 'success' });
      load();
    } catch (err: any) {
      toast({ title: t('message.restartFailed'), description: err?.message || '', status: 'error' });
    } finally {
      setRestartingID(null);
    }
  };

  if (initialLoading) {
    return (
      <Center h="400px">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>
          {t('title')}
        </Text>
        <Button
          leftIcon={<AddIcon />}
          colorScheme="brand"
          onClick={() => {
            setEditingTask(null);
            setFormOpen(true);
          }}
        >
          {t('actions.create')}
        </Button>
      </Flex>

      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          {
            name: 'status',
            label: t('fields.status'),
            options: [
              { value: 'draft', label: t('status.draft') },
              { value: 'ready', label: t('status.ready') },
              { value: 'running', label: t('status.running') },
              { value: 'error', label: t('status.error') },
            ],
          },
        ]}
      />

      <Box
        bg={bgCard}
        borderRadius="16px"
        border="1px solid"
        borderColor={borderColor}
        overflow="auto"
      >
        <Table variant="simple" size="md" minW="800px">
          <Thead>
            <Tr>
              <Th>{t('fields.name')}</Th>
              <Th>{t('fields.status')}</Th>
              <Th>{t('fields.schedule')}</Th>
              <Th>{t('fields.dateRange')}</Th>
              <Th>{t('fields.errorReason')}</Th>
              <Th>{t('fields.updatedAt')}</Th>
              <Th>{t('common:button.edit', { defaultValue: '操作' })}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {tasks.map(task => {
              const schedule = scheduleMap[task.schedule_id];
              const errorReason = formatErrorReason(task.error_reason);
              return (
                <Tr key={task.id}>
                  <Td fontWeight="600">{task.name}</Td>
                  <Td>
                    <Badge colorScheme={statusColor[task.status] || 'gray'}>
                      {task.status === 'suspended' && (
                        <WarningIcon mr={1} />
                      )}
                      {t(`status.${task.status}`)}
                    </Badge>
                  </Td>
                  <Td>
                    {schedule ? (
                      <Tooltip label={schedule.description}>
                        <Badge colorScheme="purple">{schedule.name}</Badge>
                      </Tooltip>
                    ) : (
                      <Text color="gray.400" fontSize="sm">-</Text>
                    )}
                  </Td>
                  <Td whiteSpace="nowrap">
                    {task.start_date?.substring(0, 10)} ~ {task.end_date?.substring(0, 10)}
                  </Td>
                  <Td maxW="200px" isTruncated color="red.400">
                    <Tooltip label={errorReason}>{errorReason}</Tooltip>
                  </Td>
                  <Td whiteSpace="nowrap">{formatDateTime(task.updated_at)}</Td>
                  <Td>
                    <HStack spacing={2}>
                      <IconButton
                        aria-label={t('actions.edit')}
                        icon={<EditIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="blue"
                        onClick={() => {
                          setEditingTask(task);
                          setFormOpen(true);
                        }}
                      />
                      <IconButton
                        aria-label={t('actions.restart')}
                        icon={<RepeatIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="green"
                        isLoading={restartingID === task.id}
                        onClick={() => handleRestart(task)}
                      />
                      <IconButton
                        aria-label={t('actions.delete')}
                        icon={<DeleteIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => setDeleteTarget(task.id)}
                      />
                    </HStack>
                  </Td>
                </Tr>
              );
            })}
            {tasks.length === 0 && (
              <Tr>
                <Td colSpan={7} textAlign="center" py={10} color="gray.500">
                  {t('message.empty')}
                </Td>
              </Tr>
            )}
          </Tbody>
        </Table>
        <Pagination
          page={page}
          pageSize={pageSize}
          total={total}
          onChange={changePage}
          onPageSizeChange={changePageSize}
          isLoading={pageLoading}
        />
      </Box>

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title={t('actions.delete')}
        message={t('message.deleteConfirm')}
        confirmText={t('actions.delete')}
        isLoading={isDeleting}
      />

      <TaskFormModal
        isOpen={formOpen}
        onClose={() => setFormOpen(false)}
        onSuccess={() => {
          setFormOpen(false);
          load();
        }}
        initialData={editingTask}
      />
    </Box>
  );
}
