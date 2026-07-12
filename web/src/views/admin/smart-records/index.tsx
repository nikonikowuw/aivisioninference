import { CheckIcon, DeleteIcon, DownloadIcon } from '@chakra-ui/icons';
import {
  Badge,
  Box,
  Button,
  Center,
  Checkbox,
  Code,
  Divider,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerHeader,
  DrawerOverlay,
  Flex,
  Grid,
  GridItem,
  HStack,
  Image,
  Spinner,
  Tab,
  Table,
  TabList,
  Tabs,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useToast,
  VStack,
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar, type InputConfig, type SelectConfig } from 'components/search-bar/SearchBar';
import { useAuth } from 'contexts/AuthContext';
import { useDateFormat } from 'hooks/useDateFormat';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import type { Device, DeviceGroup, Task } from 'services/api';
import { deviceGroupsApi, devicesApi, smartRecordsApi, tasksApi, type CategoryCodeOption, type SmartRecord, type SmartRecordType } from 'services/api';
import { hasPermission } from 'utils/permission';

const recordTypes: SmartRecordType[] = ['recognition', 'alarm', 'capture'];

const listPermissionByType: Record<SmartRecordType, string> = {
  recognition: 'records:recognition:list',
  alarm: 'records:alarm:list',
  capture: 'records:capture:list',
};

const statusColor: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'yellow',
  low: 'blue',
};

const alarmStatusColor: Record<string, string> = {
  unhandled: 'blue',
  handled: 'green',
};

function formatPercent(value?: number | null): string {
  if (value === undefined || value === null) return '-';
  return `${(value * 100).toFixed(1)}%`;
}

function RecordImage({ src, alt }: { src?: string; alt: string }) {
  const bg = useColorModeValue('gray.100', 'whiteAlpha.100');
  if (!src) {
    return (
      <Center w="72px" h="48px" bg={bg} borderRadius="8px">
        <Text fontSize="xs" color="gray.500">-</Text>
      </Center>
    );
  }
  return <Image src={src} alt={alt} boxSize="48px" minW="72px" objectFit="cover" borderRadius="8px" bg={bg} />;
}

