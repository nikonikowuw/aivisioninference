import { Badge, Box, Button, Flex, HStack, Select, Table, Tbody, Td, Text, Th, Thead, Tr, useColorModeValue, useToast } from '@chakra-ui/react';
import { useEffect, useMemo, useState } from 'react';
import { DownloadIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { SearchBar } from 'components/search-bar/SearchBar';
import Pagination from 'components/pagination/Pagination';
import { useDateFormat } from 'hooks/useDateFormat';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { feedbackApi, type Feedback } from 'services/api';

const statusColor: Record<string, string> = {
  open: 'yellow',
  processing: 'blue',
  resolved: 'green',
  closed: 'gray',
};

export default function FeedbackPage() {
  const { t } = useTranslation('modules/feedback');
  const { t: tCommon } = useTranslation('common');
  const toast = useToast();
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();
  const [isExporting, setIsExporting] = useState(false);

  const fetchFeedback = useMemo(() => (p: number, ps: number) => feedbackApi.list({
    page: p,
    page_size: ps,
    keyword: filters.keyword,
    source: filters.source,
    status: filters.status,
    start_time: filters.start_time,
    end_time: filters.end_time,
  }), [filters]);

  const { list, total, page, pageSize, load, changePage, changePageSize } = usePagination<Feedback>(fetchFeedback);

  useEffect(() => { load({ page: 1 }); }, [searchTrigger, load]);

  async function changeStatus(id: string, status: string) {
    try {
      await feedbackApi.updateStatus(id, status);
      toast({ title: t('message.updated'), status: 'success' });
      load();
    } catch (err) {
      toast({ title: t('message.updateFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    }
  }

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await feedbackApi.exportCsv({
        keyword: filters.keyword,
        source: filters.source,
        status: filters.status,
        start_time: filters.start_time,
        end_time: filters.end_time,
      });
    } catch (err) {
      toast({ title: tCommon('message.exportFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button leftIcon={<DownloadIcon />} variant="outline" onClick={handleExport} isLoading={isExporting}>{tCommon('button.export')}</Button>
      </Flex>
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          { name: 'source', label: t('table.source'), options: [{ value: 'user', label: t('source.user') }, { value: 'email', label: t('source.email') }] },
          { name: 'status', label: t('table.status'), options: [{ value: 'open', label: t('status.open') }, { value: 'processing', label: t('status.processing') }, { value: 'resolved', label: t('status.resolved') }, { value: 'closed', label: t('status.closed') }] },
        ]}
        dateRange
      />
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="900px">
          <Thead>
            <Tr>
              <Th>{t('table.columns.id', { ns: 'common', defaultValue: 'ID' })}</Th>
              <Th>{t('table.source')}</Th>
              <Th>{t('table.category')}</Th>
              <Th>{t('table.title')}</Th>
              <Th>{t('table.content')}</Th>
              <Th>{t('table.status')}</Th>
              <Th>{t('table.createdAt')}</Th>
              <Th>{t('table.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {list.map(item => (
              <Tr key={item.id}>
                <Td>{item.id}</Td>
                <Td>{t(`source.${item.source}`)}</Td>
                <Td>{t(`filter.resourceTypes.${item.category}`, { ns: 'modules/audit-logs', defaultValue: item.category || '-' })}</Td>
                <Td>{item.title}</Td>
                <Td maxW="320px" whiteSpace="normal">{item.content}</Td>
                <Td><Badge colorScheme={statusColor[item.status] || 'gray'}>{t(`status.${item.status}`)}</Badge></Td>
                <Td>{formatDateTime(item.created_at)}</Td>
                <Td>
                  <HStack spacing={2}>
                    <Select size="sm" value={item.status} onChange={(e) => changeStatus(item.id, e.target.value)}>
                      <option value="open">{t('status.open')}</option>
                      <option value="processing">{t('status.processing')}</option>
                      <option value="resolved">{t('status.resolved')}</option>
                      <option value="closed">{t('status.closed')}</option>
                    </Select>
                    <Button size="sm" onClick={() => changeStatus(item.id, item.status)}>{t('actions.refresh')}</Button>
                  </HStack>
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
        />
      </Box>
    </Box>
  );
}
