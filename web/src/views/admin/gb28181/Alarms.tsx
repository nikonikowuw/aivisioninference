import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, Badge, Table, Thead, Tbody, Tr, Th, Td, TableContainer,
  Button, HStack, Input, Select, useToast, IconButton,
  Drawer, DrawerBody, DrawerHeader, DrawerOverlay, DrawerContent, DrawerCloseButton,
  VStack, Flex, Spacer, Code,
} from '@chakra-ui/react';
import { MdRefresh, MdVisibility, MdDownload } from 'react-icons/md';
import { listSmartRecords, getSmartRecordsExportUrl, type SmartRecord } from '../../../services/smartRecords';

export default function Alarms() {
  const { t } = useTranslation('modules/smart-records');
  const toast = useToast();
  const [records, setRecords] = useState<SmartRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [alarmType, setAlarmType] = useState('');
  const [alarmLevel, setAlarmLevel] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  const [loading, setLoading] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<SmartRecord | null>(null);
  const [isDetailOpen, setIsDetailOpen] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listSmartRecords({
        page, page_size: pageSize,
        type: 'alarm',
        alarm_type: alarmType || undefined,
        alarm_level: alarmLevel || undefined,
        start_time: startTime || undefined,
        end_time: endTime || undefined,
      });
      setRecords(res.list || []);
      setTotal(res.total || 0);
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, alarmType, alarmLevel, startTime, endTime, toast]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleExport = () => {
    const url = getSmartRecordsExportUrl({
      type: 'alarm',
      alarm_type: alarmType || undefined,
      alarm_level: alarmLevel || undefined,
      start_time: startTime || undefined,
      end_time: endTime || undefined,
    });
    window.open(url, '_blank');
  };

  const alarmLevelColor = (level: string) => {
    switch (level) {
      case '1': return 'red';
      case '2': return 'orange';
      case '3': return 'yellow';
      default: return 'gray';
    }
  };

  return (
    <Box p={6}>
      <Text fontSize="2xl" fontWeight="bold" mb={4}>{t('title')}</Text>

      <HStack mb={4} spacing={4} flexWrap="wrap">
        <Input placeholder={t('filters.alarmType')} value={alarmType} onChange={(e) => setAlarmType(e.target.value)} maxW="200px" />
        <Select placeholder={t('filters.alarmLevel')} value={alarmLevel} onChange={(e) => setAlarmLevel(e.target.value)} maxW="150px">
          <option value="1">1 - 严重</option>
          <option value="2">2 - 重要</option>
          <option value="3">3 - 一般</option>
        </Select>
        <Input type="datetime-local" value={startTime} onChange={(e) => setStartTime(e.target.value)} maxW="200px" />
        <Input type="datetime-local" value={endTime} onChange={(e) => setEndTime(e.target.value)} maxW="200px" />
        <Button onClick={fetchData} isLoading={loading} leftIcon={<MdRefresh />}>刷新</Button>
        <Button onClick={handleExport} leftIcon={<MdDownload />} colorScheme="green">{t('actions.export')}</Button>
      </HStack>

      <TableContainer>
        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>{t('fields.alarmType')}</Th>
              <Th>{t('fields.alarmLevel')}</Th>
              <Th>{t('fields.deviceName')}</Th>
              <Th>{t('fields.createdAt')}</Th>
              <Th>操作</Th>
            </Tr>
          </Thead>
          <Tbody>
            {records.map((r) => (
              <Tr key={r.record_id}>
                <Td>{r.alarm_type || '-'}</Td>
                <Td><Badge colorScheme={alarmLevelColor(r.alarm_level || '')}>{r.alarm_level || '-'}</Badge></Td>
                <Td>{r.device_name || r.device_id || '-'}</Td>
                <Td>{new Date(r.created_at).toLocaleString()}</Td>
                <Td>
                  <IconButton aria-label="Detail" icon={<MdVisibility />} size="sm" onClick={() => { setSelectedRecord(r); setIsDetailOpen(true); }} />
                </Td>
              </Tr>
            ))}
            {records.length === 0 && (
              <Tr><Td colSpan={5} textAlign="center">暂无告警记录</Td></Tr>
            )}
          </Tbody>
        </Table>
      </TableContainer>

      <Flex mt={4}>
        <Spacer />
        <HStack>
          <Button size="sm" onClick={() => setPage(p => Math.max(1, p - 1))} isDisabled={page <= 1}>上一页</Button>
          <Text>第 {page} 页 / 共 {Math.ceil(total / pageSize)} 页</Text>
          <Button size="sm" onClick={() => setPage(p => p + 1)} isDisabled={page * pageSize >= total}>下一页</Button>
        </HStack>
      </Flex>

      <Drawer isOpen={isDetailOpen} placement="right" onClose={() => setIsDetailOpen(false)} size="md">
        <DrawerOverlay />
        <DrawerContent>
          <DrawerCloseButton />
          <DrawerHeader>{t('fields.rawResult')}</DrawerHeader>
          <DrawerBody>
            {selectedRecord && (
              <VStack align="stretch" spacing={3}>
                <Text><strong>{t('fields.recordId')}:</strong> {selectedRecord.record_id}</Text>
                <Text><strong>{t('fields.alarmType')}:</strong> {selectedRecord.alarm_type}</Text>
                <Text><strong>{t('fields.alarmLevel')}:</strong> {selectedRecord.alarm_level}</Text>
                <Text><strong>{t('fields.deviceName')}:</strong> {selectedRecord.device_name}</Text>
                <Text><strong>{t('fields.createdAt')}:</strong> {new Date(selectedRecord.created_at).toLocaleString()}</Text>
                <Box>
                  <Text fontWeight="bold" mb={2}>{t('fields.rawResult')}:</Text>
                  <Code p={3} borderRadius="md" whiteSpace="pre-wrap" fontSize="xs">
                    {JSON.stringify(selectedRecord.raw_result, null, 2)}
                  </Code>
                </Box>
              </VStack>
            )}
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </Box>
  );
}
