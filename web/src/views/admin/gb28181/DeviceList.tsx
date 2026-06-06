import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Button, HStack, VStack, Text, Badge, Input, Select,
  Table, Thead, Tbody, Tr, Th, Td,
  Drawer, DrawerBody, DrawerHeader, DrawerOverlay, DrawerContent, DrawerCloseButton,
  FormControl, FormLabel, useDisclosure, useToast, IconButton,
  Modal, ModalOverlay, ModalContent, ModalHeader, ModalBody, ModalFooter, ModalCloseButton,
  NumberInput, NumberInputField, NumberInputStepper, NumberIncrementStepper, NumberDecrementStepper,
  Flex, Spacer, Center, Spinner, useColorModeValue, Checkbox,
} from '@chakra-ui/react';
import { MdRefresh, MdDelete, MdEdit, MdVisibility, MdDeviceHub } from 'react-icons/md';
import Card from 'components/card/Card';
import Pagination from 'components/pagination/Pagination';
import {
  listGB28181Devices, getGB28181Device, updateGB28181Device, deleteGB28181Device,
  triggerCatalog, getCatalogTaskStatus, type GB28181Device, type GB28181DeviceUpdateRequest,
} from '../../../services/gb28181';
import { getAccessToken } from 'services/api';

export default function DeviceList() {
  const { t } = useTranslation('modules/gb28181');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const [devices, setDevices] = useState<GB28181Device[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(false);

  const { isOpen: isDetailOpen, onOpen: onDetailOpen, onClose: onDetailClose } = useDisclosure();
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure();
  const [selectedDevice, setSelectedDevice] = useState<GB28181Device | null>(null);
  const [editForm, setEditForm] = useState<GB28181DeviceUpdateRequest>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listGB28181Devices({ page, page_size: pageSize, keyword, status: statusFilter });
      setDevices(res.list || []);
      setTotal(res.total || 0);
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, keyword, statusFilter, toast]);

  useEffect(() => { fetchData(); }, [fetchData]);

  // WebSocket 订阅目录查询完成事件
  useEffect(() => {
    const token = getAccessToken();
    if (!token) return;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/v1/ws?token=${encodeURIComponent(token)}`);
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'gb28181_catalog_completed') {
          toast({
            title: t('devices.messages.catalogCompleted', { count: msg.payload?.channelCount || 0 }),
            status: msg.payload?.success ? 'success' : 'warning',
            duration: 3000,
          });
          fetchData();
        }
      } catch { /* 忽略非 JSON 消息 */ }
    };
    return () => ws.close();
  }, [fetchData, t, toast]);

  const handleViewDetail = async (id: string) => {
    try {
      const device = await getGB28181Device(id);
      setSelectedDevice(device);
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
      toast({ title: '更新成功', status: 'success', duration: 3000 });
      onEditClose();
      fetchData();
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('devices.messages.deleteConfirm'))) return;
    try {
      await deleteGB28181Device(id);
      toast({ title: '删除成功', status: 'success', duration: 3000 });
      fetchData();
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
              : status.error || '目录查询失败',
            status: status.status === 'completed' ? 'success' : 'error',
            duration: 3000,
          });
          fetchData();
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

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('devices.title')}</Text>
        <HStack spacing={2}>
          <Button leftIcon={<MdRefresh />} variant="outline" onClick={fetchData} isLoading={loading}>刷新</Button>
        </HStack>
      </Flex>

      <HStack mb={4} spacing={4}>
        <Input placeholder="搜索..." value={keyword} onChange={(e) => setKeyword(e.target.value)} maxW="300px" bg={bgCard} />
        <Select placeholder="状态筛选" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} maxW="200px" bg={bgCard}>
          <option value="online">{t('devices.status.online')}</option>
          <option value="offline">{t('devices.status.offline')}</option>
          <option value="registered">{t('devices.status.registered')}</option>
        </Select>
      </HStack>

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th>{t('devices.fields.deviceCode')}</Th>
                <Th>{t('devices.fields.manufacturer')}</Th>
                <Th>{t('devices.fields.model')}</Th>
                <Th>{t('devices.fields.status')}</Th>
                <Th>{t('devices.fields.channelCount')}</Th>
                <Th>{t('devices.fields.lastHeartbeatAt')}</Th>
                <Th textAlign="right">操作</Th>
              </Tr>
            </Thead>
            <Tbody>
              {loading ? (
                <Tr><Td colSpan={7}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : devices.length === 0 ? (
                <Tr><Td colSpan={7}><Center py="20px">暂无数据</Center></Td></Tr>
              ) : (
                devices.map((d) => (
                  <Tr key={d.id}>
                    <Td><Text fontSize="sm" color={textColor} fontFamily="mono">{d.device_code}</Text></Td>
                    <Td><Text fontSize="sm">{d.manufacturer || '-'}</Text></Td>
                    <Td><Text fontSize="sm">{d.model || '-'}</Text></Td>
                    <Td><Badge colorScheme={statusColor(d.status)} variant="solid">{d.status}</Badge></Td>
                    <Td><Text fontSize="sm">{d.channel_count}</Text></Td>
                    <Td><Text fontSize="sm">{d.last_heartbeat_at ? new Date(d.last_heartbeat_at).toLocaleString() : '-'}</Text></Td>
                    <Td textAlign="right">
                      <HStack justify="flex-end" spacing={1}>
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
          <Pagination page={page} pageSize={pageSize} total={total} onChange={setPage} onPageSizeChange={setPageSize} />
        </Box>
      </Card>

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
      <Modal isOpen={isEditOpen} onClose={onEditClose}>
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
                <Input type="password" placeholder="留空不修改" onChange={(e) => setEditForm({ ...editForm, sip_password: e.target.value })} />
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
            <Button variant="ghost" mr={3} onClick={onEditClose}>取消</Button>
            <Button colorScheme="brand" onClick={handleSaveEdit}>保存</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
