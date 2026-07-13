import {
  Box,
  Center,
  Flex,
  SimpleGrid,
  Spinner,
  Stat,
  StatHelpText,
  StatLabel,
  StatNumber,
  Text,
  useColorModeValue,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { edgeNodeApi, type NodeOverview } from 'services/edgeNode';

export default function EdgeNodeOverview() {
  const { t } = useTranslation('modules/edge-nodes');
  const textColor = useColorModeValue('navy.700', 'white');
  const textColorSecondary = useColorModeValue('gray.600', 'gray.400');

  const [overview, setOverview] = useState<NodeOverview | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchOverview = useCallback(async () => {
    setLoading(true);
    try {
      const data = await edgeNodeApi.getOverview();
      setOverview(data);
    } catch {
      // Silently fail; overview data is non-critical
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOverview();
  }, [fetchOverview]);

  if (loading) {
import { Box, Center, Flex, SimpleGrid, Spinner, Stat, StatLabel, StatNumber, Text, useColorModeValue, useToast } from '@chakra-ui/react';
import Card from 'components/card/Card';
import MetricsTimeSeries from 'components/charts/MetricsTimeSeries';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { edgeNodeMetricsApi, type OverviewStats } from 'services/edgeNodeMetrics';
import { edgeNodeApi, type EdgeNode } from 'services/edgeNode';
import { useWebSocket } from 'hooks/useWebSocket';

const STATUS_COLORS: Record<string, string> = {
  online: 'green.400',
  offline: 'gray.400',
  error: 'red.400',
  disabled: 'orange.400',
};

const STATUS_LABELS: Record<string, string> = {
  online: 'status.online',
  offline: 'status.offline',
  error: 'status.error',
  disabled: 'status.disabled',
};

/**
 * EdgeNodeOverview — 边缘节点监控面板
 * 展示节点总览统计：在线/离线/错误节点数、总任务数等。
 */
export default function EdgeNodeOverview() {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const [overview, setOverview] = useState<OverviewStats | null>(null);
  const [nodes, setNodes] = useState<EdgeNode[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(false);
      const [overviewData, nodeList] = await Promise.all([
        edgeNodeMetricsApi.getOverview(),
        edgeNodeApi.list({ page_size: 100 }),
      ]);
      setOverview(overviewData);
      setNodes(nodeList.list || []);
    } catch (err) {
      console.error('Failed to load overview:', err);
      setError(true);
      toast({ title: t('message.loadFailed'), status: 'error' });
    } finally {
      setLoading(false);
    }
  }, [toast, t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // WebSocket updates
  useWebSocket({
    onMessage: useCallback(
      (msg: any) => {
        if (msg.type === 'edge-node-status') {
          fetchData();
        }
      },
      [fetchData],
    ),
  });

  if (loading && !overview) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px">
          <Spinner size="xl" color="brand.500" />
        </Center>
      </Box>
    );
  }

  if (!overview) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px">
          <Text color={textColorSecondary}>{t('message.noData')}</Text>
  if (error && !overview) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px">
          <Text>{tCommon('message.loadFailed')}</Text>
        </Center>
      </Box>
    );
  }

  // Compute summary from node list (backend overview may not have all fields)
  const stats: OverviewStats = overview || {
    total: nodes.length,
    online: nodes.filter((n) => n.status === 'online').length,
    offline: nodes.filter((n) => n.status === 'offline').length,
    error: nodes.filter((n) => n.status === 'error').length,
    alert_count: 0,
  };

  // Status distribution for chart
  const statusDistribution = [
    { name: t('status.online'), value: stats.online, color: STATUS_COLORS.online },
    { name: t('status.offline'), value: stats.offline, color: STATUS_COLORS.offline },
    { name: t('status.error'), value: stats.error, color: STATUS_COLORS.error },
  ].filter((s) => s.value > 0);

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" gap="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>
          {t('overview')}
        </Text>

        <SimpleGrid columns={{ base: 1, sm: 2, md: 4 }} spacing="20px">
          {/* Total Nodes */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewTotal')}</StatLabel>
              <StatNumber fontSize="3xl" color={textColor}>
                {overview.total}
              </StatNumber>
              <StatHelpText>{t('fields.node')}</StatHelpText>
            </Stat>
          </Card>

          {/* Online */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewOnline')}</StatLabel>
              <StatNumber fontSize="3xl" color="green.500">
                {overview.online}
              </StatNumber>
              <StatHelpText color="green.500">{t('status.online')}</StatHelpText>
            </Stat>
          </Card>

          {/* Offline */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewOffline')}</StatLabel>
              <StatNumber fontSize="3xl" color="gray.500">
                {overview.offline}
              </StatNumber>
              <StatHelpText color="gray.500">{t('status.offline')}</StatHelpText>
            </Stat>
          </Card>

          {/* Error Count */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewError')}</StatLabel>
              <StatNumber fontSize="3xl" color="red.500">
                {overview.error_count}
              </StatNumber>
              <StatHelpText color="red.500">{t('status.error')}</StatHelpText>
          {t('overview.title')}
        </Text>

        {/* Stat Cards */}
        <SimpleGrid columns={{ base: 1, sm: 2, md: 4 }} spacing="20px">
          <Card px="20px" py="16px">
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">
                {t('overview.totalNodes')}
              </StatLabel>
              <StatNumber color={textColor} fontSize="2xl">
                {stats.total}
              </StatNumber>
            </Stat>
          </Card>
          <Card px="20px" py="16px">
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">
                {t('overview.onlineNodes')}
              </StatLabel>
              <StatNumber color={STATUS_COLORS.online} fontSize="2xl">
                {stats.online}
              </StatNumber>
            </Stat>
          </Card>
          <Card px="20px" py="16px">
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">
                {t('overview.offlineNodes')}
              </StatLabel>
              <StatNumber color={STATUS_COLORS.offline} fontSize="2xl">
                {stats.offline}
              </StatNumber>
            </Stat>
          </Card>
          <Card px="20px" py="16px">
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">
                {t('overview.errorNodes')}
              </StatLabel>
              <StatNumber color={STATUS_COLORS.error} fontSize="2xl">
                {stats.error}
              </StatNumber>
            </Stat>
          </Card>
        </SimpleGrid>
      </Flex>
    </Box>
  );
}
