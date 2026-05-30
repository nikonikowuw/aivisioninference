import { Badge, Box, Button, Center, Checkbox, Flex, HStack, Spinner, Table, Tbody, Td, Text, Th, Thead, Tr, useColorModeValue, useToast } from '@chakra-ui/react';
import { useDateFormat } from 'hooks/useDateFormat';
import { DownloadIcon } from '@chakra-ui/icons';
import { useEffect, useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { auditLogsApi, type AuditLog } from 'services/api';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';

const methodColorMap = {
  GET: 'green',
  POST: 'blue',
  PUT: 'orange',
  DELETE: 'red',
  PATCH: 'teal',
} as const satisfies Record<string, string>;

const methodColor = (method: string) => methodColorMap[method as keyof typeof methodColorMap] ?? 'gray';

const resultColor = (result?: string) => (result === 'success' ? 'green' : (result ? 'red' : 'gray'));

export default function AuditLogs() {
  const { t } = useTranslation('modules/audit-logs');
  const { t: tCommon } = useTranslation('common');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();
  const [isExporting, setIsExporting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);

  const fetchLogs = useCallback((p: number, ps: number) => auditLogsApi.list({
      page: p,
      page_size: ps,
      sort: 'created_at',
      order: 'desc',
      ...filters,
    }), [filters]);

  const {
    list: logs,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
  } = usePagination<AuditLog>(fetchLogs);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const pageLogIds = logs.map((l) => l.id);
  const selectedOnPage = pageLogIds.filter((id) => selectedIds.includes(id));
  const isAllPageSelected = pageLogIds.length > 0 && selectedOnPage.length === pageLogIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    if (isAllPageSelected) {
      setSelectedIds((prev) => prev.filter((id) => !pageLogIds.includes(id)));
      return;
    }
    setSelectedIds((prev) => Array.from(new Set([...prev, ...pageLogIds])));
  };

  const toggleRowSelection = (id: string) => {
    setSelectedIds((prev) => (
      prev.includes(id) ? prev.filter((selectedId) => selectedId !== id) : [...prev, id]
    ));
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await auditLogsApi.exportCsv({
        keyword: filters.keyword,
        resource_type: filters.resource_type,
        result: filters.result,
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
          {selectedIds.length > 0 ? tCommon('button.exportSelected') : tCommon('button.export')}
        </Button>
      </Flex>
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          {
            name: 'resource_type',
            label: t('table.columns.resourceType'),
            options: [
              { value: 'user', label: t('filter.resourceTypes.user') },
              { value: 'role', label: t('filter.resourceTypes.role') },
              { value: 'permission', label: t('filter.resourceTypes.permission') },
              { value: 'file', label: t('filter.resourceTypes.file') },
              { value: 'task', label: t('filter.resourceTypes.task') },
              { value: 'system', label: t('filter.resourceTypes.system') },
              { value: 'feedback', label: t('filter.resourceTypes.feedback') },
            ],
          },
          {
            name: 'result',
            label: t('table.columns.result'),
            options: [
              { value: 'success', label: t('filter.results.success') },
              { value: 'failed', label: t('filter.results.failed') },
            ],
          },
        ]}
        dateRange
      />
      {selectedIds.length > 0 && (
        <Flex mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{t('batch.selected', { count: selectedIds.length })}</Text>
        </Flex>
      )}
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="900px">
          <Thead>
            <Tr>
              <Th w="48px">
                <Checkbox
                  isChecked={isAllPageSelected}
                  isIndeterminate={isPageSelectionIndeterminate}
                  onChange={togglePageSelection}
                />
              </Th>
              <Th>{t('table.columns.username')}</Th>
              <Th>{t('table.columns.actionType')}</Th>
              <Th>{t('table.columns.method')}</Th>
              <Th>{t('table.columns.path')}</Th>
              <Th>{t('table.columns.ip')}</Th>
              <Th>{t('table.columns.status')}</Th>
              <Th>{t('table.columns.duration')}</Th>
              <Th>{t('table.columns.result')}</Th>
              <Th>{t('table.columns.time')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {logs.map((l) => (
              <Tr key={l.id}>
                <Td>
                  <Checkbox
                    isChecked={selectedIds.includes(l.id)}
                    onChange={() => toggleRowSelection(l.id)}
                  />
                </Td>
                <Td>{l.username || l.user_id || '-'}</Td>
                <Td>{t(`actionTypes.${l.action_type.replace(/:/g, '.')}`, { defaultValue: l.action_type || '-' })}</Td>
                <Td><Badge colorScheme={methodColor(l.request_method)}>{l.request_method}</Badge></Td>
                <Td maxW="240px" isTruncated>{l.request_path}</Td>
                <Td>{l.request_ip}</Td>
                <Td><Badge colorScheme={l.response_status >= 400 ? 'red' : 'green'}>{l.response_status}</Badge></Td>
                <Td>{t('table.durationMs', { value: l.duration_ms ?? 0 })}</Td>
                <Td><Badge colorScheme={resultColor(l.result_summary)}>{t(`filter.results.${l.result_summary}`, { defaultValue: l.result_summary || '-' })}</Badge></Td>
                <Td whiteSpace="nowrap">{formatDateTime(l.created_at)}</Td>
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
    </Box>
  );
}