export default function SmartRecords() {
  const { t } = useTranslation('modules/smart-records');
  const { t: tCommon } = useTranslation('common');
  const { user } = useAuth();
  const { formatDateTime } = useDateFormat();
  const toast = useToast();
  const [searchParams, setSearchParams] = useSearchParams();
  const [isExporting, setIsExporting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | null>(null);
  const [isBatching, setIsBatching] = useState(false);
  const [updatingAlarmStatusId, setUpdatingAlarmStatusId] = useState<string | null>(null);
  const [recordToDelete, setRecordToDelete] = useState<SmartRecord | null>(null);
  const [isDeletingSingle, setIsDeletingSingle] = useState(false);
  const [selectedDetailRecord, setSelectedDetailRecord] = useState<SmartRecord | null>(null);
  const [deviceOptions, setDeviceOptions] = useState<{ value: string; label: string }[]>([]);
  const [deviceGroupOptions, setDeviceGroupOptions] = useState<{ value: string; label: string }[]>([]);
  const [taskOptions, setTaskOptions] = useState<{ value: string; label: string }[]>([]);
  const [categoryCodeOptions, setCategoryCodeOptions] = useState<{ value: string; label: string }[]>([]);

  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('gray.600', 'gray.300');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const bgDrawer = useColorModeValue('white', 'navy.900');
  const bgRawCode = useColorModeValue('gray.50', 'navy.800');
  const emptyBg = useColorModeValue('gray.50', 'whiteAlpha.100');
  const emptyTextColor = useColorModeValue('gray.400', 'whiteAlpha.400');

  const permissionCodes = user?.permission_codes || [];
  const visibleTypes = useMemo(
    () => recordTypes.filter((type) => hasPermission(permissionCodes, listPermissionByType[type])),
    [permissionCodes],
  );

  const requestedType = searchParams.get('type') as SmartRecordType | null;
  const activeType = useMemo<SmartRecordType | null>(() => {
    if (requestedType && visibleTypes.includes(requestedType)) return requestedType;
    if (visibleTypes.includes('alarm')) return 'alarm';
    return visibleTypes[0] || null;
  }, [requestedType, visibleTypes]);

  const activeTabIndex = Math.max(0, activeType ? visibleTypes.indexOf(activeType) : 0);
  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  useEffect(() => {
    if (!activeType) return;
    if (requestedType !== activeType) {
      const next = new URLSearchParams(searchParams);
      next.set('type', activeType);
      setSearchParams(next, { replace: true });
    }
  }, [activeType, requestedType, searchParams, setSearchParams]);

  // 加载设备、设备分组和任务列表供下拉选择
  useEffect(() => {
    devicesApi.list({ page: 1, page_size: 500 }).then((res) => {
      setDeviceOptions(
        res.list.map((d: Device) => ({
          value: d.id,
          label: d.device_name,
        })),
      );
    }).catch(() => setDeviceOptions([]));

    deviceGroupsApi.list({ page: 1, page_size: 500 }).then((res) => {
      setDeviceGroupOptions(
        res.list.map((group: DeviceGroup) => ({
          value: group.id,
          label: group.group_name,
        })),
      );
    }).catch(() => setDeviceGroupOptions([]));

    tasksApi.list({ page: 1, page_size: 500 }).then((res) => {
      setTaskOptions(
        res.list.map((task: Task) => ({
          value: task.id,
          label: task.payload || task.id,
        })),
      );
    }).catch(() => setTaskOptions([]));
  }, []);

  // 加载类别编码列表供下拉选择
  useEffect(() => {
    smartRecordsApi.listCategoryCodes().then((opts) => {
      setCategoryCodeOptions(
        opts.map((o: CategoryCodeOption) => ({
          value: o.value.toString(),
          label: o.label,
        })),
      );
    }).catch(() => setCategoryCodeOptions([]));
  }, []);

  const fetchRecords = useCallback((p: number, ps: number) => {
    if (!activeType) return Promise.resolve({ list: [], total: 0 });
    return smartRecordsApi.list({
      page: p,
      page_size: ps,
      type: activeType,
      ...filters,
    });
  }, [activeType, filters]);

  const {
    list: records,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
  } = usePagination<SmartRecord>(fetchRecords);

  useEffect(() => {
    load({ page: 1 });
  }, [activeType, searchTrigger, load]);

  const baseSelectConfigs = useMemo<SelectConfig[]>(() => [
    {
      name: 'device_id',
      label: t('filters.deviceName'),
      placeholder: t('filters.devicePlaceholder'),
      options: [{ value: '', label: t('filters.allDevices') }, ...deviceOptions],
    },
    {
      name: 'group_id',
      label: t('filters.deviceGroup'),
      placeholder: t('filters.deviceGroupPlaceholder'),
      options: [{ value: '', label: t('filters.allDeviceGroups') }, ...deviceGroupOptions],
    },
    {
      name: 'task_id',
      label: t('filters.taskName'),
      placeholder: t('filters.taskPlaceholder'),
      options: [{ value: '', label: t('filters.allTasks') }, ...taskOptions],
    },
  ], [deviceGroupOptions, deviceOptions, taskOptions, t]);

  const inputConfigs = useMemo<InputConfig[]>(() => {
    if (activeType === 'recognition') {
      return [{ name: 'min_similarity', label: t('filters.minSimilarity') }];
    }
    if (activeType === 'capture') {
      return [{ name: 'min_confidence', label: t('filters.minConfidence') }];
    }
    return [];
  }, [activeType, t]);

  const selectConfigs = useMemo<SelectConfig[]>(() => {
    const alarmConfigs: SelectConfig[] = activeType === 'alarm'
      ? [
        {
          name: 'alarm_level',
          label: t('filters.alarmLevel'),
          options: ['critical', 'high', 'medium', 'low'].map((level) => ({ value: level, label: t(`alarmLevel.${level}`) })),
        },
        {
          name: 'alarm_type',
          label: t('filters.alarmType'),
          options: ['intrusion', 'cross_line', 'region', 'unknown'].map((type) => ({ value: type, label: t(`alarmType.${type}`) })),
        },
        {
          name: 'alarm_status',
          label: t('filters.alarmStatus'),
          options: ['unhandled', 'handled'].map((status) => ({ value: status, label: t(`status.${status}`) })),
        },
      ]
      : [];

    const captureConfigs: SelectConfig[] = activeType === 'capture'
      ? [{
        name: 'category_code',
        label: t('filters.categoryCode'),
        placeholder: t('filters.allCategories'),
        options: [{ value: '', label: t('filters.allCategories') }, ...categoryCodeOptions],
      }]
      : [];

    return [...baseSelectConfigs, ...alarmConfigs, ...captureConfigs];
  }, [activeType, baseSelectConfigs, categoryCodeOptions, t]);

  const columnCount = useMemo(() => {
    const baseColumns = 7; // 含 Checkbox, Snapshot, Target, Device, Time, Task + 1列操作列
    if (activeType === 'recognition') return baseColumns + 3;
    if (activeType === 'capture') return baseColumns + 2;
    if (activeType === 'alarm') return baseColumns + 3;
    return baseColumns;
  }, [activeType]);

  const handleTabChange = (index: number) => {
    const nextType = visibleTypes[index];
    if (!nextType) return;
    const next = new URLSearchParams(searchParams);
    next.set('type', nextType);
    setSearchParams(next);
    resetFilters();
  };

  const handleExport = async () => {
    if (!activeType || !hasPermission(permissionCodes, 'records:export')) return;
    setIsExporting(true);
    try {
      if (selectedIds.length > 0) {
        await smartRecordsApi.exportSelected(selectedIds);
      } else {
        await smartRecordsApi.exportCsv({ type: activeType, ...filters });
      }
    } catch (err) {
      toast({ title: tCommon('message.exportFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  const handleAlarmStatusUpdate = async (record: SmartRecord) => {
    if (!hasPermission(permissionCodes, 'records:alarm:update_status')) return;
    setUpdatingAlarmStatusId(record.record_id);
    try {
      await smartRecordsApi.updateAlarmStatus(record.record_id, 'handled');
      toast({ title: tCommon('message.updateSuccess'), status: 'success' });
      refresh();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setUpdatingAlarmStatusId(null);
    }
  };

  const pageIds = records.map((record) => record.record_id);
  const selectedIdsSet = new Set(selectedIds);
  const selectedOnPage = pageIds.filter((id) => selectedIdsSet.has(id));
  const isAllSelected = pageIds.length > 0 && selectedOnPage.length === pageIds.length;
  const isIndeterminate = selectedOnPage.length > 0 && !isAllSelected;

  const toggleAll = () => {
    setSelectedIds((prev) => {
      const prevSet = new Set(prev);
      const pageIdsSet = new Set(pageIds);
      const allSelected = pageIds.every((id) => prevSet.has(id));
      if (allSelected) {
        return prev.filter((id) => !pageIdsSet.has(id));
      }
      return Array.from(new Set([...prev, ...pageIds]));
    });
  };

  const toggleOne = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((sid) => sid !== id) : [...prev, id]
    );
  };

  const handleBatchDelete = async () => {
    if (selectedIds.length === 0) return;
    setIsBatching(true);
    try {
      const result = await smartRecordsApi.batchDelete(selectedIds);
      toast({ title: tCommon('message.batchDeleteSuccess', { count: result.success }), status: 'success' });
      setSelectedIds([]);
      refresh();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  const handleSingleDelete = async () => {
    if (!recordToDelete) return;
    setIsDeletingSingle(true);
    try {
      await smartRecordsApi.batchDelete([recordToDelete.record_id]);
      toast({ title: tCommon('message.deleteSuccess'), status: 'success' });
      setRecordToDelete(null);
      refresh();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsDeletingSingle(false);
    }
  };

  if (visibleTypes.length === 0) {
    return (
      <Center h="400px">
        <Text color={mutedColor}>{t('empty.noPermission')}</Text>
      </Center>
    );
  }

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align={{ base: 'flex-start', md: 'center' }} direction={{ base: 'column', md: 'row' }} gap={3} mb="20px">
        <Box>
          <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
          <Text fontSize="sm" color={mutedColor}>{t('subtitle')}</Text>
        </Box>
        {hasPermission(permissionCodes, 'records:export') && (
          <Button leftIcon={<DownloadIcon />} variant="outline" onClick={handleExport} isLoading={isExporting}>
            {selectedIds.length > 0 ? tCommon('button.exportSelected', { count: selectedIds.length }) : tCommon('button.export')}
          </Button>
        )}
      </Flex>

      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} p={4} mb={4}>
        <Tabs index={activeTabIndex} onChange={handleTabChange} colorScheme="brand">
          <TabList overflowX="auto" overflowY="hidden">
            {visibleTypes.map((type) => (
              <Tab key={type} whiteSpace="nowrap">{t(`tabs.${type}`)}</Tab>
            ))}
          </TabList>
        </Tabs>
      </Box>

      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        inputs={inputConfigs}
        selects={selectConfigs.length > 0 ? selectConfigs : undefined}
        dateRange
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
            <Button size="sm" colorScheme="red" leftIcon={<DeleteIcon />} onClick={() => setBatchAction('delete')}>
              {t('actions.batchDelete')}
            </Button>
          </HStack>
        </Flex>
      )}

      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="1040px">
          <Thead>
            <Tr>
              <Th pe="10px" w="48px">
                <Checkbox isChecked={isAllSelected} isIndeterminate={isIndeterminate} onChange={toggleAll} />
              </Th>
              <Th>{t('table.snapshot')}</Th>
              <Th>{t('table.target')}</Th>
              <Th>{t('table.deviceName')}</Th>
              <Th>{t('table.captureTime')}</Th>
              {activeType === 'recognition' && <Th>{t('table.person')}</Th>}
              {activeType === 'recognition' && <Th>{t('table.personImage')}</Th>}
              {activeType === 'recognition' && <Th>{t('table.similarity')}</Th>}
              {activeType === 'alarm' && <Th>{t('table.alarmType')}</Th>}
              {activeType === 'alarm' && <Th>{t('table.alarmLevel')}</Th>}
              {activeType === 'alarm' && <Th>{t('table.status')}</Th>}
              {activeType === 'capture' && <Th>{t('table.category')}</Th>}
              {activeType === 'capture' && <Th>{t('table.confidence')}</Th>}
              <Th>{t('table.taskName')}</Th>
              <Th textAlign="right">{t('table.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {records.map((record) => (
              <Tr key={`${record.record_id}-${record.capture_time}`}>
                <Td pe="10px">
                  <Checkbox isChecked={selectedIds.includes(record.record_id)} onChange={() => toggleOne(record.record_id)} />
                </Td>
                <Td><RecordImage src={record.snapshot_image_url || record.background_image_url} alt={t('table.snapshot')} /></Td>
                <Td><RecordImage src={record.target_crop_url} alt={t('table.target')} /></Td>
                <Td>{record.device_name || '-'}</Td>
                <Td>{formatDateTime(record.capture_time)}</Td>
                {activeType === 'recognition' && <Td>{record.person_name || '-'}</Td>}
                {activeType === 'recognition' && <Td><RecordImage src={record.person_image_url} alt={t('table.personImage')} /></Td>}
                {activeType === 'recognition' && <Td>{formatPercent(record.similarity)}</Td>}
                {activeType === 'alarm' && <Td>{record.alarm_type ? t(`alarmType.${record.alarm_type}`, { defaultValue: record.alarm_type }) : '-'}</Td>}
                {activeType === 'alarm' && (
                  <Td>
                    {record.alarm_level ? (
                      <Badge colorScheme={statusColor[record.alarm_level] || 'gray'}>
                        {t(`alarmLevel.${record.alarm_level}`, { defaultValue: record.alarm_level })}
                      </Badge>
                    ) : '-'}
                  </Td>
                )}
                {activeType === 'alarm' && (
                  <Td>
                    <Badge colorScheme={alarmStatusColor[record.alarm_status || 'unhandled'] || 'gray'}>
                      {t(`status.${record.alarm_status || 'unhandled'}`, { defaultValue: record.alarm_status || 'unhandled' })}
                    </Badge>
                  </Td>
                )}
                {activeType === 'capture' && <Td>{record.category_name || record.category_code?.toString() || '-'}</Td>}
                {activeType === 'capture' && <Td>{formatPercent(record.confidence)}</Td>}
                <Td>{record.task_name || '-'}</Td>
                <Td textAlign="right">
                  <HStack spacing={2} justify="flex-end">
                    {activeType === 'alarm' &&
                      hasPermission(permissionCodes, 'records:alarm:update_status') &&
                      (record.alarm_status || 'unhandled') === 'unhandled' && (
                        <Button
                          size="sm"
                          colorScheme="green"
                          leftIcon={<CheckIcon />}
                          onClick={() => handleAlarmStatusUpdate(record)}
                          isLoading={updatingAlarmStatusId === record.record_id}
                        >
                          {t('actions.markHandled')}
                        </Button>
                      )}
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => setSelectedDetailRecord(record)}
                    >
                      {t('actions.viewDetail')}
                    </Button>
                    {hasPermission(permissionCodes, 'records:batch-delete') && (
                      <Button
                        size="sm"
                        colorScheme="red"
                        variant="outline"
                        leftIcon={<DeleteIcon />}
                        onClick={() => setRecordToDelete(record)}
                      >
                        {tCommon('button.delete') || '删除'}
                      </Button>
                    )}
                  </HStack>
                </Td>
              </Tr>
            ))}
            {records.length === 0 && (
              <Tr>
                <Td colSpan={columnCount}>
                  <Center py={10}><Text color={mutedColor}>{t('empty.noData')}</Text></Center>
                </Td>
              </Tr>
            )}
          </Tbody>
        </Table>
      </Box>

      <Pagination
        page={page}
        pageSize={pageSize}
        total={total}
        onChange={changePage}
        onPageSizeChange={changePageSize}
        isLoading={pageLoading}
      />

      <ConfirmDialog
        isOpen={batchAction === 'delete'}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchDelete}
        title={t('actions.batchDelete')}
        message={t('message.batchDeleteConfirm', { count: selectedIds.length })}
        isLoading={isBatching}
      />

      {/* 单条记录删除确认弹窗 */}
      <ConfirmDialog
        isOpen={!!recordToDelete}
        onClose={() => setRecordToDelete(null)}
        onConfirm={handleSingleDelete}
        title={tCommon('button.delete')}
        message={tCommon('message.confirmDelete')}
        isLoading={isDeletingSingle}
      />

      {/* 详情抽屉 */}
      <Drawer isOpen={!!selectedDetailRecord} placement="right" onClose={() => setSelectedDetailRecord(null)} size="lg">
        <DrawerOverlay />
        <DrawerContent bg={bgDrawer}>
          <DrawerCloseButton />
          <DrawerHeader borderBottomWidth="1px" borderColor={borderColor}>
            {t('details.title') || '智能记录详情'}
          </DrawerHeader>

          <DrawerBody py={6}>
            {selectedDetailRecord && (
              <VStack align="stretch" spacing={6}>
                {/* 大图排版区域 */}
                <VStack align="stretch" spacing={3}>
                  <Text fontWeight="bold" fontSize="sm" color={textColor}>
                    {t('details.images') || '图像预览'}
                  </Text>

                  {/* 快照背景图 */}
                  {(selectedDetailRecord.snapshot_image_url || selectedDetailRecord.background_image_url) ? (
                    <Box borderRadius="lg" overflow="hidden" border="1px solid" borderColor={borderColor}>
                      <Image
                        src={selectedDetailRecord.snapshot_image_url || selectedDetailRecord.background_image_url}
                        alt="Snapshot"
                        w="100%"
                        maxH="300px"
                        objectFit="contain"
                        bg="black"
                      />
                    </Box>
                  ) : (
                    <Center h="150px" bg={emptyBg} borderRadius="lg">
                      <Text fontSize="sm" color={emptyTextColor}>{t('details.noSnapshot') || '无快照图像'}</Text>
                    </Center>
                  )}

                  {/* 目标抠图 & 人员匹配图 */}
                  <Grid templateColumns={selectedDetailRecord.record_type === 'recognition' ? 'repeat(2, 1fr)' : '1fr'} gap={3}>
                    <GridItem>
                      <Text fontSize="xs" color={mutedColor} mb={1}>
                        {t('table.target')}
                      </Text>
                      {selectedDetailRecord.target_crop_url ? (
                        <Box borderRadius="md" overflow="hidden" border="1px solid" borderColor={borderColor} bg="black" h="120px" display="flex" alignItems="center" justifyContent="center">
                          <Image src={selectedDetailRecord.target_crop_url} alt="Target" maxH="120px" objectFit="contain" />
                        </Box>
                      ) : (
                        <Center h="120px" bg={emptyBg} borderRadius="md" border="1px dashed" borderColor={borderColor}>
                          <Text fontSize="xs" color={emptyTextColor}>-</Text>
                        </Center>
                      )}
                    </GridItem>

                    {selectedDetailRecord.record_type === 'recognition' && (
                      <GridItem>
                        <Text fontSize="xs" color={mutedColor} mb={1}>
                          {t('table.personImage')}
                        </Text>
                        {selectedDetailRecord.person_image_url ? (
                          <Box borderRadius="md" overflow="hidden" border="1px solid" borderColor={borderColor} bg="black" h="120px" display="flex" alignItems="center" justifyContent="center">
                            <Image src={selectedDetailRecord.person_image_url} alt="Person" maxH="120px" objectFit="contain" />
                          </Box>
                        ) : (
                          <Center h="120px" bg={emptyBg} borderRadius="md" border="1px dashed" borderColor={borderColor}>
                            <Text fontSize="xs" color={emptyTextColor}>-</Text>
                          </Center>
                        )}
                      </GridItem>
                    )}
                  </Grid>
                </VStack>

                <Divider />

                {/* 属性信息面板 */}
                <VStack align="stretch" spacing={3}>
                  <Text fontWeight="bold" fontSize="sm" color={textColor}>
                    {t('details.basicInfo') || '基本信息'}
                  </Text>

                  <Grid templateColumns="repeat(2, 1fr)" gap={4} fontSize="sm">
                    <GridItem>
                      <Text color={mutedColor}>{t('fields.recordId')}</Text>
                      <Text fontWeight="medium" wordBreak="break-all">{selectedDetailRecord.record_id}</Text>
                    </GridItem>
                    <GridItem>
                      <Text color={mutedColor}>{t('table.captureTime')}</Text>
                      <Text fontWeight="medium">{formatDateTime(selectedDetailRecord.capture_time)}</Text>
                    </GridItem>
                    <GridItem>
                      <Text color={mutedColor}>{t('fields.deviceName')}</Text>
                      <Text fontWeight="medium">{selectedDetailRecord.device_name || '-'}</Text>
                    </GridItem>
                    <GridItem>
                      <Text color={mutedColor}>{t('fields.taskName')}</Text>
                      <Text fontWeight="medium">{selectedDetailRecord.task_name || '-'}</Text>
                    </GridItem>
                    {selectedDetailRecord.algorithm_name && (
                      <GridItem colSpan={2}>
                        <Text color={mutedColor}>{t('details.algorithm') || '识别算法'}</Text>
                        <Text fontWeight="medium">{selectedDetailRecord.algorithm_name} {selectedDetailRecord.algorithm_version ? `(v${selectedDetailRecord.algorithm_version})` : ''}</Text>
                      </GridItem>
                    )}
                  </Grid>
                </VStack>

                {/* 个性化属性面板 */}
                {(selectedDetailRecord.record_type === 'recognition' ||
                  selectedDetailRecord.record_type === 'alarm' ||
                  selectedDetailRecord.record_type === 'capture') && (
                    <>
                      <Divider />
                      <VStack align="stretch" spacing={3}>
                        <Text fontWeight="bold" fontSize="sm" color={textColor}>
                          {t('details.detailInfo') || '详情属性'}
                        </Text>

                        <Grid templateColumns="repeat(2, 1fr)" gap={4} fontSize="sm">
                          {/* 识别记录字段 */}
                          {selectedDetailRecord.record_type === 'recognition' && (
                            <>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.person')}</Text>
                                <Text fontWeight="bold" color="blue.500">{selectedDetailRecord.person_name || t('actions.unknownPerson') || '未知人员'}</Text>
                              </GridItem>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.similarity')}</Text>
                                <Text fontWeight="bold">{formatPercent(selectedDetailRecord.similarity)}</Text>
                              </GridItem>
                            </>
                          )}

                          {/* 告警记录字段 */}
                          {selectedDetailRecord.record_type === 'alarm' && (
                            <>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.alarmType')}</Text>
                                <Text fontWeight="medium">{selectedDetailRecord.alarm_type ? t(`alarmType.${selectedDetailRecord.alarm_type}`, { defaultValue: selectedDetailRecord.alarm_type }) : '-'}</Text>
                              </GridItem>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.alarmLevel')}</Text>
                                <Badge colorScheme={statusColor[selectedDetailRecord.alarm_level || ''] || 'gray'} variant="solid">
                                  {selectedDetailRecord.alarm_level ? t(`alarmLevel.${selectedDetailRecord.alarm_level}`, { defaultValue: selectedDetailRecord.alarm_level }) : '-'}
                                </Badge>
                              </GridItem>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.status')}</Text>
                                <Badge colorScheme={alarmStatusColor[selectedDetailRecord.alarm_status || 'unhandled'] || 'gray'}>
                                  {t(`status.${selectedDetailRecord.alarm_status || 'unhandled'}`, { defaultValue: selectedDetailRecord.alarm_status || 'unhandled' })}
                                </Badge>
                              </GridItem>
                            </>
                          )}

                          {/* 抓拍记录字段 */}
                          {selectedDetailRecord.record_type === 'capture' && (
                            <>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.category')}</Text>
                                <Text fontWeight="medium">{selectedDetailRecord.category_name || selectedDetailRecord.category_code?.toString() || '-'}</Text>
                              </GridItem>
                              <GridItem>
                                <Text color={mutedColor}>{t('table.confidence')}</Text>
                                <Text fontWeight="bold">{formatPercent(selectedDetailRecord.confidence)}</Text>
                              </GridItem>
                            </>
                          )}
                        </Grid>
                      </VStack>
                    </>
                  )}

                {/* 原始 JSON 数据面板 */}
                {selectedDetailRecord.raw_result && (
                  <>
                    <Divider />
                    <VStack align="stretch" spacing={3}>
                      <Text fontWeight="bold" fontSize="sm" color={textColor}>
                        {t('fields.rawResult')}
                      </Text>
                      <Box
                        maxH="200px"
                        overflowY="auto"
                        p={3}
                        bg={bgRawCode}
                        borderRadius="md"
                        border="1px solid"
                        borderColor={borderColor}
                      >
                        <Code fontSize="xs" whiteSpace="pre-wrap" bg="transparent" p={0} w="100%">
                          {JSON.stringify(selectedDetailRecord.raw_result, null, 2)}
                        </Code>
                      </Box>
                    </VStack>
                  </>
                )}
              </VStack>
            )}
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </Box>
  );
}
