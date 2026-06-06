import {
  Box,
  Button,
  Flex,
  Input,
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
  HStack,
  Select,
} from '@chakra-ui/react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import { DatePicker } from 'components/search-bar/DatePicker';

interface TimeConfig {
  current_time: string;
  timezone: string;
  time_mode: string;
  ntp_config?: {
    enabled: boolean;
    servers: Array<{ host: string; status: string; latency_ms?: number }>;
    sync_interval: number;
    last_sync_at?: string;
    last_sync_status?: string;
  };
}

export default function TimeConfigTab() {
  const { t, i18n } = useTranslation(['modules/system', 'common']);
  const { formatDateTime } = useDateFormat();
  const toast = useToast();
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const [config, setConfig] = useState<TimeConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [newServer, setNewServer] = useState('');
  const [manualDate, setManualDate] = useState('');
  const [manualTime, setManualTime] = useState('00:00:00');

  useEffect(() => {
    loadConfig();
  }, []);

  async function loadConfig() {
    setLoading(true);
    try {
      const data = await request<TimeConfig>('/system/time');
      setConfig(data);
      if (data?.current_time) {
        const d = new Date(data.current_time);
        const y = d.getFullYear();
        const m = String(d.getMonth() + 1).padStart(2, '0');
        const day = String(d.getDate()).padStart(2, '0');
        setManualDate(`${y}-${m}-${day}`);
        setManualTime(`${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`);
      }
    } catch (err) {
      console.error('Failed to load time config:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleAddServer() {
    if (!newServer) return;
    try {
      await request('/system/time/ntp/servers', {
        method: 'POST',
        body: JSON.stringify({ host: newServer }),
      });
      toast({ title: t('time.server_added'), status: 'success', duration: 3000 });
      setNewServer('');
      loadConfig();
    } catch (err: any) {
      toast({ title: err.message || t('time.add_failed'), status: 'error', duration: 3000 });
    }
  }

  async function handleRemoveServer(host: string) {
    try {
      await request(`/system/time/ntp/servers/${host}`, { method: 'DELETE' });
      toast({ title: t('time.server_removed'), status: 'success', duration: 3000 });
      loadConfig();
    } catch (err: any) {
      toast({ title: err.message || t('time.remove_failed'), status: 'error', duration: 3000 });
    }
  }

  async function handleSyncNTP() {
    try {
      await request('/system/time/ntp/sync', { method: 'POST' });
      toast({ title: t('time.sync_success'), status: 'success', duration: 3000 });
      loadConfig();
    } catch (err: any) {
      toast({ title: err.message || t('time.sync_failed'), status: 'error', duration: 3000 });
    }
  }

  async function handleSetManualTime() {
    if (!manualDate || !manualTime) return;
    try {
      // 组合日期和时间
      const isoString = `${manualDate}T${manualTime}`;
      await request('/system/time/manual', {
        method: 'POST',
        body: JSON.stringify({ time: new Date(isoString).toISOString() }),
      });
      toast({ title: t('time.time_set'), status: 'success', duration: 3000 });
      loadConfig();
    } catch (err: any) {
      toast({ title: err.message || t('time.set_failed'), status: 'error', duration: 3000 });
    }
  }

  if (loading) {
    return (
      <Flex justify="center" align="center" h="200px">
        <Spinner size="xl" />
      </Flex>
    );
  }

  // 生成小时/分钟/秒的选项
  const hours = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'));
  const minutes = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, '0'));
  const seconds = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, '0'));

  const [h, m, s] = manualTime.split(':');

  return (
    <Box>
      {/* 当前时间信息 */}
      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor} mb={6}>
        <Text fontWeight="600" mb={4}>{t('time.current_time')}</Text>
        <Flex gap={8}>
          <Box>
            <Text fontSize="sm" color="gray.500">{t('time.time')}</Text>
            <Text fontSize="xl" fontWeight="600">{config?.current_time ? formatDateTime(new Date(config.current_time)) : '-'}</Text>
          </Box>
          <Box>
            <Text fontSize="sm" color="gray.500">{t('time.timezone')}</Text>
            <Text fontSize="xl" fontWeight="600">{config?.timezone}</Text>
          </Box>
          <Box>
            <Text fontSize="sm" color="gray.500">{t('time.mode')}</Text>
            <Text fontSize="xl" fontWeight="600">{config?.time_mode === 'ntp' ? t('time.mode_ntp') : t('time.mode_manual')}</Text>
          </Box>
        </Flex>
      </Box>

      {/* NTP 配置 */}
      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor} mb={6}>
        <Flex justify="space-between" align="center" mb={4}>
          <Text fontWeight="600">{t('time.ntp_servers')}</Text>
          <Button size="sm" colorScheme="brand" onClick={handleSyncNTP}>
            {t('time.sync_now')}
          </Button>
        </Flex>

        <Table variant="simple" mb={4}>
          <Thead>
            <Tr>
              <Th>{t('time.server')}</Th>
              <Th>{t('time.status')}</Th>
              <Th>{t('time.latency')}</Th>
              <Th>{t('common:actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {config?.ntp_config?.servers?.map((server) => (
              <Tr key={server.host}>
                <Td>{server.host}</Td>
                <Td>
                  <Text color={server.status === 'reachable' ? 'green.500' : 'red.500'}>
                    {server.status === 'reachable' ? t('time.status_reachable') : (server.status === 'unreachable' ? t('time.status_unreachable') : t('time.status_unknown'))}
                  </Text>
                </Td>
                <Td>{server.latency_ms ? `${server.latency_ms}ms` : '-'}</Td>
                <Td>
                  <Button size="sm" colorScheme="red" variant="ghost" onClick={() => handleRemoveServer(server.host)}>
                    {t('common:delete')}
                  </Button>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>

        <Flex gap={4}>
          <Input
            placeholder={t('time.new_server_placeholder')}
            value={newServer}
            onChange={(e) => setNewServer(e.target.value)}
          />
          <Button colorScheme="brand" onClick={handleAddServer}>{t('time.add_server')}</Button>
        </Flex>
      </Box>

      {/* 手动设置时间 */}
      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
        <Text fontWeight="600" mb={4}>{t('time.manual_set')}</Text>
        <Flex direction={{ base: 'column', md: 'row' }} gap={4} align={{ md: 'center' }}>
          <HStack spacing={2} flex={1}>
            <Box flex={2}>
              <DatePicker value={manualDate} onChange={setManualDate} />
            </Box>
            <HStack flex={3} spacing={1}>
              <Select value={h} onChange={(e) => setManualTime(`${e.target.value}:${m}:${s}`)}>
                {hours.map(v => <option key={v} value={v}>{v}</option>)}
              </Select>
              <Text>:</Text>
              <Select value={m} onChange={(e) => setManualTime(`${h}:${e.target.value}:${s}`)}>
                {minutes.map(v => <option key={v} value={v}>{v}</option>)}
              </Select>
              <Text>:</Text>
              <Select value={s} onChange={(e) => setManualTime(`${h}:${m}:${e.target.value}`)}>
                {seconds.map(v => <option key={v} value={v}>{v}</option>)}
              </Select>
            </HStack>
          </HStack>
          <Button colorScheme="brand" onClick={handleSetManualTime} px={8}>
            {t('time.set_time')}
          </Button>
        </Flex>
      </Box>
    </Box>
  );
}
