import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Button, HStack, VStack, Text, Badge,
  Table, Thead, Tbody, Tr, Th, Td,
  Drawer, DrawerBody, DrawerHeader, DrawerOverlay, DrawerContent, DrawerCloseButton,
  FormControl, FormLabel, useDisclosure, useToast, IconButton, Checkbox,
  Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter, ModalCloseButton,
  NumberInput, NumberInputField, NumberInputStepper, NumberIncrementStepper, NumberDecrementStepper,
  Flex, Center, Spinner, useColorModeValue, Input, Icon,
} from '@chakra-ui/react';
import { MdRefresh, MdDelete, MdEdit, MdVisibility, MdDeviceHub, MdAdd, MdViewList } from 'react-icons/md';
import Card from 'components/card/Card';
import { EmptyState } from 'components/empty/EmptyState';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useWebSocket } from 'hooks/useWebSocket';
import { WS_TOPIC } from 'constants/websocket';
import {
  listGB28181Devices, getGB28181Device, updateGB28181Device, deleteGB28181Device, createGB28181Device, batchDeleteGB28181Devices,
  listGB28181NVRs, getGB28181NVRChannels,
  triggerCatalog, getCatalogTaskStatus, type GB28181Device, type GB28181Channel, type GB28181DeviceUpdateRequest, type GB28181DeviceCreateRequest,
} from '../../../services/gb28181';
import { getAccessToken } from 'services/api';

const getStatusOptions = (t: any) => [
  { value: 'online', label: t('devices.status.online') },
  { value: 'offline', label: t('devices.status.offline') },
  { value: 'registered', label: t('devices.status.registered') },
];

