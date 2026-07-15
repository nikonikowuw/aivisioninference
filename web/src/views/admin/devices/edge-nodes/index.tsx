import { AddIcon, ViewIcon } from '@chakra-ui/icons';
import {
  Box,
  Button,
  Center,
  Flex,
  HStack,
  Spinner,
  Text,
  Select,
  SimpleGrid,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { edgeNodeApi, type EdgeNodeCardSnapshot } from 'services/edgeNode';
import { useWebSocket } from 'hooks/useWebSocket';
import { AlgorithmDeployModal } from './components/AlgorithmDeployModal';
import EdgeNodeCreateModal from './components/EdgeNodeCreateModal';
import EdgeNodeCard from './components/EdgeNodeCard';

export default function EdgeNodeList() {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const navigate = useNavigate();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure();
  const { isOpen: isDeployOpen, onOpen: onDeployOpen, onClose: onDeployClose } = useDisclosure();
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure();
  const [deleteTarget, setDeleteTarget] = useState<EdgeNodeCardSnapshot | null>(null);
  const [deployTarget, setDeployTarget] = useState<EdgeNodeCardSnapshot | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [lastRefreshTime, setLastRefreshTime] = useState(0);

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
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load: loadNodes,
    changePage,
    changePageSize,
    setList: setNodes,
  } = usePagination<EdgeNodeCardSnapshot>(fetchNodes);

  useEffect(() => {
    loadNodes({ page: 1 }).catch(() => {
      toast({ title: t('message.loadFailed'), status: 'error' });
    });
  }, [searchTrigger, loadNodes, toast, t]);

  // WebSocket real-time updates with throttling to prevent excessive refreshes
  const handleWsMessage = useCallback((msg: any) => {
    if (msg.type === 'edge-node-status') {
      const now = Date.now();
      // Throttle refreshes to at most once every 2 seconds
      if (now - lastRefreshTime > 2000) {
        setLastRefreshTime(now);
        refresh();
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
  }, [lastRefreshTime, refresh, setNodes]);

  const { send } = useWebSocket({
    onOpen: useCallback((ws: WebSocket) => {
      ws.send(JSON.stringify({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-status' } }));
      ws.send(JSON.stringify({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-engine-metrics' } }));
    }, []),
    onMessage: handleWsMessage,
  });

  useEffect(() => {
    send({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-status' } });
    send({ type: 'subscribe', payload: { node_id: '*', topic: 'edge-node-engine-metrics' } });
    return () => {
      send({ type: 'unsubscribe', payload: { node_id: '*', topic: 'edge-node-status' } });
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
      loadNodes({ page: page > 1 && nodes.length <= 1 ? page - 1 : page });
    } catch {
      toast({ title: t('message.deleteFailed'), status: 'error' });
    } finally {
      setIsDeleting(false);
    }
  }, [deleteTarget, toast, t, onDeleteClose, loadNodes, page, nodes.length]);

  const handleDeploy = useCallback((node: EdgeNodeCardSnapshot) => {
    setDeployTarget(node);
    onDeployOpen();
  }, [onDeployOpen]);

  // Client-side sorting
  const sortedNodes = [...nodes].sort((a, b) => {
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

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
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
              bg={useColorModeValue('rgba(0,0,0,0.02)', 'rgba(255,255,255,0.02)')}
            >
              <Button
                size="sm"
                variant={!compact ? 'brand' : 'ghost'}
                borderRadius="7px"
                px="3"
                h="32px"
                onClick={() => setCompact(false)}
              >
                {t('view.normal') || '卡片'}
              </Button>
              <Button
                size="sm"
                variant={compact ? 'brand' : 'ghost'}
                borderRadius="7px"
                px="3"
                h="32px"
                onClick={() => setCompact(true)}
              >
                {t('view.compact') || '紧凑'}
              </Button>
            </HStack>
            <Button
              h="36px"
              leftIcon={<ViewIcon />}
              variant="outline"
              onClick={() => navigate('/admin/devices/edge-nodes/overview')}
            >
              {t('overview')}
            </Button>
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
              label: t('fields.sortBy') || '排序方式',
              options: [
                { value: 'default', label: t('sort.default') || '默认排序' },
                { value: 'cpu', label: t('sort.cpu') || 'CPU 使用率' },
                { value: 'memory', label: t('sort.memory') || '内存使用率' },
                { value: 'load', label: t('sort.load') || '任务负载' },
              ],
            },
          ]}
        />

        {/* Card Grid */}
        <Box>
          {(pageLoading && sortedNodes.length === 0) ? (
            <Center py="100px"><Spinner color="brand.500" size="xl" /></Center>
          ) : sortedNodes.length === 0 ? (
            <Center py="100px" bg={useColorModeValue('white', 'gray.700')} borderRadius="14px" border="1px solid" borderColor={borderColor}>
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
        </Box>

        {/* Pagination */}
        {total > 0 && (
          <Box mt="10px">
            <Pagination
              total={total}
              page={page}
              pageSize={pageSize}
              onChange={changePage}
              onPageSizeChange={changePageSize}
            />
          </Box>
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
          onSuccess={() => loadNodes({ page: 1 })}
        />
      </Flex>
    </Box>
  );
}
