import {
  Badge,
  Box,
  Center,
  Flex,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
} from '@chakra-ui/react';
import { useDateFormat } from 'hooks/useDateFormat';
import { useEffect, useCallback } from 'react';
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

const resultColor = (result?: string) => result === 'success' ? 'green' : (result ? 'red' : 'gray');

export default function AuditLogs() {
  const { t } = useTranslation('modules/audit-logs');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchLogs = useCallback((p: number, ps: number) => {
    return auditLogsApi.list({
      page: p,
      page_size: ps,
      sort: 'created_at',
      order: 'desc',
      keyword: filters.keyword,
      resource_type: filters.resource_type,
      result: filters.result,
      start_time: filters.start_time,
      end_time: filters.end_time,
    });
  }, [filters]);

  const { list: logs, total, page, pageSize, initialLoading, pageLoading, load, changePage, changePageSize } = usePagination<AuditLog>(fetchLogs);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
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
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="900px">
          <Thead>
            <Tr>
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
                <Td>{l.username || l.user_id || '-'}</Td>
                <Td>{t(`actionTypes.${l.action_type}`, { defaultValue: l.action_type || '-' })}</Td>
                <Td><Badge colorScheme={methodColor(l.request_method)}>{l.request_method}</Badge></Td>
                <Td maxW="240px" isTruncated>{l.request_path}</Td>
                <Td>{l.request_ip}</Td>
                <Td><Badge colorScheme={l.response_status >= 400 ? 'red' : 'green'}>{l.response_status}</Badge></Td>
                <Td>{t('table.durationMs', { value: l.duration_ms ?? 0 })}</Td>
                <Td><Badge colorScheme={resultColor(l.result_summary)}>{l.result_summary || '-'}</Badge></Td>
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
