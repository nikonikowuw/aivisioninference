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
  useToast,
  Spinner,
  Center,
  Badge,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { auditLogsApi, type AuditLog } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';

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
      const data = await auditLogsApi.list({ page: 1, page_size: 100 });
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
              <Th>{t('table.columns.id')}</Th>
              <Th>{t('table.columns.userId')}</Th>
              <Th>{t('table.columns.action')}</Th>
              <Th>{t('table.columns.resourceType')}</Th>
              <Th>{t('table.columns.resourceId')}</Th>
              <Th>{t('table.columns.ip')}</Th>
              <Th>{t('table.columns.detail')}</Th>
              <Th>{t('table.columns.time')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {logs.map((l) => (
              <Tr key={l.id}>
                <Td>{l.id}</Td>
                <Td>{l.user_id}</Td>
                <Td><Badge colorScheme="blue">{l.action}</Badge></Td>
                <Td>{l.resource_type}</Td>
                <Td>{l.resource_id}</Td>
                <Td>{l.ip}</Td>
                <Td maxW="200px" isTruncated>{l.detail}</Td>
                <Td whiteSpace="nowrap">{formatDateTime(l.created_at)}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </Box>
  );
}
