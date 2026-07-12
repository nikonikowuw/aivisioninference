import { CheckIcon } from '@chakra-ui/icons';
import {
  Box,
  Button,
  HStack,
  Select,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useToast,
  Badge,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import Pagination from 'components/pagination/Pagination';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { alertEventApi, type AlertEvent } from 'services/alertEvent';

const STATUS_COLORS: Record<string, string> = {
  firing: 'red',
  resolved: 'green',
  acknowledged: 'blue',
};

export default function AlertEventHistory() {
  const { t } = useTranslation('modules/edge-nodes');
  const textColor = useColorModeValue('navy.700', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const [statusFilter, setStatusFilter] = useState('');

  const fetchEvents = useCallback((page: number, pageSize: number) => alertEventApi.list({
    page,
    page_size: pageSize,
    status: statusFilter || undefined,
  }), [statusFilter]);

  const { list: events, total, page, pageSize, initialLoading, pageLoading, load: loadEvents, changePage, changePageSize } = usePagination<AlertEvent>(fetchEvents);

  const handleAcknowledge = async (eventId: string) => {
    try {
      await alertEventApi.acknowledge(eventId, 'admin');
      toast({ title: 'Event acknowledged', status: 'success' });
      loadEvents({ page, pageSize });
    } catch {
      toast({ title: 'Failed to acknowledge event', status: 'error' });
    }
  };

  const formatTime = (timeStr?: string) => {
    if (!timeStr) return '-';
    try {
      const d = new Date(timeStr);
      return d.toLocaleString();
    } catch {
      return timeStr;
    }
  };

  return (
    <Box pt={{ sm: '50px', md: '20px' }}>
      <Card>
        <Box p="6">
          <Text fontSize="xl" fontWeight="bold" color={textColor} mb="4">
            {t('alert.events')}
          </Text>

          <HStack mb="4" spacing="3">
            <Select
              placeholder="All Status"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              w="200px"
            >
              <option value="firing">Firing</option>
              <option value="resolved">Resolved</option>
              <option value="acknowledged">Acknowledged</option>
            </Select>
          </HStack>

          {initialLoading ? (
            <Text>Loading...</Text>
          ) : (
            <>
              <Table variant="simple" color={textColor} borderColor={borderColor}>
                <Thead>
                  <Tr>
                    <Th>Rule</Th>
                    <Th>Node</Th>
                    <Th>Value</Th>
                    <Th>Status</Th>
                    <Th>Fired At</Th>
                    <Th>Resolved At</Th>
                    <Th>Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {events.map((event) => (
                    <Tr key={event.id}>
                      <Td>
                        <Text fontWeight="medium">{event.rule_name || '-'}</Text>
                      </Td>
                      <Td>{event.node_name || '-'}</Td>
                      <Td>
                        <Text fontWeight="bold" color={event.status === 'firing' ? 'red.500' : 'green.500'}>
                          {event.metric_value?.toFixed(1)}
                        </Text>
                      </Td>
                      <Td>
                        <Badge colorScheme={STATUS_COLORS[event.status] || 'gray'}>
                          {event.status}
                        </Badge>
                      </Td>
                      <Td>{formatTime(event.fired_at)}</Td>
                      <Td>{formatTime(event.resolved_at)}</Td>
                      <Td>
                        {event.status === 'firing' && (
                          <Button
                            leftIcon={<CheckIcon />}
                            size="sm"
                            colorScheme="blue"
                            variant="outline"
                            onClick={() => handleAcknowledge(event.id)}
                          >
                            Acknowledge
                          </Button>
                        )}
                      </Td>
                    </Tr>
                  ))}
                  {events.length === 0 && (
                    <Tr>
                      <Td colSpan={7} textAlign="center" py="8">
                        <Text color="gray.500">No alert events</Text>
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
              />
            </>
          )}
        </Box>
      </Card>
    </Box>
  );
}
