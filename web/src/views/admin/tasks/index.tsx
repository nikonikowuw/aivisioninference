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
  Checkbox,
} from '@chakra-ui/react';
import { CloseIcon, DownloadIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { tasksApi, type Task } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';

const statusColor: Record<string, string> = {
  pending: 'yellow',
  running: 'blue',
  completed: 'green',
  failed: 'red',
  cancelled: 'gray',
};

export default function Tasks() {
  const { t } = useTranslation('modules/tasks');
  const { t: tCommon } = useTranslation('common');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const [cancelTarget, setCancelTarget] = useState<string | null>(null);
  const [isCancelling, setIsCancelling] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [isBatching, setIsBatching] = useState(false);
  const [isBatchConfirmOpen, setIsBatchConfirmOpen] = useState(false);

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchTasks = useCallback((p: number, ps: number) => tasksApi.list({
      page: p,
      page_size: ps,
      ...filters,
    }), [filters]);

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
  } = usePagination<Task>(fetchTasks);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const pageTaskIds = tasks.map((t) => t.id);
  const selectedOnPage = pageTaskIds.filter((id) => selectedIds.includes(id));
  const isAllPageSelected = pageTaskIds.length > 0 && selectedOnPage.length === pageTaskIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    if (isAllPageSelected) {
      setSelectedIds((prev) => prev.filter((id) => !pageTaskIds.includes(id)));
      return;
    }
    setSelectedIds((prev) => Array.from(new Set([...prev, ...pageTaskIds])));
  };

  const toggleRowSelection = (id: string) => {
    setSelectedIds((prev) => (
      prev.includes(id) ? prev.filter((selectedId) => selectedId !== id) : [...prev, id]
    ));
  };

  const handleBatchCancel = async () => {
    setIsBatching(true);
    try {
      const result = await tasksApi.batchCancel(selectedIds);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      setSelectedIds([]);
      await load();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setIsBatchConfirmOpen(false);
    }
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await tasksApi.exportCsv({
        keyword: filters.keyword,
        type: filters.type,
        status: filters.status,
        start_time: filters.start_time,
        end_time: filters.end_time,
        ids: selectedIds.length > 0 ? selectedIds.join(',') : undefined,
      });
    } catch (err) {
      toast({ title: tCommon('message.exportFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  const handleCancel = async () => {
    if (!cancelTarget) return;
    setIsCancelling(true);
    try {
      await tasksApi.cancel(cancelTarget);
      toast({ title: t('message.cancelled'), status: 'success' });
      load();
    } catch (err) {
      toast({ title: t('message.cancelFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsCancelling(false);
      setCancelTarget(null);
    }
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button
          leftIcon={<DownloadIcon />}
          variant="outline"
          onClick={handleExport}
          isLoading={isExporting}
        >
          {selectedIds.length > 0 ? t('actions.exportSelected') : tCommon('button.export')}
        </Button>
      </Flex>
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          {
            name: 'type',
            label: t('table.columns.type'),
            options: [
              { value: 'email', label: t('filter.taskTypes.email') },
              { value: 'export', label: t('filter.taskTypes.export') },
              { value: 'import', label: t('filter.taskTypes.import') },
              { value: 'backup', label: t('filter.taskTypes.backup') },
            ],
          },
          {
            name: 'status',
            label: t('table.columns.status'),
            options: [
              { value: 'pending', label: t('table.status.pending') },
              { value: 'running', label: t('table.status.running') },
              { value: 'completed', label: t('table.status.completed') },
              { value: 'failed', label: t('table.status.failed') },
              { value: 'cancelled', label: t('table.status.cancelled') },
            ],
          },
        ]}
        dateRange
      />
      {selectedIds.length > 0 && (
        <Flex mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{t('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <Button size="sm" colorScheme="red" onClick={() => setIsBatchConfirmOpen(true)}>{t('actions.cancel')}</Button>
          </HStack>
        </Flex>
      )}
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="700px">
          <Thead>
            <Tr>
              <Th w="48px">
                <Checkbox
                  isChecked={isAllPageSelected}
                  isIndeterminate={isPageSelectionIndeterminate}
                  onChange={togglePageSelection}
                />
              </Th>
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
                <Td>
                  <Checkbox
                    isChecked={selectedIds.includes(task.id)}
                    onChange={() => toggleRowSelection(task.id)}
                  />
                </Td>
                <Td>{task.id}</Td>
                <Td fontWeight="600">{t(`filter.taskTypes.${task.type}`, { defaultValue: task.type })}</Td>
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
        isOpen={cancelTarget !== null}
        onClose={() => setCancelTarget(null)}
        onConfirm={handleCancel}
        title={t('actions.cancel')}
        message={t('message.cancelConfirm')}
        confirmText={t('message.confirmCancel')}
        isLoading={isCancelling}
      />
      <ConfirmDialog
        isOpen={isBatchConfirmOpen}
        onClose={() => setIsBatchConfirmOpen(false)}
        onConfirm={handleBatchCancel}
        title={t('actions.cancel')}
        message={t('message.batchCancelConfirm', { count: selectedIds.length })}
        confirmText={t('message.confirmCancel')}
        isLoading={isBatching}
      />
    </Box>
  );
}
