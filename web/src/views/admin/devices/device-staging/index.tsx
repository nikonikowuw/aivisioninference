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
  useDisclosure,
  Icon,
  VStack,
  Code,
  FormControl,
  FormLabel,
  Input,
  Switch,
} from '@chakra-ui/react';
import { DeleteIcon, DownloadIcon, ViewIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback, useMemo } from 'react';
import { deviceStagingApi, type DiscoveredDevice } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import Card from 'components/card/Card';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { MdOutlineDeviceHub, MdWifiFind } from 'react-icons/md';

const sourceColor: Record<string, string> = {
  onvif: 'blue',
  gb28181: 'purple',
  nvr: 'teal',
  scan: 'cyan',
};

const statusColor: Record<string, string> = {
  pending: 'yellow',
  imported: 'green',
  ignored: 'gray',
  expired: 'red',
};

const SourceBadge = ({ source, t }: { source: string; t: any }) => (
  <Badge colorScheme={sourceColor[source] || 'gray'}>
    {t(`source.${source}` as any) || source}
  </Badge>
);

const StatusBadge = ({ status, t }: { status: string; t: any }) => (
  <Badge colorScheme={statusColor[status] || 'gray'}>
    {t(`status.${status}` as any) || status}
  </Badge>
);

const DeviceDetailContent = ({ device, t }: { device: DiscoveredDevice; t: any }) => (
  <VStack align="stretch" spacing={4}>
    <Box>
      <Text fontSize="sm" fontWeight="bold" color="gray.500" mb={2}>
        {t('detail.basicInfo')}
      </Text>
      <Table size="sm" variant="simple">
        <Tbody>
          <Tr>
            <Td fontWeight="500">{t('fields.deviceName')}</Td>
            <Td>{device.device_name || '-'}</Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.source')}</Td>
            <Td>
              <SourceBadge source={device.source} t={t} />
            </Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.manufacturer')}</Td>
            <Td>{device.manufacturer || '-'}</Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.model')}</Td>
            <Td>{device.model || '-'}</Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.firmwareVersion')}</Td>
            <Td>{device.firmware_version || '-'}</Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.status')}</Td>
            <Td>
              <StatusBadge status={device.status} t={t} />
            </Td>
          </Tr>
        </Tbody>
      </Table>
    </Box>

    <Box>
      <Text fontSize="sm" fontWeight="bold" color="gray.500" mb={2}>
        {t('detail.networkInfo')}
      </Text>
      <Table size="sm" variant="simple">
        <Tbody>
          <Tr>
            <Td fontWeight="500">{t('fields.deviceIP')}</Td>
            <Td>
              <Code>{device.device_ip || '-'}</Code>
            </Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.deviceMAC')}</Td>
            <Td>
              <Code>{device.device_mac || '-'}</Code>
            </Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.accessType')}</Td>
            <Td>
              <Badge variant="outline">{device.access_type || '-'}</Badge>
            </Td>
          </Tr>
          <Tr>
            <Td fontWeight="500">{t('fields.accessURL')}</Td>
            <Td>
              <Code fontSize="xs" maxW="300px" overflow="hidden" textOverflow="ellipsis">
                {device.access_url || '-'}
              </Code>
            </Td>
          </Tr>
          {device.gb28181_code && (
            <Tr>
              <Td fontWeight="500">{t('fields.gb28181Code')}</Td>
              <Td>
                <Code>{device.gb28181_code}</Code>
              </Td>
            </Tr>
          )}
        </Tbody>
      </Table>
    </Box>

    {device.extra_info && Object.keys(device.extra_info).length > 0 && (
      <Box>
        <Text fontSize="sm" fontWeight="bold" color="gray.500" mb={2}>
          {t('detail.extraInfo')}
        </Text>
        <Code p={3} borderRadius="md" fontSize="xs" whiteSpace="pre-wrap" w="100%">
          {JSON.stringify(device.extra_info, null, 2)}
        </Code>
      </Box>
    )}
  </VStack>
);

