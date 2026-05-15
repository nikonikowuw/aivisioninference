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
  Spinner,
  Center,
  Badge,
} from '@chakra-ui/react';
import { useEffect, useState, useCallback } from 'react';
import { auditLogsApi, type AuditLog } from 'services/api';

export default function AuditLogs() {
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);

  const loadLogs = useCallback(async () => {
    try {
      const data = await auditLogsApi.list({ page: 1, page_size: 100 });
      setLogs(data.list);
    } catch {}
  }, []);

  useEffect(() => {
    loadLogs().finally(() => setLoading(false));
  }, [loadLogs]);

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>审计日志</Text>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr><Th>ID</Th><Th>用户ID</Th><Th>操作</Th><Th>资源类型</Th><Th>资源ID</Th><Th>IP</Th><Th>详情</Th><Th>时间</Th></Tr>
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
                <Td whiteSpace="nowrap">{new Date(l.created_at).toLocaleString('zh-CN')}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </Box>
  );
}
