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

export default function AuditLogs() {
  const { t } = useTranslation('modules/audit-logs');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const { filters, setFilter, resetFilters, searchTrigger, handleSearch } = useFilter();

  const fetchLogs = useCallback((p: number, ps: number) => {
    return auditLogsApi.list({
      page: p,
      page_size: ps,
      sort: 'created_at',
      order: 'desc',
      keyword: filters.keyword,
      action: filters.action,
      resource_type: filters.resource_type,
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
        onSearch={handleSearch}
        onReset={resetFilters}
        selects={[
          {
            name: 'action',
            label: t('table.columns.action'),
            options: [
              { value: 'login', label: t('filter.actions.login') },
              { value: 'logout', label: t('filter.actions.logout') },
              { value: 'create', label: t('filter.actions.create') },
              { value: 'update', label: t('filter.actions.update') },
              { value: 'delete', label: t('filter.actions.delete') },
            ],
          },
          {
            name: 'resource_type',
            label: t('table.columns.resourceType'),
            options: [
              { value: 'user', label: t('filter.resourceTypes.user') },
              { value: 'role', label: t('filter.resourceTypes.role') },
              { value: 'permission', label: t('filter.resourceTypes.permission') },
              { value: 'file', label: t('filter.resourceTypes.file') },
              { value: 'task', label: t('filter.resourceTypes.task') },
            ],
          },
        ]}
        dateRange
      />
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>{t('table.columns.username')}</Th>
              <Th>{t('table.columns.action')}</Th>
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
                <Td><Badge colorScheme="blue">{l.action}</Badge></Td>
                <Td>{l.request_method}</Td>
                <Td maxW="240px" isTruncated>{l.request_path}</Td>
                <Td>{l.request_ip}</Td>
                <Td><Badge colorScheme={l.response_status >= 400 ? 'red' : 'green'}>{l.response_status}</Badge></Td>
                <Td>{t('table.durationMs', { value: l.duration_ms ?? 0 })}</Td>
                <Td maxW="200px" isTruncated>{l.error_summary || l.result_summary}</Td>
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
