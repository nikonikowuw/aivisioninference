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
  useToast,
  Tooltip,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { aiTimeSchedulesApi, type AITimeSchedule } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import { EmptyState } from 'components/empty/EmptyState';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import ScheduleFormModal from './components/ScheduleFormModal';

export default function AITimeSchedules() {
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { t } = useTranslation(['modules/ai-time-schedules', 'common']);

  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editingSchedule, setEditingSchedule] = useState<AITimeSchedule | null>(null);

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchSchedules = useCallback(
    (p: number, ps: number) =>
      aiTimeSchedulesApi.list({ page: p, page_size: ps, ...filters }),
    [filters],
  );

  const {
    list: schedules,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
  } = usePagination<AITimeSchedule>(fetchSchedules);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await aiTimeSchedulesApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      load();
    } catch (err: any) {
      toast({ title: t('message.deleteFailed'), description: err?.message || '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  const formatTimeWindows = (tw: { start: string; end: string }[]) => {
    if (!tw || tw.length === 0) return '-';
    return tw.map(w => `${w.start} - ${w.end}`).join('、');
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
            setEditingSchedule(null);
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
      />

      <Box
        bg={bgCard}
        borderRadius="16px"
        border="1px solid"
        borderColor={borderColor}
        overflow="auto"
      >
        <Table variant="simple" size="md" minW="700px">
          <Thead>
            <Tr>
              <Th>{t('fields.name')}</Th>
              <Th>{t('fields.description')}</Th>
              <Th>{t('fields.dateRange')}</Th>
              <Th>{t('fields.timeWindows')}</Th>
              <Th>{t('fields.updatedAt')}</Th>
              <Th>{t('common:button.edit', { defaultValue: '操作' })}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {schedules.map(schedule => (
              <Tr key={schedule.id}>
                <Td fontWeight="600">{schedule.name}</Td>
                <Td maxW="200px" isTruncated>
                  <Tooltip label={schedule.description}>{schedule.description || '-'}</Tooltip>
                </Td>
                <Td whiteSpace="nowrap">
                  {schedule.start_date?.substring(0, 10)} ~ {schedule.end_date?.substring(0, 10)}
                </Td>
                <Td maxW="200px" isTruncated>
                  <Tooltip label={formatTimeWindows(schedule.time_windows)}>
                    {formatTimeWindows(schedule.time_windows)}
                  </Tooltip>
                </Td>
                <Td whiteSpace="nowrap">{formatDateTime(schedule.updated_at)}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton
                      aria-label={t('actions.edit')}
                      icon={<EditIcon />}
                      size="sm"
                      variant="ghost"
                      colorScheme="blue"
                      onClick={() => {
                        setEditingSchedule(schedule);
                        setFormOpen(true);
                      }}
                    />
                    <IconButton
                      aria-label={t('actions.delete')}
                      icon={<DeleteIcon />}
                      size="sm"
                      variant="ghost"
                      colorScheme="red"
                      onClick={() => setDeleteTarget(schedule.id)}
                    />
                  </HStack>
                </Td>
              </Tr>
            ))}
            {schedules.length === 0 && (
              <Tr>
                <Td colSpan={6}>
                  <EmptyState title={t('empty')} />
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

      <ScheduleFormModal
        isOpen={formOpen}
        onClose={() => setFormOpen(false)}
        onSuccess={() => {
          setFormOpen(false);
          load();
        }}
        initialData={editingSchedule}
      />
    </Box>
  );
}
