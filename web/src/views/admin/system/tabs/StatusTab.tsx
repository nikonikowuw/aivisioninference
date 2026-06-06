import {
  Box,
  Flex,
  Grid,
  GridItem,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tooltip,
  Tr,
  useColorModeValue,
} from '@chakra-ui/react';
import { ApexOptions } from 'apexcharts';
import Card from 'components/card/Card';
import EChartsRadial from 'components/charts/EChartsRadial';
import ReactApexChart from 'react-apexcharts';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useDateFormat } from 'hooks/useDateFormat';
import { request } from 'services/api';

interface RealtimeStatus {
  cpu: { usage_percent: number };
  memory: { total_mb: number; used_mb: number; usage_percent: number };
  uptime_seconds: number;
  tasks: { running: number; failed: number };
}

interface ResourceStatus {
  npu: { usage_percent: number; temperature: number; supported: boolean };
  disks: Array<{ mount_point: string; total_gb: number; used_gb: number; usage_percent: number }>;
  networks: Array<{ name: string; mac: string; ip: string; state: string }>;
}

interface ServiceStatus {
  name: string;
  status: string;
  pid?: number;
  port?: number;
  message?: string;
}

interface HistoryPoint {
  time: string;
  value: number;
}

interface HistoryData {
  metric: string;
  points: HistoryPoint[];
}

interface ServiceStatusResponse {
  device_model: string;
  versions: { app: string; go: string; cpp_engine: string };
  time: { current: string; timezone: string; ntp_status: string };
  services: ServiceStatus[];
  timestamp: string;
}

