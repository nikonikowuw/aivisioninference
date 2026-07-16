import { DeleteIcon, EditIcon, ViewIcon } from '@chakra-ui/icons';
import {
  Box,
  Button,
  Flex,
  HStack,
  IconButton,
  Spinner,
  Table,
  Tag,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import { EmptyState } from 'components/empty/EmptyState';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { useDateFormat } from 'hooks/useDateFormat';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  edgeNodeScheduledTaskApi,
  type EdgeNodeScheduledTask,
} from 'services/edgeNodeScheduledTask';
import ScheduledTaskForm from './form';

interface ScheduledTaskListProps {
  nodeId: string;
}

const STATUS_COLORS: Record<string, string> = {
  active: 'green',
  disabled: 'gray',
  running: 'blue',
};

export default function ScheduledTaskList({ nodeId }: ScheduledTaskListProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const [editingTask, setEditingTask] = useState<EdgeNodeScheduledTask | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<EdgeNodeScheduledTask | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const {
    isOpen: isDeleteOpen,
    onOpen: onDeleteOpen,
    onClose: onDeleteClose,
  } = useDisclosure();

  const fetchTasks = useCallback(
    (page: number, pageSize: number) =>
      edgeNodeScheduledTaskApi.list(nodeId, { page, page_size: pageSize }),
    [nodeId],
  );

  const {
    list: tasks,
    total,
    page,
    pageSize,
    load: loadTasks,
    changePage,
    changePageSize,
  } = usePagination<EdgeNodeScheduledTask>(fetchTasks);

  useEffect(() => {
    if (nodeId) {
      loadTasks({ page: 1 }).catch(() => {
        toast({ title: t('message.loadFailed'), status: 'error' });
      });
    }
  }, [nodeId, loadTasks, toast, t]);

  const handleCreate = useCallback(() => {
    setEditingTask(null);
    setShowForm(true);
  }, []);

  const handleEdit = useCallback((task: EdgeNodeScheduledTask) => {
    setEditingTask(task);
    setShowForm(true);
  }, []);

  const handleFormClose = useCallback(() => {
    setShowForm(false);
    setEditingTask(null);
    loadTasks({ page, pageSize });
  }, [loadTasks, page, pageSize]);

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await edgeNodeScheduledTaskApi.delete(nodeId, deleteTarget.id);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      loadTasks({ page, pageSize });
    } catch {
      toast({ title: t('message.deleteFailed'), status: 'error' });
    } finally {
      setIsDeleting(false);
      onDeleteClose();
    }
  }, [deleteTarget, nodeId, toast, t, loadTasks, page, pageSize, onDeleteClose]);

  const handleViewExecutions = useCallback((task: EdgeNodeScheduledTask) => {
    // Navigate to detail view or expand inline
    toast({
      title: t('terminal.selectPrompt'),
      status: 'info',
      duration: 2000,
    });
  }, [t, toast]);

  if (showForm) {
    return (
      <ScheduledTaskForm
        nodeId={nodeId}
        task={editingTask}
        onClose={handleFormClose}
      />
    );
  }

  return (
    <Box>
      <Flex justify="space-between" align="center" mb="15px">
        <Text color={textColor} fontSize="lg" fontWeight="bold">
          {t('scheduledTasks.title')}
        </Text>
        <Button colorScheme="brand" size="sm" onClick={handleCreate}>
          {t('scheduledTasks.create')}
        </Button>
      </Flex>

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500">
            <Thead>
              <Tr>
                <Th>{t('scheduledTasks.taskName')}</Th>
                <Th>{t('scheduledTasks.command')}</Th>
                <Th>{t('scheduledTasks.cronExpr')}</Th>
                <Th>{t('scheduledTasks.taskStatus')}</Th>
                <Th>{t('scheduledTasks.timeout')}</Th>
                <Th>{t('scheduledTasks.lastRun')}</Th>
                <Th>{t('fields.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {tasks.length === 0 ? (
                <Tr>
                  <Td colSpan={7}>
                    <EmptyState />
                  </Td>
                </Tr>
              ) : (
                tasks.map((task) => (
                  <Tr key={task.id}>
                    <Td>
                      <Text fontWeight="medium" color={textColor} fontSize="sm">
                        {task.name}
                      </Text>
                    </Td>
                    <Td>
                      <Text fontSize="sm" maxW="200px" isTruncated fontFamily="mono">
                        {task.command}
                      </Text>
                    </Td>
                    <Td>
                      <Text fontSize="sm" fontFamily="mono">
                        {task.cron_expr || '-'}
                      </Text>
                    </Td>
                    <Td>
                      <Tag
                        colorScheme={STATUS_COLORS[task.status] || 'gray'}
                        size="sm"
                      >
                        {t(`scheduledTasks.status.${task.status}`, task.status)}
                      </Tag>
                    </Td>
                    <Td>
                      <Text fontSize="sm">{task.timeout_seconds}s</Text>
                    </Td>
                    <Td>
                      <Text fontSize="sm">
                        {task.last_run_at
                          ? formatDateTime(task.last_run_at)
                          : '-'}
                      </Text>
                    </Td>
                    <Td>
                      <HStack spacing={1}>
                        <IconButton
                          aria-label={t('scheduledTasks.viewExecutions')}
                          icon={<ViewIcon />}
                          size="sm"
                          variant="ghost"
                          colorScheme="blue"
                          onClick={() => handleViewExecutions(task)}
                        />
                        <IconButton
                          aria-label={tCommon('button.edit')}
                          icon={<EditIcon />}
                          size="sm"
                          variant="ghost"
                          colorScheme="blue"
                          onClick={() => handleEdit(task)}
                        />
                        <IconButton
                          aria-label={tCommon('button.delete')}
                          icon={<DeleteIcon />}
                          size="sm"
                          variant="ghost"
                          colorScheme="red"
                          onClick={() => {
                            setDeleteTarget(task);
                            onDeleteOpen();
                          }}
                        />
                      </HStack>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>
        <Pagination
          page={page}
          pageSize={pageSize}
          total={total}
          onChange={changePage}
          onPageSizeChange={changePageSize}
        />
      </Card>

      <ConfirmDialog
        isOpen={isDeleteOpen}
        onClose={onDeleteClose}
        onConfirm={handleDelete}
        title={tCommon('button.confirm')}
        message={t('scheduledTasks.deleteConfirm')}
        isLoading={isDeleting}
      />
    </Box>
  );
}