export default function DeviceStaging() {
  const { t } = useTranslation('modules/device-staging');
  const { t: tCommon } = useTranslation('common');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchDevices = useCallback(
    (page: number, pageSize: number) =>
      deviceStagingApi.list({
        page,
        page_size: pageSize,
        keyword: filters.keyword,
        source: filters.source,
        status: filters.status,
      }),
    [filters],
  );

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
  } = usePagination<DiscoveredDevice>(fetchDevices);

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'import' | 'ignore' | null>(null);
  const [isBatching, setIsBatching] = useState(false);
  const [isScanning, setIsScanning] = useState(false);
  const [detailDevice, setDetailDevice] = useState<DiscoveredDevice | null>(null);
  const { isOpen: isDetailOpen, onOpen: onDetailOpen, onClose: onDetailClose } = useDisclosure();

  // 导入设备模态框状态
  const [importDeviceId, setImportDeviceId] = useState<string | null>(null);
  const [importDeviceName, setImportDeviceName] = useState('');
  const [importUsername, setImportUsername] = useState('');
  const [importPassword, setImportPassword] = useState('');
  const [importEnableInfer, setImportEnableInfer] = useState(true);
  const [isImporting, setIsImporting] = useState(false);
  const { isOpen: isImportOpen, onOpen: onImportOpen, onClose: onImportClose } = useDisclosure();

  // 批量导入模态框状态
  const [batchImportUsername, setBatchImportUsername] = useState('');
  const [batchImportPassword, setBatchImportPassword] = useState('');
  const [batchImportEnableInfer, setBatchImportEnableInfer] = useState(true);
  const { isOpen: isBatchImportOpen, onOpen: onBatchImportOpen, onClose: onBatchImportClose } = useDisclosure();

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const runAction = async (action: () => Promise<any>, successMsg: string, failMsg: string) => {
    try {
      await action();
      toast({ title: successMsg, status: 'success' });
      refresh();
    } catch (err) {
      toast({
        title: failMsg,
        description: err instanceof Error ? err.message : '',
        status: 'error',
      });
    }
  };

  const handleImport = (id: string) => {
    // 找到设备信息
    const device = devices.find(d => d.id === id);
    setImportDeviceId(id);
    setImportDeviceName(device?.device_name || '');
    setImportUsername('');
    setImportPassword('');
    setImportEnableInfer(true);
    onImportOpen();
  };

  const handleImportSubmit = async () => {
    if (!importDeviceId || !importUsername || !importPassword) {
      toast({ title: t('message.credentialsRequired'), status: 'warning' });
      return;
    }
    setIsImporting(true);
    try {
      await deviceStagingApi.import(importDeviceId, {
        username: importUsername,
        password: importPassword,
        device_name: importDeviceName,
        enable_infer: importEnableInfer,
      });
      toast({ title: t('message.importSuccess'), status: 'success' });
      onImportClose();
      refresh();
    } catch (err) {
      toast({
        title: t('message.importFailed'),
        description: err instanceof Error ? err.message : '',
        status: 'error',
      });
    } finally {
      setIsImporting(false);
    }
  };

  const handleIgnore = (id: string) =>
    runAction(() => deviceStagingApi.ignore(id), t('message.ignoreSuccess'), t('message.ignoreFailed'));

  const handleBatchAction = async () => {
    if (!batchAction || selectedIds.length === 0) return;
    
    if (batchAction === 'import') {
      // 打开批量导入模态框
      setBatchImportUsername('');
      setBatchImportPassword('');
      setBatchImportEnableInfer(true);
      onBatchImportOpen();
      return;
    }
    
    // 忽略操作直接执行
    setIsBatching(true);
    await runAction(
      () => deviceStagingApi.batchIgnore(selectedIds),
      t('message.batchIgnoreSuccess'),
      t('message.batchIgnoreFailed')
    );
    setSelectedIds([]);
    setIsBatching(false);
    setBatchAction(null);
  };

  const handleBatchImportSubmit = async () => {
    if (!batchImportUsername || !batchImportPassword) {
      toast({ title: t('message.credentialsRequired'), status: 'warning' });
      return;
    }
    setIsBatching(true);
    try {
      await deviceStagingApi.batchImport(selectedIds, {
        username: batchImportUsername,
        password: batchImportPassword,
        enable_infer: batchImportEnableInfer,
      });
      toast({ title: t('message.batchImportSuccess'), status: 'success' });
      onBatchImportClose();
      setSelectedIds([]);
      setBatchAction(null);
      refresh();
    } catch (err) {
      toast({
        title: t('message.batchImportFailed'),
        description: err instanceof Error ? err.message : '',
        status: 'error',
      });
    } finally {
      setIsBatching(false);
    }
  };

  const handleScanONVIF = async () => {
    setIsScanning(true);
    try {
      await deviceStagingApi.scanONVIF();
      toast({ title: t('message.scanONVIFStarted'), status: 'info' });
      // 延迟刷新，等待扫描完成
      setTimeout(() => refresh(), 3000);
    } catch (err) {
      toast({
        title: t('message.scanONVIFFailed'),
        description: err instanceof Error ? err.message : '',
        status: 'error',
      });
    } finally {
      setIsScanning(false);
    }
  };

  const openDetail = (device: DiscoveredDevice) => {
    setDetailDevice(device);
    onDetailOpen();
  };

  const { pageIds, isAllSelected, isIndeterminate } = useMemo(() => {
    const ids = devices.map((d) => d.id);
    const selectedCount = ids.filter((id) => selectedIds.includes(id)).length;
    return {
      pageIds: ids,
      isAllSelected: ids.length > 0 && selectedCount === ids.length,
      isIndeterminate: selectedCount > 0 && selectedCount < ids.length,
    };
  }, [devices, selectedIds]);

  const toggleAll = () => {
    setSelectedIds((prev) => 
      isAllSelected 
        ? prev.filter((id) => !pageIds.includes(id)) 
        : Array.from(new Set([...prev, ...pageIds]))
    );
  };

  const toggleOne = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((sid) => sid !== id) : [...prev, id],
    );
  };

  if (initialLoading) {
    return (
      <Center h="400px">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <HStack spacing={2}>
          <Icon as={MdOutlineDeviceHub} w={8} h={8} color="brand.500" />
          <Text fontSize="2xl" fontWeight="bold" color={textColor}>
            {t('title')}
          </Text>
        </HStack>
        <HStack spacing={2}>
          <Button
            leftIcon={<MdWifiFind />}
            variant="outline"
            onClick={handleScanONVIF}
            isLoading={isScanning}
            loadingText={t('actions.scanONVIF')}
          >
            {t('actions.scanONVIF')}
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
            name: 'source',
            label: t('fields.source'),
            options: [
              { value: 'onvif', label: t('source.onvif') },
              { value: 'gb28181', label: t('source.gb28181') },
              { value: 'nvr', label: t('source.nvr') },
              { value: 'scan', label: t('source.scan') },
            ],
          },
          {
            name: 'status',
            label: t('fields.status'),
            options: [
              { value: 'pending', label: t('status.pending') },
              { value: 'imported', label: t('status.imported') },
              { value: 'ignored', label: t('status.ignored') },
              { value: 'expired', label: t('status.expired') },
            ],
          },
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
          <Text fontSize="sm" color={textColor}>
            {tCommon('batch.selected', { count: selectedIds.length })}
          </Text>
          <HStack spacing={2}>
            <Button
              size="sm"
              colorScheme="green"
              leftIcon={<DownloadIcon />}
              onClick={() => setBatchAction('import')}
            >
              {t('actions.batchImport')}
            </Button>
            <Button
              size="sm"
              colorScheme="red"
              leftIcon={<DeleteIcon />}
              onClick={() => setBatchAction('ignore')}
            >
              {t('actions.batchIgnore')}
            </Button>
          </HStack>
        </Flex>
      )}

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          {devices.length === 0 ? (
            <VStack py="60px" spacing={4}>
              <Icon as={MdOutlineDeviceHub} w={12} h={12} color="gray.400" />
              <Text fontSize="lg" color="gray.500">
                {t('empty.title')}
              </Text>
              <Text fontSize="sm" color="gray.400">
                {t('empty.description')}
              </Text>
            </VStack>
          ) : (
            <>
              <Table variant="simple" color="gray.500" mb="24px">
                <Thead>
                  <Tr>
                    <Th pe="10px" w="48px">
                      <Checkbox
                        isChecked={isAllSelected}
                        isIndeterminate={isIndeterminate}
                        onChange={toggleAll}
                      />
                    </Th>
                    <Th>{t('fields.source')}</Th>
                    <Th>{t('fields.deviceName')}</Th>
                    <Th>{t('fields.deviceIP')}</Th>
                    <Th>{t('fields.manufacturer')}</Th>
                    <Th>{t('fields.model')}</Th>
                    <Th>{t('fields.accessType')}</Th>
                    <Th>{t('fields.status')}</Th>
                    <Th>{t('fields.createdAt')}</Th>
                    <Th>{t('fields.actions')}</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {devices.map((device) => (
                    <Tr key={device.id}>
                      <Td pe="10px">
                        <Checkbox
                          isChecked={selectedIds.includes(device.id)}
                          onChange={() => toggleOne(device.id)}
                        />
                      </Td>
                      <Td>
                        <SourceBadge source={device.source} t={t} />
                      </Td>
                      <Td>
                        <VStack align="start" spacing={0}>
                          <Text color={textColor} fontWeight="500">
                            {device.device_name || '-'}
                          </Text>
                          {device.gb28181_code && (
                            <Text fontSize="xs" color="gray.400">
                              SN: {device.gb28181_code}
                            </Text>
                          )}
                        </VStack>
                      </Td>
                      <Td>
                        <Code fontSize="sm">{device.device_ip || '-'}</Code>
                      </Td>
                      <Td>{device.manufacturer || '-'}</Td>
                      <Td>{device.model || '-'}</Td>
                      <Td>
                        <Badge variant="outline">{device.access_type || '-'}</Badge>
                      </Td>
                      <Td>
                        <StatusBadge status={device.status} t={t} />
                      </Td>
                      <Td>{formatDateTime(device.created_at)}</Td>
                      <Td>
                        <HStack spacing={1}>
                          <IconButton
                            aria-label={t('actions.viewDetail')}
                            icon={<ViewIcon />}
                            size="sm"
                            variant="ghost"
                            onClick={() => openDetail(device)}
                          />
                          {device.status === 'pending' && (
                            <>
                              <Button
                                size="sm"
                                colorScheme="green"
                                variant="ghost"
                                onClick={() => handleImport(device.id)}
                              >
                                {t('actions.import')}
                              </Button>
                              <Button
                                size="sm"
                                colorScheme="red"
                                variant="ghost"
                                onClick={() => handleIgnore(device.id)}
                              >
                                {t('actions.ignore')}
                              </Button>
                            </>
                          )}
                        </HStack>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>

              <Pagination
                total={total}
                page={page}
                pageSize={pageSize}
                onChange={changePage}
                onPageSizeChange={changePageSize}
              />
            </>
          )}
        </Box>
      </Card>

      {/* 批量操作确认对话框 */}
      <ConfirmDialog
        isOpen={!!batchAction}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchAction}
        title={batchAction === 'import' ? t('actions.batchImport') : t('actions.batchIgnore')}
        message={
          batchAction === 'import'
            ? t('message.batchImportConfirm', { count: selectedIds.length })
            : t('message.batchIgnoreConfirm', { count: selectedIds.length })
        }
        confirmText={tCommon('confirm')}
        cancelText={tCommon('cancel')}
        isLoading={isBatching}
        colorScheme={batchAction === 'import' ? 'green' : 'red'}
      />

      {/* 设备详情模态框 */}
      <Modal isOpen={isDetailOpen} onClose={onDetailClose} size="lg" isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('detail.title')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            {detailDevice && <DeviceDetailContent device={detailDevice} t={t} />}
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" onClick={onDetailClose}>
              {tCommon('close')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 单个设备导入模态框 */}
      <Modal isOpen={isImportOpen} onClose={onImportClose} size="md" isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('import.title')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>{t('fields.deviceName')}</FormLabel>
                <Input
                  value={importDeviceName}
                  onChange={(e) => setImportDeviceName(e.target.value)}
                  placeholder={t('import.deviceNamePlaceholder')}
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('fields.username')}</FormLabel>
                <Input
                  value={importUsername}
                  onChange={(e) => setImportUsername(e.target.value)}
                  placeholder={t('import.usernamePlaceholder')}
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('fields.password')}</FormLabel>
                <Input
                  type="password"
                  value={importPassword}
                  onChange={(e) => setImportPassword(e.target.value)}
                  placeholder={t('import.passwordPlaceholder')}
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb="0">{t('fields.enableInfer')}</FormLabel>
                <Switch
                  isChecked={importEnableInfer}
                  onChange={(e) => setImportEnableInfer(e.target.checked)}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onImportClose}>
              {tCommon('cancel')}
            </Button>
            <Button
              colorScheme="green"
              onClick={handleImportSubmit}
              isLoading={isImporting}
              loadingText={tCommon('loading')}
            >
              {tCommon('confirm')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* 批量导入模态框 */}
      <Modal isOpen={isBatchImportOpen} onClose={onBatchImportClose} size="md" isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('import.batchTitle', { count: selectedIds.length })}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              <Text fontSize="sm" color="gray.500">
                {t('import.batchDescription')}
              </Text>
              <FormControl isRequired>
                <FormLabel>{t('fields.username')}</FormLabel>
                <Input
                  value={batchImportUsername}
                  onChange={(e) => setBatchImportUsername(e.target.value)}
                  placeholder={t('import.usernamePlaceholder')}
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('fields.password')}</FormLabel>
                <Input
                  type="password"
                  value={batchImportPassword}
                  onChange={(e) => setBatchImportPassword(e.target.value)}
                  placeholder={t('import.passwordPlaceholder')}
                />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb="0">{t('fields.enableInfer')}</FormLabel>
                <Switch
                  isChecked={batchImportEnableInfer}
                  onChange={(e) => setBatchImportEnableInfer(e.target.checked)}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onBatchImportClose}>
              {tCommon('cancel')}
            </Button>
            <Button
              colorScheme="green"
              onClick={handleBatchImportSubmit}
              isLoading={isBatching}
              loadingText={tCommon('loading')}
            >
              {tCommon('confirm')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
