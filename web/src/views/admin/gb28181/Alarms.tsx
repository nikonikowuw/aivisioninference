import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, Badge, Table, Thead, Tbody, Tr, Th, Td,
  Button, HStack, Input, Select, useToast, IconButton, Flex,
  Drawer, DrawerBody, DrawerHeader, DrawerOverlay, DrawerContent, DrawerCloseButton,
  VStack, Center, Spinner, useColorModeValue, Code,
} from '@chakra-ui/react';
import { MdRefresh, MdVisibility, MdDownload } from 'react-icons/md';
import Card from 'components/card/Card';
import Pagination from 'components/pagination/Pagination';
import { listSmartRecords, getSmartRecordsExportUrl, type SmartRecord } from '../../../services/smartRecords';

export default function Alarms() {
  const { t } = useTranslation('modules/smart-records');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const toast = useToast();
  const [records, setRecords] = useState<SmartRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
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
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <HStack spacing={2}>
          <Button leftIcon={<MdRefresh />} variant="outline" onClick={fetchData} isLoading={loading}>刷新</Button>
          <Button leftIcon={<MdDownload />} colorScheme="green" onClick={handleExport}>{t('actions.export')}</Button>
        </HStack>
      </Flex>

      <Card p="20px" mb={4}>
        <HStack spacing={4} flexWrap="wrap">
          <Input placeholder={t('filters.alarmType')} value={alarmType} onChange={(e) => setAlarmType(e.target.value)} maxW="200px" bg={bgCard} />
          <Select placeholder={t('filters.alarmLevel')} value={alarmLevel} onChange={(e) => setAlarmLevel(e.target.value)} maxW="150px" bg={bgCard}>
            <option value="1">1 - 严重</option>
            <option value="2">2 - 重要</option>
            <option value="3">3 - 一般</option>
          </Select>
          <Input type="datetime-local" value={startTime} onChange={(e) => setStartTime(e.target.value)} maxW="200px" bg={bgCard} />
          <Input type="datetime-local" value={endTime} onChange={(e) => setEndTime(e.target.value)} maxW="200px" bg={bgCard} />
        </HStack>
      </Card>

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th>{t('fields.alarmType')}</Th>
                <Th>{t('fields.alarmLevel')}</Th>
                <Th>{t('fields.deviceName')}</Th>
                <Th>{t('fields.createdAt')}</Th>
                <Th textAlign="right">操作</Th>
              </Tr>
            </Thead>
            <Tbody>
              {loading ? (
                <Tr><Td colSpan={5}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : records.length === 0 ? (
                <Tr><Td colSpan={5}><Center py="20px">暂无告警记录</Center></Td></Tr>
              ) : (
                records.map((r) => (
                  <Tr key={r.record_id}>
                    <Td><Text fontSize="sm">{r.alarm_type || '-'}</Text></Td>
                    <Td><Badge colorScheme={alarmLevelColor(r.alarm_level || '')} variant="solid">{r.alarm_level || '-'}</Badge></Td>
                    <Td><Text fontSize="sm" color={textColor} fontWeight="700">{r.device_name || r.device_id || '-'}</Text></Td>
                    <Td><Text fontSize="sm">{new Date(r.created_at).toLocaleString()}</Text></Td>
                    <Td textAlign="right">
                      <IconButton aria-label="Detail" icon={<MdVisibility />} size="sm" variant="ghost" onClick={() => { setSelectedRecord(r); setIsDetailOpen(true); }} />
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