export default function DeviceList() {
  const { t } = useTranslation('modules/gb28181');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchDevices = useCallback(
    (page: number, pageSize: number) =>
      listGB28181Devices({ page, page_size: pageSize, keyword: filters.keyword, status: filters.status }),
    [filters],
  );

  const {
    list: devices, total, page, pageSize,
    initialLoading, pageLoading, load, changePage, changePageSize,
  } = usePagination<GB28181Device>(fetchDevices);

  useEffect(() => { load({ page: 1 }); }, [searchTrigger, load]);

  const { isOpen: isDetailOpen, onOpen: onDetailOpen, onClose: onDetailClose } = useDisclosure();
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure();
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure();
  const [selectedDevice, setSelectedDevice] = useState<GB28181Device | null>(null);
  const [editForm, setEditForm] = useState<GB28181DeviceUpdateRequest>({});
  const [createForm, setCreateForm] = useState<GB28181DeviceCreateRequest>({ device_code: '', sip_domain: '' });
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | null>(null);
  const [selectedNVRId, setSelectedNVRId] = useState<string | null>(null);
  const [channels, setChannels] = useState<GB28181Channel[]>([]);
  const [channelsLoading, setChannelsLoading] = useState(false);

  // WebSocket 订阅目录查询完成事件（带自动重连）
  useWebSocket({
    onMessage: (msg: any) => {
      if (msg.type === WS_TOPIC.GB28181_CATALOG_COMPLETED) {
        toast({ title: t('devices.messages.catalogCompleted', { count: msg.payload?.channelCount || 0 }), status: msg.payload?.success ? 'success' : 'warning', duration: 3000 });
        refresh();
      }
    }
  });

  const handleViewDetail = async (id: string) => {
    try {
      setSelectedDevice(await getGB28181Device(id));
      onDetailOpen();
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const handleEdit = (device: GB28181Device) => {
    setSelectedDevice(device);
    setEditForm({ sip_id: device.sip_id, sip_domain: device.sip_domain, heartbeat_interval: device.heartbeat_interval });
    onEditOpen();
  };

  const handleSaveEdit = async () => {
    if (!selectedDevice) return;
    try {
      await updateGB28181Device(selectedDevice.id, editForm);
      toast({ title: t('common.updateSuccess'), status: 'success', duration: 3000 });
      onEditClose();
      refresh();
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const handleCreate = async () => {
    try {
      await createGB28181Device(createForm);
      toast({ title: t('common.createSuccess'), status: 'success', duration: 3000 });
      onCreateClose();
      setCreateForm({ device_code: '', sip_domain: '' });
      refresh();
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('devices.messages.deleteConfirm'))) return;
    try {
      await deleteGB28181Device(id);
      toast({ title: t('common.deleteSuccess'), status: 'success', duration: 3000 });
      refresh();
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const pageIds = devices.map(d => d.id);
  const selectedIdSet = new Set(selectedIds);
  const selectedOnPage = pageIds.filter(id => selectedIdSet.has(id));
  const isAllSelected = pageIds.length > 0 && selectedOnPage.length === pageIds.length;
  const isIndeterminate = selectedOnPage.length > 0 && !isAllSelected;

  const toggleAll = () => {
    setSelectedIds(prev => {
      const currentPageIds = new Set(pageIds);
      const allSelected = pageIds.every(id => prev.includes(id));
      if (allSelected) {
        return prev.filter(id => !currentPageIds.has(id));
      }
      return Array.from(new Set([...prev, ...pageIds]));
    });
  };

  const toggleOne = (id: string) => {
    setSelectedIds(prev => {
      if (prev.includes(id)) {
        return prev.filter(selectedId => selectedId !== id);
      }
      return [...prev, id];
    });
  };

  const handleBatchDelete = async () => {
    try {
      const result = await batchDeleteGB28181Devices(selectedIds);
      toast({ title: tCommon('batch.done', { success: result.success, failed: result.failed }), status: result.failed > 0 ? 'warning' : 'success', duration: 3000 });
      setSelectedIds([]);
      setBatchAction(null);
      refresh();
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const pollCatalogTask = async (taskId: string) => {
    for (let i = 0; i < 15; i += 1) {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      try {
        const status = await getCatalogTaskStatus(taskId);
        if (status.status === 'completed' || status.status === 'failed') {
          toast({
            title: status.status === 'completed'
              ? t('devices.messages.catalogCompleted', { count: status.channel_count })
              : status.error || t('common.catalogFailed'),
            status: status.status === 'completed' ? 'success' : 'error',
            duration: 3000,
          });
          refresh();
          return;
        }
      } catch { return; }
    }
  };

  const handleRefreshCatalog = async (id: string) => {
    try {
      const res = await triggerCatalog(id);
      toast({ title: t('devices.messages.catalogTriggered'), status: 'info', duration: 3000 });
      void pollCatalogTask(res.task_id);
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const statusColor = (status: string) => {
    switch (status) {
      case 'online': return 'green';
      case 'offline': return 'gray';
      case 'registered': return 'blue';
      default: return 'yellow';
    }
  };

  const handleViewChannels = async (nvrId: string) => {
    setSelectedNVRId(nvrId === selectedNVRId ? null : nvrId);
    if (nvrId === selectedNVRId) {
      setChannels([]);
      return;
    }
    setChannelsLoading(true);
    try {
      const data = await getGB28181NVRChannels(nvrId);
      setChannels(data || []);
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    } finally {
      setChannelsLoading(false);
    }
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('devices.title')}</Text>
        <HStack spacing={2}>
          <Button leftIcon={<MdAdd />} colorScheme="brand" onClick={onCreateOpen}>{t('devices.actions.create')}</Button>
          <Button leftIcon={<MdRefresh />} variant="outline" onClick={() => refresh()} isLoading={pageLoading}>{t('common.refresh')}</Button>
        </HStack>
      </Flex>

      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[{
          name: 'status',
          label: t('devices.fields.status'),
          options: getStatusOptions(t),
          placeholder: t('common.statusFilter'),
        }]}
      />

      {selectedIds.length > 0 && (
        <Flex bg={bgCard} p="4" borderRadius="lg" border="1px solid" borderColor={borderColor} mb="4" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{tCommon('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <Button size="sm" colorScheme="red" onClick={() => setBatchAction('delete')}>{t('devices.actions.batchDelete')}</Button>
          </HStack>
        </Flex>
      )}

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th pe="10px" w="48px"><Checkbox isChecked={isAllSelected} isIndeterminate={isIndeterminate} onChange={toggleAll} /></Th>
                <Th>{t('devices.fields.deviceCode')}</Th>
                <Th>{t('devices.fields.manufacturer')}</Th>
                <Th>{t('devices.fields.model')}</Th>
                <Th>{t('devices.fields.status')}</Th>
                <Th>{t('devices.fields.channelCount')}</Th>
                <Th>{t('devices.fields.lastHeartbeatAt')}</Th>
                <Th textAlign="right">{t('common.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {pageLoading ? (
                <Tr><Td colSpan={8}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : devices.length === 0 ? (
                <Tr><Td colSpan={8}><EmptyState /></Td></Tr>
              ) : (
                devices.map((d) => (
                  <Tr key={d.id}>
                    <Td pe="10px"><Checkbox isChecked={selectedIdSet.has(d.id)} onChange={() => toggleOne(d.id)} /></Td>
                    <Td><Text fontSize="sm" color={textColor} fontFamily="mono">{d.device_code}</Text></Td>
                    <Td><Text fontSize="sm">{d.manufacturer || '-'}</Text></Td>
                    <Td><Text fontSize="sm">{d.model || '-'}</Text></Td>
                    <Td><Badge colorScheme={statusColor(d.status)} variant="solid">{d.status}</Badge></Td>
                    <Td><Text fontSize="sm">{d.channel_count}</Text></Td>
                    <Td><Text fontSize="sm">{d.last_heartbeat_at ? new Date(d.last_heartbeat_at).toLocaleString() : '-'}</Text></Td>
                    <Td textAlign="right">
                      <HStack justify="flex-end" spacing={1}>
                        <IconButton aria-label="Channels" icon={<MdViewList />} size="sm" variant="ghost" colorScheme="teal" onClick={() => handleViewChannels(d.id)} title={t('devices.actions.viewChannels')} />
                        <IconButton aria-label="Detail" icon={<MdVisibility />} size="sm" variant="ghost" onClick={() => handleViewDetail(d.id)} />
                        <IconButton aria-label="Edit" icon={<MdEdit />} size="sm" variant="ghost" onClick={() => handleEdit(d)} />
                        <IconButton aria-label="Refresh" icon={<MdDeviceHub />} size="sm" variant="ghost" colorScheme="blue" onClick={() => handleRefreshCatalog(d.id)} />
                        <IconButton aria-label="Delete" icon={<MdDelete />} size="sm" variant="ghost" colorScheme="red" onClick={() => handleDelete(d.id)} />
                      </HStack>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>
        <Box px="25px">
          <Pagination page={page} pageSize={pageSize} total={total} onChange={changePage} onPageSizeChange={changePageSize} />
        </Box>
      </Card>

      {selectedNVRId && (
        <Card px="0px" pb="20px" mt={4}>
          <Box px="25px" pt="15px" pb="10px">
            <Text fontSize="lg" fontWeight="bold" color={textColor}>{t('channels.title')}</Text>
          </Box>
          <Box overflowX="auto">
            <Table variant="simple" color="gray.500" mb="24px">
              <Thead>
                <Tr>
                  <Th>{t('channels.fields.channelId')}</Th>
                  <Th>{t('channels.fields.channelName')}</Th>
                  <Th>{t('channels.fields.status')}</Th>
                  <Th>{t('channels.fields.manufacturer')}</Th>
                  <Th>{t('channels.fields.model')}</Th>
                </Tr>
              </Thead>
              <Tbody>
                {channelsLoading ? (
                  <Tr><Td colSpan={5}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
                ) : channels.length === 0 ? (
                  <Tr><Td colSpan={5}><EmptyState title={t('channels.noData')} /></Td></Tr>
                ) : (
                  channels.map((ch) => (
                    <Tr key={ch.id}>
                      <Td><Text fontSize="sm" fontFamily="mono" color={textColor}>{ch.gb28181_device_id || ch.id}</Text></Td>
                      <Td><Text fontSize="sm" fontWeight="700" color={textColor}>{ch.device_name}</Text></Td>
                      <Td><Badge colorScheme={statusColor(ch.status)} variant="solid">{ch.status}</Badge></Td>
                      <Td><Text fontSize="sm">{ch.manufacturer || '-'}</Text></Td>
                      <Td><Text fontSize="sm">{ch.model || '-'}</Text></Td>
                    </Tr>
                  ))
                )}
              </Tbody>
            </Table>
          </Box>
        </Card>
      )}

      {/* 详情抽屉 */}
      <Drawer isOpen={isDetailOpen} placement="right" onClose={onDetailClose} size="md">
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader>{t('devices.actions.viewDetail')}</DrawerHeader>
          <DrawerBody>
            {selectedDevice && (
              <VStack align="stretch" spacing={3}>
                <Text><strong>{t('devices.fields.deviceCode')}:</strong> {selectedDevice.device_code}</Text>
                <Text><strong>{t('devices.fields.sipId')}:</strong> {selectedDevice.sip_id}</Text>
                <Text><strong>{t('devices.fields.sipDomain')}:</strong> {selectedDevice.sip_domain}</Text>
                <Text><strong>{t('devices.fields.registerAddress')}:</strong> {selectedDevice.register_address}:{selectedDevice.register_port}</Text>
                <Text><strong>{t('devices.fields.status')}:</strong> <Badge colorScheme={statusColor(selectedDevice.status)}>{selectedDevice.status}</Badge></Text>
                <Text><strong>{t('devices.fields.channelCount')}:</strong> {selectedDevice.channel_count}</Text>
                <Text><strong>{t('devices.fields.manufacturer')}:</strong> {selectedDevice.manufacturer}</Text>
                <Text><strong>{t('devices.fields.model')}:</strong> {selectedDevice.model}</Text>
                <Text><strong>{t('devices.fields.firmware')}:</strong> {selectedDevice.firmware}</Text>
                <Text><strong>{t('devices.fields.heartbeatInterval')}:</strong> {selectedDevice.heartbeat_interval}s</Text>
                <Text><strong>{t('devices.fields.lastHeartbeatAt')}:</strong> {selectedDevice.last_heartbeat_at ? new Date(selectedDevice.last_heartbeat_at).toLocaleString() : '-'}</Text>
                <Text><strong>{t('devices.fields.lastCatalogAt')}:</strong> {selectedDevice.last_catalog_at ? new Date(selectedDevice.last_catalog_at).toLocaleString() : '-'}</Text>
              </VStack>
            )}
          </DrawerBody>
        </DrawerContent>
      </Drawer>

      {/* 编辑弹窗 */}
      <Modal isOpen={isEditOpen} onClose={onEditClose} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('devices.actions.edit')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl>
                <FormLabel>{t('devices.fields.sipId')}</FormLabel>
                <Input value={editForm.sip_id || ''} onChange={(e) => setEditForm({ ...editForm, sip_id: e.target.value })} />
              </FormControl>
              <FormControl>
                <FormLabel>{t('devices.fields.sipDomain')}</FormLabel>
                <Input value={editForm.sip_domain || ''} onChange={(e) => setEditForm({ ...editForm, sip_domain: e.target.value })} />
              </FormControl>
              <FormControl>
                <FormLabel>{t('devices.fields.sipPassword')}</FormLabel>
                <Input type="password" placeholder={t('devices.fields.passwordPlaceholder')} onChange={(e) => setEditForm({ ...editForm, sip_password: e.target.value })} />
              </FormControl>
              <FormControl>
                <FormLabel>{t('devices.fields.heartbeatInterval')}</FormLabel>
                <NumberInput value={editForm.heartbeat_interval || 60} min={10} max={300} onChange={(_, v) => setEditForm({ ...editForm, heartbeat_interval: v })}>
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onEditClose}>{t('common.cancel')}</Button>
            <Button colorScheme="brand" onClick={handleSaveEdit}>{t('common.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 创建弹窗 */}
      <Modal isOpen={isCreateOpen} onClose={onCreateClose} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('devices.actions.create')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('devices.fields.deviceCode')}</FormLabel>
                <Input value={createForm.device_code} onChange={(e) => setCreateForm({ ...createForm, device_code: e.target.value })} placeholder="34020000001320000001" />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('devices.fields.sipDomain')}</FormLabel>
                <Input value={createForm.sip_domain} onChange={(e) => setCreateForm({ ...createForm, sip_domain: e.target.value })} placeholder="3402000000" />
              </FormControl>
              <FormControl>
                <FormLabel>{t('devices.fields.sipId')} ({tCommon('optional')})</FormLabel>
                <Input value={createForm.sip_id || ''} onChange={(e) => setCreateForm({ ...createForm, sip_id: e.target.value })} placeholder={t('devices.fields.sipIdPlaceholder')} />
              </FormControl>
              <FormControl>
                <FormLabel>{t('devices.fields.sipPassword')}</FormLabel>
                <Input type="password" onChange={(e) => setCreateForm({ ...createForm, sip_password: e.target.value })} />
              </FormControl>
              <FormControl>
                <FormLabel>{t('devices.fields.heartbeatInterval')}</FormLabel>
                <NumberInput value={createForm.heartbeat_interval || 60} min={10} max={300} onChange={(_, v) => setCreateForm({ ...createForm, heartbeat_interval: v })}>
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onCreateClose}>{t('common.cancel')}</Button>
            <Button colorScheme="brand" onClick={handleCreate} isDisabled={!createForm.device_code || !createForm.sip_domain}>{t('common.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <ConfirmDialog
        isOpen={batchAction === 'delete'}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchDelete}
        title={t('devices.actions.batchDelete')}
        message={t('devices.messages.batchDeleteConfirm', { count: selectedIds.length })}
      />
    </Box>
  );
}
