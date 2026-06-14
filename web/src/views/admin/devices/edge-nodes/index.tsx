import { AddIcon, DeleteIcon, InfoIcon, ViewIcon } from '@chakra-ui/icons';
import {
  Box,
  Button,
  Center,
  Flex,
  HStack,
  IconButton,
  Spinner,
  Table,
  Tag,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { edgeNodeApi, type EdgeNode } from 'services/edgeNode';
import { useWebSocket } from 'hooks/useWebSocket';
import { AlgorithmDeployModal } from './components/AlgorithmDeployModal';
import EdgeNodeCreateModal from './components/EdgeNodeCreateModal';

const STATUS_COLORS: Record<string, string> = {
  online: 'green',
  offline: 'gray',
  error: 'red',
  disabled: 'orange',
};

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
  const [deleteTarget, setDeleteTarget] = useState<EdgeNode | null>(null);
  const [deployTarget, setDeployTarget] = useState<EdgeNode | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [lastRefreshTime, setLastRefreshTime] = useState(0);

  const fetchNodes = useCallback((page: number, pageSize: number) => edgeNodeApi.list({
    page,
    page_size: pageSize,
    keyword: filters.keyword,
    status: filters.status,
  }), [filters]);

  const { list: nodes, total, page, pageSize, initialLoading, pageLoading, load: loadNodes, changePage, changePageSize } = usePagination<EdgeNode>(fetchNodes);

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
    }
  }, [lastRefreshTime, refresh]);

  useWebSocket({ onMessage: handleWsMessage });

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

  const handleDeploy = useCallback((node: EdgeNode) => {
    setDeployTarget(node);
    onDeployOpen();
  }, [onDeployOpen]);

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
            <Button
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
          ]}
        />

        {/* Table */}
        <Card px="0px" pb="20px">
          <Box overflowX="auto">
            <Table variant="simple" color="gray.500" mb="24px">
              <Thead>
                <Tr>
                  <Th>{t('fields.name')}</Th>
                  <Th>{t('fields.status')}</Th>
                  <Th>{t('fields.currentLoad')}</Th>
                  <Th>{t('fields.platform')}</Th>
                  <Th>{t('fields.engineVersion')}</Th>
                  <Th>{t('fields.lastHeartbeat')}</Th>
                  <Th textAlign="right">{t('fields.actions')}</Th>
                </Tr>
              </Thead>
              <Tbody>
                {(pageLoading && nodes.length === 0) ? (
                  <Tr><Td colSpan={7}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
                ) : nodes.length === 0 ? (
                  <Tr><Td colSpan={7}><Center py="20px">{tCommon('noData')}</Center></Td></Tr>
                ) : (
                  nodes.map((node) => (
                    <Tr key={node.id}>
                      <Td>
                        <Text color={textColor} fontSize="sm" fontWeight="700">
                          {node.name}
                        </Text>
                      </Td>
                      <Td>
                        <Tag colorScheme={STATUS_COLORS[node.status] || 'gray'} variant="solid" size="sm">
                          {t(`status.${node.status}`)}
                        </Tag>
                      </Td>
                      <Td>
                        <Text fontSize="sm">
                          {node.current_load ?? 0}/{node.max_load ?? '-'}
                        </Text>
                      </Td>
                      <Td>
                        <Text fontSize="sm">{node.hardware_info?.platform || node.platform || '-'}</Text>
                      </Td>
                      <Td>
                        <Tag
                          colorScheme={node.engine_version ? 'green' : 'gray'}
                          size="sm"
                          variant="outline"
                        >
                          {node.engine_version || '-'}
                        </Tag>
                      </Td>
                      <Td>
                        <Text fontSize="sm">
                          {node.last_heartbeat
                            ? new Date(node.last_heartbeat).toLocaleString()
                            : '-'}
                        </Text>
                      </Td>
                      <Td textAlign="right">
                        <HStack justify="flex-end">
                          <IconButton
                            aria-label={t('actions.detail')}
                            icon={<InfoIcon />}
                            size="sm"
                            variant="ghost"
                            onClick={() => navigate(`/admin/devices/edge-nodes/${node.id}`)}
                          />
                          <IconButton
                            aria-label={t('actions.deployAlgo')}
                            icon={<ViewIcon />}
                            size="sm"
                            variant="ghost"
                            onClick={() => handleDeploy(node)}
                          />
                          <IconButton
                            aria-label={t('actions.delete')}
                            icon={<DeleteIcon />}
                            size="sm"
                            variant="ghost"
                            colorScheme="red"
                            onClick={() => {
                              setDeleteTarget(node);
                              onDeleteOpen();
                            }}
                          />
                        </HStack>
                      </Td>
                    </Tr>
                  ))
                )}
              </Tbody>
            </Table>
          </Box>

          {total > 0 && (
            <Box px="25px">
              <Pagination
                total={total}
                page={page}
                pageSize={pageSize}
                onChange={changePage}
                onPageSizeChange={changePageSize}
              />
            </Box>
          )}
        </Card>

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
            node={deployTarget}
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