export default function StatusTab() {
  const { t } = useTranslation('modules/system');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const [realtime, setRealtime] = useState<RealtimeStatus | null>(null);
  const [resources, setResources] = useState<ResourceStatus | null>(null);
  const [serviceData, setServiceData] = useState<ServiceStatusResponse | null>(null);
  const [cpuHistory, setCpuHistory] = useState<HistoryPoint[]>([]);
  const [memoryHistory, setMemoryHistory] = useState<HistoryPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [sipOnlineCount, setSipOnlineCount] = useState(0);

  useEffect(() => {
    loadStatus();
    const interval = setInterval(() => {
      loadRealtime();
      loadHistory();
    }, 5000);
    return () => clearInterval(interval);
  }, []);

  async function loadStatus() {
    setLoading(true);
    try {
      await Promise.all([loadRealtime(), loadResources(), loadServices(), loadHistory(), loadSipStatus()]);
    } finally {
      setLoading(false);
    }
  }

  async function loadRealtime() {
    try {
      const data = await request<RealtimeStatus>('/system/status/realtime');
      setRealtime(data);
    } catch (err) {
      console.error('Failed to load realtime status:', err);
    }
  }

  async function loadResources() {
    try {
      const data = await request<ResourceStatus>('/system/status/resources');
      setResources(data);
    } catch (err) {
      console.error('Failed to load resources:', err);
    }
  }

  async function loadServices() {
    try {
      const data = await request<ServiceStatusResponse>('/system/status/services');
      setServiceData(data);
    } catch (err) {
      console.error('Failed to load services:', err);
    }
  }

  async function loadSipStatus() {
    try {
      await request('/system/status/sip');
      const data = await request<{ list: unknown[]; total: number }>('/gb28181/devices?status=online&page=1&page_size=1');
      setSipOnlineCount(data?.total || 0);
    } catch (err) {
      console.error('Failed to load SIP status:', err);
      setSipOnlineCount(0);
    }
  }

  async function loadHistory() {
    try {
      const [cpuData, memData] = await Promise.all([
        request<HistoryData>('/system/status/history?metric=cpu&duration=5m'),
        request<HistoryData>('/system/status/history?metric=memory&duration=5m'),
      ]);
      setCpuHistory(cpuData?.points || []);
      setMemoryHistory(memData?.points || []);
    } catch (err) {
      console.error('Failed to load history:', err);
    }
  }

  function formatUptime(seconds: number): string {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) return `${days}${t('status.days')} ${hours}${t('status.hours')}`;
    if (hours > 0) return `${hours}${t('status.hours')} ${minutes}${t('status.minutes')}`;
    return `${minutes}${t('status.minutes')}`;
  }

  function getStatusColor(status: string): string {
    switch (status) {
      case 'running': return 'green.500';
      case 'stopped': return 'red.500';
      case 'error': return 'red.500';
      default: return 'gray.500';
    }
  }

  // 径向条形图通用配置（已迁移至 EChartsRadial 组件）

  // CPU 图表配置
  const cpuChartData = [{
    name: t('status.cpu'),
    data: cpuHistory.map(p => p.value),
  }];

  const cpuChartOptions: ApexOptions = {
    chart: {
      type: 'area',
      toolbar: { show: false },
      sparkline: { enabled: false },
    },
    colors: ['#4318FF'],
    markers: {
      size: 0,
      strokeWidth: 0,
    },
    tooltip: {
      theme: 'dark',
      y: { formatter: (val) => `${val.toFixed(1)}%` },
    },
    dataLabels: { enabled: false },
    stroke: {
      curve: 'smooth',
      width: 2,
    },
    xaxis: {
      categories: cpuHistory.map(p => {
        const date = new Date(p.time);
        return `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`;
      }),
      labels: { show: false },
      axisBorder: { show: false },
      axisTicks: { show: false },
    },
    yaxis: {
      min: 0,
      max: 100,
      labels: {
        formatter: (val) => `${val}%`,
        style: { colors: '#A3AED0', fontSize: '11px' },
      },
    },
    fill: {
      type: 'gradient',
      gradient: {
        shadeIntensity: 1,
        opacityFrom: 0.7,
        opacityTo: 0.2,
        stops: [0, 90, 100],
      },
    },
    grid: {
      show: false,
    },
  };

  // 内存图表配置
  const memChartData = [{
    name: t('status.memory'),
    data: memoryHistory.map(p => p.value),
  }];

  const memChartOptions: ApexOptions = {
    ...cpuChartOptions,
    colors: ['#05CD99'],
    yaxis: {
      min: 0,
      max: 100,
      labels: {
        formatter: (val) => `${val}%`,
        style: { colors: '#A3AED0', fontSize: '11px' },
      },
    },
  };

  // 计算磁盘总量和已用量
  const totalDiskGb = resources?.disks?.reduce((acc, disk) => acc + disk.total_gb, 0) || 0;
  const usedDiskGb = resources?.disks?.reduce((acc, disk) => acc + disk.used_gb, 0) || 0;
  const diskUsagePercent = totalDiskGb > 0 ? (usedDiskGb / totalDiskGb) * 100 : 0;

  if (loading) {
    return (
      <Flex justify="center" align="center" h="200px">
        <Spinner size="xl" />
      </Flex>
    );
  }

  return (
    <Box>
      <Grid templateColumns={{ base: "1fr", md: "repeat(3, 1fr)", xl: "repeat(5, 1fr)" }} gap={6} mb={6}>
        <GridItem>
          <Card p={5} h="100%" display="flex" flexDirection="column" alignItems="center" justifyContent="center" variant="outline" border="1px solid" borderColor={borderColor} boxShadow="sm">
            <Text color="gray.500" fontSize="xs" fontWeight="700" textTransform="uppercase" mb={2} letterSpacing="wider" alignSelf="flex-start">
              {t('status.cpu')}
            </Text>
            <Box h="130px" w="100%">
              <EChartsRadial percent={realtime?.cpu?.usage_percent || 0} />
            </Box>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={5} h="100%" display="flex" flexDirection="column" alignItems="center" justifyContent="center" variant="outline" border="1px solid" borderColor={borderColor} boxShadow="sm">
            <Text color="gray.500" fontSize="xs" fontWeight="700" textTransform="uppercase" mb={2} letterSpacing="wider" alignSelf="flex-start">
              {t('status.memory')}
            </Text>
            <Box h="130px" w="100%" position="relative">
              <EChartsRadial percent={realtime?.memory?.usage_percent || 0} />
              <Text fontSize="xs" fontWeight="600" color="gray.400" textAlign="center" position="absolute" bottom="-10px" w="100%">
                {realtime?.memory?.used_mb}MB / {realtime?.memory?.total_mb}MB
              </Text>
            </Box>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={5} h="100%" display="flex" flexDirection="column" alignItems="center" justifyContent="center" variant="outline" border="1px solid" borderColor={borderColor} boxShadow="sm">
            <Text color="gray.500" fontSize="xs" fontWeight="700" textTransform="uppercase" mb={2} letterSpacing="wider" alignSelf="flex-start">
              {t('status.disks')}
            </Text>
            <Box h="130px" w="100%" position="relative">
              <EChartsRadial percent={diskUsagePercent} />
              <Text fontSize="xs" fontWeight="600" color="gray.400" textAlign="center" position="absolute" bottom="-10px" w="100%">
                {usedDiskGb.toFixed(1)}GB / {totalDiskGb.toFixed(1)}GB
              </Text>
            </Box>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={5} h="100%" display="flex" flexDirection="column" variant="outline" border="1px solid" borderColor={borderColor} boxShadow="sm">
            <Text color="gray.500" fontSize="xs" fontWeight="700" textTransform="uppercase" mb={4} letterSpacing="wider">
              {t('status.uptime')}
            </Text>
            <Flex align="center" justify="center" flex={1}>
              <Text fontSize="2xl" fontWeight="800" color={textColor} letterSpacing="tight">
                {formatUptime(realtime?.uptime_seconds || 0)}
              </Text>
            </Flex>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={5} h="100%" display="flex" flexDirection="column" variant="outline" border="1px solid" borderColor={borderColor} boxShadow="sm">
            <Text color="gray.500" fontSize="xs" fontWeight="700" textTransform="uppercase" mb={2} letterSpacing="wider">
              {t('status.tasks')}
            </Text>
            <Flex direction="column" align="center" justify="center" flex={1}>
              <Flex align="baseline">
                <Text fontSize="3xl" fontWeight="800" color="green.500">{realtime?.tasks?.running || 0}</Text>
                <Text fontSize="xl" fontWeight="600" color="gray.300" mx={2}>/</Text>
                <Text fontSize="3xl" fontWeight="800" color="red.500">{realtime?.tasks?.failed || 0}</Text>
              </Flex>
              <Text fontSize="xs" fontWeight="700" color="gray.400" mt={1}>
                {t('status.running').toUpperCase()} / {t('status.failed').toUpperCase()}
              </Text>
            </Flex>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={5} h="100%" display="flex" flexDirection="column" variant="outline" border="1px solid" borderColor={borderColor} boxShadow="sm">
            <Text color="gray.500" fontSize="xs" fontWeight="700" textTransform="uppercase" mb={2} letterSpacing="wider">
              SIP 服务
            </Text>
            <Flex direction="column" align="center" justify="center" flex={1}>
              <Text fontSize="3xl" fontWeight="800" color={sipOnlineCount > 0 ? 'green.500' : 'gray.400'}>{sipOnlineCount}</Text>
              <Text fontSize="xs" fontWeight="700" color="gray.400" mt={1}>
                已注册设备数
              </Text>
              <Text fontSize="xs" color="blue.500" mt={2} cursor="pointer" onClick={() => window.location.hash = '/admin/gb28181/devices'}>
                查看详情 →
              </Text>
            </Flex>
          </Card>
        </GridItem>
      </Grid>

      {/* CPU 和内存趋势图 */}
      <Grid templateColumns="repeat(2, 1fr)" gap={6} mb={6}>
        <GridItem>
          <Card p={4}>
            <Text fontWeight="600" mb={4} color={textColor}>{t('status.cpu')} {t('status.trend')}</Text>
            <Box h="200px">
              {cpuHistory.length > 0 ? (
                <ReactApexChart
                  options={cpuChartOptions}
                  series={cpuChartData}
                  type="area"
                  height="100%"
                />
              ) : (
                <Flex justify="center" align="center" h="100%">
                  <Text color="gray.500">{t('status.collecting_data')}</Text>
                </Flex>
              )}
            </Box>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={4}>
            <Text fontWeight="600" mb={4} color={textColor}>{t('status.memory')} {t('status.trend')}</Text>
            <Box h="200px">
              {memoryHistory.length > 0 ? (
                <ReactApexChart
                  options={memChartOptions}
                  series={memChartData}
                  type="area"
                  height="100%"
                />
              ) : (
                <Flex justify="center" align="center" h="100%">
                  <Text color="gray.500">{t('status.collecting_data')}</Text>
                </Flex>
              )}
            </Box>
          </Card>
        </GridItem>
      </Grid>

      {/* NPU (如果存在) */}
      {resources?.npu?.supported && (
        <Grid templateColumns="repeat(1, 1fr)" gap={6} mb={6}>
          <Card p={4} display="flex" flexDirection="column">
            <Text fontWeight="600" mb={4} color={textColor}>{t('status.npu')}</Text>
            <Flex align="center" justify="center" flex={1} gap={4}>
              <Box h="120px" w="120px" position="relative">
                <EChartsRadial percent={resources.npu.usage_percent} />
              </Box>
              <Box>
                <Text fontSize="sm" color="gray.500">{t('status.temperature')}</Text>
                <Text fontSize="2xl" fontWeight="700" color={resources.npu.temperature > 80 ? 'red.500' : textColor}>
                  {resources.npu.temperature}°C
                </Text>
              </Box>
            </Flex>
          </Card>
        </Grid>
      )}

      {/* 服务状态 */}
      <Grid templateColumns="repeat(2, 1fr)" gap={6} mb={6}>
        <GridItem>
          <Card p={4}>
            <Text fontWeight="600" mb={4} color={textColor}>{t('status.system_info')}</Text>
            <Flex direction="column" gap={3}>
              <Flex justify="space-between">
                <Text color="gray.500">{t('status.device_model')}</Text>
                <Text fontWeight="600">{serviceData?.device_model || '-'}</Text>
              </Flex>
              <Flex justify="space-between">
                <Text color="gray.500">{t('status.app_version')}</Text>
                <Text fontWeight="600">{serviceData?.versions?.app || '-'}</Text>
              </Flex>
              <Flex justify="space-between">
                <Text color="gray.500">{t('status.current_time')}</Text>
                <Text fontWeight="600">
                  {serviceData?.time?.current ? formatDateTime(new Date(serviceData.time.current)) : '-'}
                </Text>
              </Flex>
              <Flex justify="space-between">
                <Text color="gray.500">{t('status.timezone')}</Text>
                <Text fontWeight="600">{serviceData?.time?.timezone || '-'}</Text>
              </Flex>
            </Flex>
          </Card>
        </GridItem>

        <GridItem>
          <Card p={4}>
            <Text fontWeight="600" mb={4} color={textColor}>{t('status.services')}</Text>
            <Table variant="simple" size="sm">
              <Thead>
                <Tr>
                  <Th px={0} color="gray.500">{t('status.service_name')}</Th>
                  <Th px={0} color="gray.500">{t('status.service_status')}</Th>
                </Tr>
              </Thead>
              <Tbody>
                {serviceData?.services?.map((service, index) => {
                  const statusColor = getStatusColor(service.status);
                  const statusText = t(`status.status_values.${service.status}`, { defaultValue: service.status });
                  return (
                    <Tr key={index}>
                      <Td px={0} fontWeight="500">{t(`status.services_dict.${service.name}`, { defaultValue: service.name })}</Td>
                      <Td px={0}>
                        <Tooltip
                          label={service.message || statusText}
                          placement="top"
                          hasArrow
                          isDisabled={!service.message}
                        >
                          <Flex align="center" gap={2} cursor={service.message ? 'help' : 'default'}>
                            <Box w={2} h={2} borderRadius="full" bg={statusColor} />
                            <Text color={statusColor}>{statusText}</Text>
                          </Flex>
                        </Tooltip>
                      </Td>
                    </Tr>
                  );
                })}
              </Tbody>
            </Table>
          </Card>
        </GridItem>
      </Grid>
    </Box>
  );
}
