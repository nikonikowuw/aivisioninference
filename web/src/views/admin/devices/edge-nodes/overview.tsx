import {
  Box,
  Center,
  Flex,
  SimpleGrid,
  Spinner,
  Stat,
  StatLabel,
  StatNumber,
  Text,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { edgeNodeMetricsApi, type OverviewStats } from 'services/edgeNodeMetrics';
import { edgeNodeApi, type EdgeNode } from 'services/edgeNode';
import { WS_TOPIC } from 'constants/websocket';
import { useWebSocket } from 'hooks/useWebSocket';

const STATUS_COLORS: Record<string, string> = {
  online: 'green.500',
  offline: 'gray.500',
  error: 'red.500',
  disabled: 'orange.400',
};

/**
 * EdgeNodeOverview — 边缘节点监控面板
 * 展示节点总览统计：在线/离线/错误节点数。
 */
export default function EdgeNodeOverview() {
  const { t } = useTranslation('modules/edge-nodes');
  const textColor = useColorModeValue('secondaryGray.900', 'white');
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
    } catch {
      setError(true);
      toast({ title: t('message.loadFailed'), status: 'error' });
    } finally {
      setLoading(false);
    }
  }, [toast, t]);

  useEffect(() => { fetchData(); }, [fetchData]);

  // WebSocket updates
  useWebSocket({
    onMessage: useCallback(
      (msg: any) => {
        if (msg.type === WS_TOPIC.EDGE_NODE_STATUS) fetchData();
      },
      [fetchData],
    ),
  });

  if (loading && !overview) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>
      </Box>
    );
  }

  if (error && !overview) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px"><Text>{t('message.loadFailed')}</Text></Center>
      </Box>
    );
  }

  // Fallback: compute from node list if backend overview is incomplete
  const stats: OverviewStats = overview || {
    total: nodes.length,
    online: nodes.filter((n) => n.status === 'online').length,
    offline: nodes.filter((n) => n.status === 'offline').length,
    error: nodes.filter((n) => n.status === 'error').length,
    alert_count: 0,
  };

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" gap="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>
          {t('overview')}
        </Text>

        <SimpleGrid columns={{ base: 1, sm: 2, md: 4 }} spacing="20px">
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewTotal')}</StatLabel>
              <StatNumber color={textColor} fontSize="2xl">{stats.total}</StatNumber>
            </Stat>
          </Card>
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewOnline')}</StatLabel>
              <StatNumber color={STATUS_COLORS.online} fontSize="2xl">{stats.online}</StatNumber>
            </Stat>
          </Card>
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewOffline')}</StatLabel>
              <StatNumber color={STATUS_COLORS.offline} fontSize="2xl">{stats.offline}</StatNumber>
            </Stat>
          </Card>
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewError')}</StatLabel>
              <StatNumber color={STATUS_COLORS.error} fontSize="2xl">{stats.error}</StatNumber>
            </Stat>
          </Card>
        </SimpleGrid>
      </Flex>
    </Box>
  );
}
