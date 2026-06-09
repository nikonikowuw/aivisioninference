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
  Badge,
  Spinner,
  Center,
  SimpleGrid,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  Icon,
  Tooltip,
  VStack,
  HStack,
  IconButton,
} from '@chakra-ui/react';
import { RepeatIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { mediaApi, type StreamState } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import Card from 'components/card/Card';
import { MdRefresh, MdTimeline } from 'react-icons/md';

const statusColor: Record<string, string> = {
  inactive: 'gray',
  pulling: 'blue',
  active: 'green',
  error: 'red',
};

const StatCard = ({ label, value, helpText, color, secondaryColor }: any) => (
  <Card p={4}>
    <Stat>
      <StatLabel color={secondaryColor}>{label}</StatLabel>
      <StatNumber fontSize="2xl" color={color}>{value}</StatNumber>
      <StatHelpText>{helpText}</StatHelpText>
    </Stat>
  </Card>
);

export default function StreamStatus() {
  const { t } = useTranslation('modules/media');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const secondaryColor = useColorModeValue('gray.600', 'gray.400');

  const [streams, setStreams] = useState<StreamState[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchStreams = useCallback(async () => {
    try {
      const data = await mediaApi.listStreams();
      setStreams(data);
    } catch (err) {
      console.error('Failed to fetch streams:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStreams();
    const timer = setInterval(() => {
      if (document.visibilityState === 'visible') {
        fetchStreams();
      }
    }, 5000); // 5s 自动刷新
    return () => clearInterval(timer);
  }, [fetchStreams]);

  const safeStreams = streams || [];
  const activeCount = safeStreams.filter(s => s.status === 'active').length;
  const errorCount = safeStreams.filter(s => s.status === 'error').length;
  const totalRefCount = safeStreams.reduce((acc, s) => acc + (s.ref_count || 0), 0);

  if (loading && safeStreams.length === 0) {
    return (
      <Center h="400px">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <HStack spacing={2}>
          <Icon as={MdTimeline} w={8} h={8} color="brand.500" />
          <Text fontSize="2xl" fontWeight="bold" color={textColor}>
            {t('streamStatus.title')}
          </Text>
        </HStack>
        <IconButton
          aria-label="Refresh"
          variant="ghost"
          icon={<MdRefresh size={20} />}
          onClick={() => {
            setLoading(true);
            fetchStreams();
          }}
        />
      </Flex>

      <SimpleGrid columns={{ base: 1, md: 3 }} spacing="20px" mb="20px">
        <StatCard
          label={t('streamStatus.runningStreams')}
          value={activeCount}
          helpText={t('streamStatus.totalStreams', { count: safeStreams.length })}
          color="green.500"
          secondaryColor={secondaryColor}
        />
        <StatCard
          label={t('streamStatus.activeRefs')}
          value={totalRefCount}
          helpText={t('streamStatus.activeRefsDetail')}
          color="brand.500"
          secondaryColor={secondaryColor}
        />
        <StatCard
          label={t('streamStatus.errorStreams')}
          value={errorCount}
          helpText={t('streamStatus.errorStreamsDetail')}
          color="red.500"
          secondaryColor={secondaryColor}
        />
      </SimpleGrid>

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500">
            <Thead>
              <Tr>
                <Th>{t('streamStatus.deviceId')}</Th>
                <Th>{t('streamStatus.status')}</Th>
                <Th>{t('streamStatus.refCount')}</Th>
                <Th>{t('streamStatus.consumers')}</Th>
                <Th>{t('streamStatus.startTime')}</Th>
                <Th>{t('streamStatus.retryInfo')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {safeStreams.length === 0 ? (
                <Tr>
                  <Td colSpan={6}>
                    <Center py={10}>
                      <Text color="gray.400">{t('streamStatus.noActiveStreams')}</Text>
                    </Center>
                  </Td>
                </Tr>
              ) : (
                safeStreams.map((stream) => (
                  <Tr key={stream.device_id || `${stream.app}-${stream.stream}`}>
                    <Td>
                      <VStack align="start" spacing={0}>
                        <Text color={textColor} fontWeight="bold" fontSize="sm">
                          {stream.device_id}
                        </Text>
                        <Text fontSize="xs" color="gray.400">
                          {stream.app}/{stream.stream}
                        </Text>
                      </VStack>
                    </Td>
                    <Td>
                      <Badge colorScheme={statusColor[stream.status] || 'gray'}>
                        {(stream.status || 'unknown').toUpperCase()}
                      </Badge>
                    </Td>
                    <Td>
                      <Badge variant="outline" colorScheme="brand">
                        {stream.ref_count}
                      </Badge>
                    </Td>
                    <Td>
                      <HStack wrap="wrap" spacing={1}>
                        {stream.consumers && Object.entries(stream.consumers).map(([key, c]) => (
                          <Tooltip key={key} label={t('streamStatus.refTime', { time: formatDateTime(c.ref_at) })}>
                            <Badge size="sm" variant="subtle" colorScheme="blue" textTransform="none">
                              {c.reason}
                            </Badge>
                          </Tooltip>
                        ))}
                      </HStack>
                    </Td>
                    <Td fontSize="sm">
                      {stream.started_at ? formatDateTime(stream.started_at) : '-'}
                    </Td>
                    <Td>
                      {stream.retry_count > 0 ? (
                        <VStack align="start" spacing={0}>
                          <HStack spacing={1} color="orange.500">
                            <RepeatIcon boxSize={3} />
                            <Text fontSize="xs">{t('streamStatus.retrying', { count: stream.retry_count })}</Text>
                          </HStack>
                          {stream.retry_at && (
                            <Text fontSize="xs" color="gray.400">
                              {t('streamStatus.nextRetry', { time: formatDateTime(stream.retry_at) })}
                            </Text>
                          )}
                        </VStack>
                      ) : (
                        <Text fontSize="xs" color="gray.400">{t('streamStatus.normal')}</Text>
                      )}
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>
      </Card>
    </Box>
  );
}
