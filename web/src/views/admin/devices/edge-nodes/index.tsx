import { AddIcon } from '@chakra-ui/icons';
import {
  Box,
  Button,
  Center,
  Flex,
  HStack,
  Spinner,
  Text,
  SimpleGrid,
  Stat,
  StatLabel,
  StatNumber,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Card from 'components/card/Card';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useFilter } from 'hooks/useFilter';
import { useInfiniteScroll } from 'hooks/useInfiniteScroll';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { edgeNodeApi, type EdgeNodeCardSnapshot } from 'services/edgeNode';
import { edgeNodeMetricsApi, type OverviewStats } from 'services/edgeNodeMetrics';
import { useWebSocket } from 'hooks/useWebSocket';
import { AlgorithmDeployModal } from './components/AlgorithmDeployModal';
import EdgeNodeCreateModal from './components/EdgeNodeCreateModal';
import EdgeNodeCard from './components/EdgeNodeCard';

export default function EdgeNodeList() {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const emptyBg = useColorModeValue('white', 'gray.700');
  const toggleBg = useColorModeValue('rgba(0,0,0,0.02)', 'rgba(255,255,255,0.02)');
  const footerColor = useColorModeValue('secondaryGray.600', 'gray.500');
  const toast = useToast();
  const navigate = useNavigate();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure();
  const { isOpen: isDeployOpen, onOpen: onDeployOpen, onClose: onDeployClose } = useDisclosure();
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure();
  const [deleteTarget, setDeleteTarget] = useState<EdgeNodeCardSnapshot | null>(null);
  const [deployTarget, setDeployTarget] = useState<EdgeNodeCardSnapshot | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const lastRefreshTimeRef = useRef(0);

  // Overview stats
  const [overview, setOverview] = useState<OverviewStats | null>(null);
  const fetchOverview = useCallback(async () => {
    try {
      const data = await edgeNodeMetricsApi.getOverview();
      setOverview(data);
    } catch {
      // Silent — overview is a non-critical enhancement
    }
  }, []);

  // Card view density state
  const [compact, setCompact] = useState(false);
  const sortBy = filters.sortBy || 'default';

  const fetchNodes = useCallback((page: number, pageSize: number) => edgeNodeApi.list({
    page,
    page_size: pageSize,
    keyword: filters.keyword,
    status: filters.status,
  }), [filters]);

  const {
    list: nodes,
    total,
    initialLoading,
    loadingMore,
    reload: reloadNodes,
    setList: setNodes,
  } = useInfiniteScroll<EdgeNodeCardSnapshot>(fetchNodes, { pageSize: 24 });

  // 首次加载 & 筛选条件变更时重新拉取
  useEffect(() => {
    reloadNodes().catch(() => {
      toast({ title: t('message.loadFailed'), status: 'error' });
    });
  }, [searchTrigger, reloadNodes, toast, t]);

  // Fetch overview stats on mount
  useEffect(() => {
    fetchOverview();
  }, [fetchOverview]);

  // WebSocket real-time updates with throttling to prevent excessive refreshes
  const handleWsMessage = useCallback((msg: any) => {
    if (msg.type === 'edge-node-status') {
      const now = Date.now();
      // Throttle refreshes to at most once every 2 seconds
      if (now - lastRefreshTimeRef.current > 2000) {
        lastRefreshTimeRef.current = now;
        refresh();
        fetchOverview();
      }
    } else if (msg.type === 'edge-node-metrics') {
      // Update node metrics in-place for real-time display
      setNodes((prev) => {
        if (!prev || !prev.length) return prev;
        const payload = msg.payload || msg;
        const idx = prev.findIndex((n) => n.id === payload.node_id);
        if (idx === -1) return prev;
        const updated = [...prev];
        updated[idx] = {
          ...updated[idx],
          cpu_usage: payload.cpu_usage,
          memory_usage: payload.memory_usage,
          net_rx_speed: payload.net_rx_speed,
          net_tx_speed: payload.net_tx_speed,
        };
        return updated;
      });
    } else if (msg.type === 'edge-node-engine-metrics') {
      // Update engine & accelerator metrics in real-time
      setNodes((prev) => {
        if (!prev || !prev.length) return prev;
        const payload = msg.payload || msg;
        const idx = prev.findIndex((n) => n.id === payload.node_id);
        if (idx === -1) return prev;
        const updated = [...prev];
        updated[idx] = {
          ...updated[idx],
          active_stream_count: payload.active_stream_count,
          decode_slots_used: payload.decode_slots_used,
          encode_slots_used: payload.encode_slots_used,
          egress_bps: payload.egress_bps,
          accelerator_utilization: payload.accelerator_utilization,
          accelerator_metrics_valid: payload.accelerator_metrics_valid,
          metrics_received_at: payload.received_at,
        };
        return updated;
      });
    }
  }, [refresh, setNodes, fetchOverview]);
  const { send } = useWebSocket({
    onOpen: useCallback((_ws: WebSocket) => {
      // Subscriptions are handled in the useEffect below for cleaner lifecycle management
    }, []),
    onMessage: handleWsMessage,
  });

  useEffect(() => {
    send({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-status' } });
    send({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-metrics' } });
    send({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-engine-metrics' } });
    return () => {
      send({ type: 'unsubscribe', payload: { node_id: '*', topic: 'edge-node-status' } });
      send({ type: 'unsubscribe', payload: { node_id: '*', topic: 'edge-node-metrics' } });
      send({ type: 'unsubscribe', payload: { node_id: '*', topic: 'edge-node-engine-metrics' } });
    };
  }, [send]);

  const handleDelete = useCallback(async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await edgeNodeApi.delete(deleteTarget.id);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      setDeleteTarget(null);
      onDeleteClose();
      reloadNodes();
    } catch {
      toast({ title: t('message.deleteFailed'), status: 'error' });
    } finally {
      setIsDeleting(false);
    }
  }, [deleteTarget, toast, t, onDeleteClose, reloadNodes]);

  const handleDeploy = useCallback((node: EdgeNodeCardSnapshot) => {
    setDeployTarget(node);
    onDeployOpen();
  }, [onDeployOpen]);

  // Client-side sorting
  const sortedNodes = useMemo(() => {
    return [...nodes].sort((a, b) => {
      if (sortBy === 'cpu') {
        return (b.cpu_usage ?? 0) - (a.cpu_usage ?? 0);
      }
      if (sortBy === 'memory') {
        return (b.memory_usage ?? 0) - (a.memory_usage ?? 0);
      }
      if (sortBy === 'load') {
        return (b.current_load ?? 0) - (a.current_load ?? 0);
      }
      return 0; // default
    });
  }, [nodes, sortBy]);


  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      {initialLoading ? (
        <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>
      ) : (
        <Flex direction="column" gap="20px">
        {/* Header */}
        <Flex justify="space-between" align="center">
          <Text fontSize="2xl" fontWeight="bold" color={textColor}>
            {t('title')}
          </Text>
          <HStack spacing={3}>
            <HStack
              spacing="1px"
              border="1px solid"
              borderColor={borderColor}
              borderRadius="9px"
              p="2px"
              bg={toggleBg}
            >
              <Button
                size="sm"
                variant={!compact ? 'brand' : 'ghost'}
                borderRadius="7px"
                px="3"
                h="32px"
                onClick={() => setCompact(false)}
              >
                {t('view.normal')}
              </Button>
              <Button
                size="sm"
                variant={compact ? 'brand' : 'ghost'}
                borderRadius="7px"
                px="3"
                h="32px"
                onClick={() => setCompact(true)}
              >
                {t('view.compact')}
              </Button>
            </HStack>
            <Button
              h="36px"
              leftIcon={<AddIcon />}
              colorScheme="brand"
              onClick={onCreateOpen}
            >
              {t('actions.create')}
            </Button>
          </HStack>
        </Flex>

        {/* Overview Stats */}
        <SimpleGrid columns={{ base: 2, sm: 4 }} spacing="16px">
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewTotal')}</StatLabel>
              <StatNumber color={textColor} fontSize="2xl">{overview?.total ?? total}</StatNumber>
            </Stat>
          </Card>
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewOnline')}</StatLabel>
              <StatNumber color="green.500" fontSize="2xl">{overview?.online ?? '-'}</StatNumber>
            </Stat>
          </Card>
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewOffline')}</StatLabel>
              <StatNumber color="gray.500" fontSize="2xl">{overview?.offline ?? '-'}</StatNumber>
            </Stat>
          </Card>
          <Card p={4}>
            <Stat>
              <StatLabel color="gray.400" fontSize="sm">{t('overviewError')}</StatLabel>
              <StatNumber color="red.500" fontSize="2xl">{overview?.error ?? '-'}</StatNumber>
            </Stat>
          </Card>
        </SimpleGrid>

        {/* Filters */}
        <SearchBar
          filters={filters}
          onFilterChange={setFilter}
          onReset={resetFilters}
          onRefresh={refresh}
          selects={[
            {
              name: 'status',
              label: t('fields.status'),
              options: [
                { value: 'online', label: t('status.online') },
                { value: 'offline', label: t('status.offline') },
                { value: 'error', label: t('status.error') },
                { value: 'disabled', label: t('status.disabled') },
              ],
            },
            {
              name: 'sortBy',
              label: t('fields.sortBy'),
              options: [
                { value: 'default', label: t('sort.default') },
                { value: 'cpu', label: t('sort.cpu') },
                { value: 'memory', label: t('sort.memory') },
                { value: 'load', label: t('sort.load') },
              ],
            },
          ]}
        />

        {/* Card Grid */}
        <Box>
          {sortedNodes.length === 0 ? (
            <Center py="100px" bg={emptyBg} borderRadius="14px" border="1px solid" borderColor={borderColor}>
              <Text color={textColor}>{tCommon('noData')}</Text>
            </Center>
          ) : (
            <SimpleGrid columns={{ base: 1, md: 2, lg: 3, xl: compact ? 4 : 3 }} gap="16px">
              {sortedNodes.map((node) => (
                <EdgeNodeCard
                  key={node.id}
                  node={node}
                  compact={compact}
                  onClick={() => navigate(`/admin/devices/edge-nodes/${node.id}`)}
                  onDeployClick={(e) => {
                    e.stopPropagation();
                    handleDeploy(node);
                  }}
                />
              ))}
            </SimpleGrid>
          )}

          {/* 滚动加载指示器 */}
          {loadingMore && (
            <Center py="32px">
              <Spinner color="brand.500" size="md" />
            </Center>
          )}

          {/* 已加载全部提示 */}
          {!loadingMore && nodes.length > 0 && nodes.length >= total && (
            <Center py="24px">
              <Text fontSize="xs" color={footerColor}>
                {t('fields.node', { count: total })}
              </Text>
            </Center>
          )}
        </Box>

        </Flex>
      )}

        {/* Delete Confirmation */}
        <ConfirmDialog
          isOpen={isDeleteOpen}
          onClose={() => { setDeleteTarget(null); onDeleteClose(); }}
          onConfirm={handleDelete}
          title={tCommon('button.confirm')}
          message={t('message.deleteConfirm')}
          isLoading={isDeleting}
        />

        {/* Deploy Algorithm Modal */}
        {deployTarget && (
          <AlgorithmDeployModal
            isOpen={isDeployOpen}
            onClose={() => { setDeployTarget(null); onDeployClose(); }}
            node={deployTarget as any}
          />
        )}

        {/* Create Node Modal */}
        <EdgeNodeCreateModal
          isOpen={isCreateOpen}
          onClose={onCreateClose}
          onSuccess={() => reloadNodes()}
        />
    </Box>
  );
}
