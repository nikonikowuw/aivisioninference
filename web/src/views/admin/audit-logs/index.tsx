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
  useToast,
} from '@chakra-ui/react';
import { useDateFormat } from 'hooks/useDateFormat';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { auditLogsApi, type AuditLog } from 'services/api';

export default function AuditLogs() {
  const { t } = useTranslation('modules/audit-logs');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

  const loadLogs = useCallback(async () => {
    try {
      const data = await auditLogsApi.list({ page: 1, page_size: 100, sort: 'created_at', order: 'desc' });
      setLogs(data.list);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    }
  }, [toast, t]);

  useEffect(() => {
    loadLogs().finally(() => setLoading(false));
  }, [loadLogs]);

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
      </Box>
    </Box>
  );
}
