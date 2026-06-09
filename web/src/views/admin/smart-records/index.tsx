import {
  Badge,
  Box,
  Button,
  Checkbox,
  Center,
  Flex,
  HStack,
  Image,
  Spinner,
  Table,
  Tabs,
  Tab,
  TabList,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { CheckIcon, DeleteIcon, DownloadIcon } from '@chakra-ui/icons';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from 'contexts/AuthContext';
import { smartRecordsApi, devicesApi, deviceGroupsApi, tasksApi, type SmartRecord, type SmartRecordType, type CategoryCodeOption } from 'services/api';
import type { Device, DeviceGroup, Task } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { hasPermission } from 'utils/permission';
import Pagination from 'components/pagination/Pagination';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import { SearchBar, type InputConfig, type SelectConfig } from 'components/search-bar/SearchBar';

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
  const [deviceOptions, setDeviceOptions] = useState<{ value: string; label: string }[]>([]);
  const [deviceGroupOptions, setDeviceGroupOptions] = useState<{ value: string; label: string }[]>([]);
  const [taskOptions, setTaskOptions] = useState<{ value: string; label: string }[]>([]);
  const [categoryCodeOptions, setCategoryCodeOptions] = useState<{ value: string; label: string }[]>([]);

  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('gray.600', 'gray.300');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

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
    const baseColumns = 6;
    if (activeType === 'recognition') return baseColumns + 3;
    if (activeType === 'capture') return baseColumns + 2;
    if (activeType === 'alarm') {
      const actionColumn = hasPermission(permissionCodes, 'records:alarm:update_status') ? 1 : 0;
      return baseColumns + 3 + actionColumn;
    }
    return baseColumns;
  }, [activeType, permissionCodes]);

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
              {activeType === 'alarm' && hasPermission(permissionCodes, 'records:alarm:update_status') && <Th>{t('table.actions')}</Th>}
              {activeType === 'capture' && <Th>{t('table.category')}</Th>}
              {activeType === 'capture' && <Th>{t('table.confidence')}</Th>}
              <Th>{t('table.taskName')}</Th>
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
                {activeType === 'alarm' && hasPermission(permissionCodes, 'records:alarm:update_status') && (
                  <Td>
                    {(record.alarm_status || 'unhandled') === 'unhandled' ? (
                      <Button
                        size="sm"
                        leftIcon={<CheckIcon />}
                        onClick={() => handleAlarmStatusUpdate(record)}
                        isLoading={updatingAlarmStatusId === record.record_id}
                      >
                        {t('actions.markHandled')}
                      </Button>
                    ) : '-'}
                  </Td>
                )}
                {activeType === 'capture' && <Td>{record.category_name || record.category_code?.toString() || '-'}</Td>}
                {activeType === 'capture' && <Td>{formatPercent(record.confidence)}</Td>}
                <Td>{record.task_name || '-'}</Td>
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
    </Box>
  );
}
