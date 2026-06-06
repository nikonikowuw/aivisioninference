import {
  Box,
  Button,
  Flex,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  useColorModeValue,
  IconButton,
  HStack,
  Center,
  Spinner,
  Badge,
  useToast,
  Checkbox,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  Select,
  useDisclosure,
  Icon,
  CheckboxGroup,
  Stack,
} from '@chakra-ui/react';
import {
  AddIcon,
  DeleteIcon,
  DownloadIcon,
  EditIcon,
} from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useRef, useState, useCallback } from 'react';
import { devicesApi, deviceGroupsApi, type Device, type DeviceGroup } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import Card from 'components/card/Card';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { MdSettings, MdVideocam } from 'react-icons/md';

const statusColor: Record<string, string> = {
  unknown: 'gray',
  online: 'green',
  offline: 'orange',
  error: 'red',
  disabled: 'gray',
};

export default function Devices() {
  const { t } = useTranslation('modules/devices');
  const { t: tCommon } = useTranslation('common');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const modalBg = useColorModeValue('white', 'navy.700');

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchDevices = useCallback((page: number, pageSize: number) => devicesApi.list({
    page,
    page_size: pageSize,
    keyword: filters.keyword,
    status: filters.status,
    access_type: filters.access_type,
    group_id: filters.group_id,
  }), [filters]);

  const {
    list: devices,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
  } = usePagination<Device>(fetchDevices);

  const [allGroups, setAllGroups] = useState<DeviceGroup[]>([]);
  useEffect(() => {
    deviceGroupsApi.list({ page: 1, page_size: 1000 }).then((d) => setAllGroups(d.list)).catch(() => {});
  }, []);

  const [editing, setEditing] = useState<Device | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | null>(null);
  const [isBatching, setIsBatching] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const importInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const [form, setForm] = useState({
    device_name: '',
    access_type: 'rtsp',
    rtsp_url: '',
    gb28181_device_id: '',
    gb28181_channel_id: '',
    username: '',
    password: '',
    manufacturer: '',
    model: '',
    firmware_version: '',
    location_desc: '',
    remark: '',
    group_ids: [] as string[],
  });

  const resetForm = (device?: Device) => {
    setEditing(device || null);
    setForm({
      device_name: device?.device_name || '',
      access_type: device?.access_type || 'rtsp',
      rtsp_url: device?.rtsp_url || '',
      gb28181_device_id: device?.gb28181_device_id || '',
      gb28181_channel_id: device?.gb28181_channel_id || '',
      username: device?.username || '',
      password: '',
      manufacturer: device?.manufacturer || '',
      model: device?.model || '',
      firmware_version: device?.firmware_version || '',
      location_desc: device?.location_desc || '',
      remark: device?.remark || '',
      group_ids: device?.groups?.map((g) => g.id) || [],
    });
    onOpen();
  };

  const openCreate = () => resetForm();
  const openEdit = (device: Device) => resetForm(device);

  const handleSave = async () => {
    try {
      if (editing) {
        await devicesApi.update(editing.id, form);
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await devicesApi.create(form as any);
        toast({ title: t('message.createSuccess'), status: 'success' });
      }
      onClose();
      refresh();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await devicesApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      refresh();
    } catch (err) {
      toast({ title: t('message.deleteFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  const handleTestConnection = async (device: Device) => {
    setIsTesting(true);
    try {
      const result = await devicesApi.test(device.id);
      toast({
        title: result.success ? t('message.testSuccess') : t('message.testFailed'),
        description: result.message,
        status: result.success ? 'success' : 'error',
      });
    } catch (err) {
      toast({ title: t('message.testError'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsTesting(false);
    }
  };

  const handleBatchDelete = async () => {
    setIsBatching(true);
    try {
      const result = await devicesApi.batchDelete(selectedIds);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      setSelectedIds([]);
      refresh();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await devicesApi.exportCsv(filters);
    } catch (err) {
      toast({ title: t('message.exportFailed'), status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  const handleImport = async (file?: File) => {
    if (!file) return;
    try {
      const result = await devicesApi.importCsv(file);
      toast({
        title: t('message.importDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      refresh();
    } catch (err) {
      toast({ title: t('message.importFailed'), status: 'error' });
    }
    if (importInputRef.current) importInputRef.current.value = '';
  };

  const pageIds = devices.map((d) => d.id);
  const selectedOnPage = pageIds.filter((id) => selectedIds.includes(id));
  const isAllSelected = pageIds.length > 0 && selectedOnPage.length === pageIds.length;
  const isIndeterminate = selectedOnPage.length > 0 && !isAllSelected;

  const toggleAll = () => {
    setSelectedIds((prev) => {
      const pageIds = devices.map((d) => d.id);
      const allSelected = pageIds.every((id) => prev.includes(id));
      if (allSelected) {
        return prev.filter((id) => !pageIds.includes(id));
      }
      return Array.from(new Set([...prev, ...pageIds]));
    });
  };

  const toggleOne = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((sid) => sid !== id) : [...prev, id]
    );
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <HStack spacing={2}>
          <Input
            ref={importInputRef}
            type="file"
            accept=".csv"
            display="none"
            onChange={(e) => handleImport(e.target.files?.[0])}
          />
          <Button variant="outline" onClick={() => importInputRef.current?.click()}>{t('actions.import')}</Button>
          <Button leftIcon={<DownloadIcon />} variant="outline" onClick={handleExport} isLoading={isExporting}>
            {t('actions.export')}
          </Button>
          <Button leftIcon={<AddIcon />} colorScheme="brand" onClick={openCreate}>
            {t('actions.create')}
          </Button>
        </HStack>
      </Flex>

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
              { value: 'unknown', label: t('status.unknown') },
              { value: 'online', label: t('status.online') },
              { value: 'offline', label: t('status.offline') },
              { value: 'error', label: t('status.error') },
              { value: 'disabled', label: t('status.disabled') },
            ]
          },
          {
            name: 'access_type',
            label: t('fields.accessType'),
            options: [
              { value: 'rtsp', label: 'RTSP' },
              { value: 'gb28181', label: 'GB28181' },
              { value: 'nvr_channel', label: 'NVR' },
            ]
          }
        ]}
      />

      {selectedIds.length > 0 && (
        <Flex
          bg={bgCard}
          p="4"
          borderRadius="lg"
          border="1px solid"
          borderColor={borderColor}
          mb="4"
          justify="space-between"
          align="center"
        >
          <Text fontSize="sm" color={textColor}>{tCommon('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <Button size="sm" colorScheme="red" onClick={() => setBatchAction('delete')}>{t('actions.batchDelete')}</Button>
          </HStack>
        </Flex>
      )}

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th pe="10px" w="48px">
                  <Checkbox isChecked={isAllSelected} isIndeterminate={isIndeterminate} onChange={toggleAll} />
                </Th>
                <Th>{t('fields.id')}</Th>
                <Th>{t('fields.deviceName')}</Th>
                <Th>{t('fields.accessType')}</Th>
                <Th>{t('fields.status')}</Th>
                <Th>{t('fields.manufacturer')}</Th>
                <Th>{t('fields.lastOnlineAt')}</Th>
                <Th textAlign="right">{t('fields.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {pageLoading ? (
                <Tr><Td colSpan={8}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : devices.length === 0 ? (
                <Tr><Td colSpan={8}><Center py="20px">{tCommon('noData')}</Center></Td></Tr>
              ) : (
                devices.map((device) => (
                  <Tr key={device.id}>
                    <Td pe="10px">
                      <Checkbox isChecked={selectedIds.includes(device.id)} onChange={() => toggleOne(device.id)} />
                    </Td>
                    <Td><Text fontSize="sm" color={textColor} fontFamily="mono">{device.id.slice(0, 8)}</Text></Td>
                    <Td>
                      <HStack>
                        <Icon as={MdVideocam} color="brand.500" />
                        <Text color={textColor} fontSize="sm" fontWeight="700">
                          {device.device_name}
                        </Text>
                      </HStack>
                    </Td>
                    <Td>
                      <Badge variant="outline" colorScheme="gray">{device.access_type.toUpperCase()}</Badge>
                    </Td>
                    <Td>
                      <Badge colorScheme={statusColor[device.status]} variant="solid">
                        {t(`status.${device.status}`)}
                      </Badge>
                    </Td>
                    <Td><Text fontSize="sm">{device.manufacturer || '-'}</Text></Td>
                    <Td><Text fontSize="sm">{device.last_online_at ? formatDateTime(device.last_online_at) : '-'}</Text></Td>
                    <Td textAlign="right">
                      <HStack justify="flex-end">
                        <IconButton aria-label={t('actions.edit')} icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(device)} />
                        <IconButton aria-label={t('actions.test')} icon={<MdSettings />} size="sm" variant="ghost" isLoading={isTesting} onClick={() => handleTestConnection(device)} />
                        <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(device.id)} />
                      </HStack>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>

        <Box px="25px">
          <Pagination
            page={page}
            pageSize={pageSize}
            total={total}
            onChange={changePage}
            onPageSizeChange={changePageSize}
          />
        </Box>
      </Card>

      {/* Modal & Dialogs */}
      <Modal isOpen={isOpen} onClose={onClose} size="xl">
        <ModalOverlay />
        <ModalContent bg={modalBg}>
          <ModalHeader color={textColor}>{editing ? t('actions.edit') : t('actions.create')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb="4" isRequired>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.deviceName')}</FormLabel>
              <Input variant="main" value={form.device_name} onChange={(e) => setForm({ ...form, device_name: e.target.value })} />
            </FormControl>
            <FormControl mb="4" isRequired>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.accessType')}</FormLabel>
              <Select variant="main" value={form.access_type} onChange={(e) => setForm({ ...form, access_type: e.target.value })}>
                <option value="rtsp">RTSP</option>
                <option value="gb28181">GB28181</option>
                <option value="nvr_channel">NVR</option>
              </Select>
            </FormControl>
            {form.access_type === 'rtsp' && (
              <FormControl mb="4" isRequired>
                <FormLabel fontSize="sm" fontWeight="700" color={textColor}>RTSP URL</FormLabel>
                <Input variant="main" value={form.rtsp_url} onChange={(e) => setForm({ ...form, rtsp_url: e.target.value })} placeholder="rtsp://user:pass@ip:port/path" />
              </FormControl>
            )}
            {form.access_type === 'gb28181' && (
              <FormControl mb="4" isRequired>
                <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.gb28181DeviceId')}</FormLabel>
                <Input variant="main" value={form.gb28181_device_id} onChange={(e) => setForm({ ...form, gb28181_device_id: e.target.value })} maxLength={20} />
              </FormControl>
            )}
            <Flex gap="4">
              <FormControl mb="4">
                <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.username')}</FormLabel>
                <Input variant="main" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
              </FormControl>
              <FormControl mb="4">
                <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.password')}</FormLabel>
                <Input variant="main" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder={editing ? t('fields.passwordPlaceholder') : ''} />
              </FormControl>
            </Flex>
            <Flex gap="4">
              <FormControl mb="4">
                <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.manufacturer')}</FormLabel>
                <Input variant="main" value={form.manufacturer} onChange={(e) => setForm({ ...form, manufacturer: e.target.value })} />
              </FormControl>
              <FormControl mb="4">
                <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.model')}</FormLabel>
                <Input variant="main" value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })} />
              </FormControl>
            </Flex>
            <FormControl mb="4">
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.deviceGroups')}</FormLabel>
              <CheckboxGroup
                colorScheme="brand"
                value={form.group_ids}
                onChange={(values) => setForm({ ...form, group_ids: values as string[] })}
              >
                <Stack spacing={[2, 4]} direction="row" wrap="wrap" p="4px">
                  {allGroups.map((group) => (
                    <Checkbox key={group.id} value={group.id} fontWeight="500" fontSize="sm">
                      {group.group_name}
                    </Checkbox>
                  ))}
                </Stack>
              </CheckboxGroup>
            </FormControl>
            <FormControl mb="4">
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('fields.remark')}</FormLabel>
              <Input variant="main" value={form.remark} onChange={(e) => setForm({ ...form, remark: e.target.value })} />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr="3" onClick={onClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleSave}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title={t('actions.delete')}
        message={t('message.deleteConfirm')}
        isLoading={isDeleting}
      />

      <ConfirmDialog
        isOpen={batchAction === 'delete'}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchDelete}
        title={t('actions.batchDelete')}
        message={t('message.batchDeleteConfirm', { count: selectedIds.length })}
        isLoading={isBatching}
      />
    </Box>
  );
}
