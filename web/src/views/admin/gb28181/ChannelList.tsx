import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, Badge, Table, Thead, Tbody, Tr, Th, Td, TableContainer,
  Button, HStack, Select, useToast, IconButton, Flex, Spacer,
} from '@chakra-ui/react';
import { MdRefresh, MdPlayArrow, MdVideoLibrary } from 'react-icons/md';
import { listGB28181Devices, getGB28181Channels, type GB28181Device } from '../../../services/gb28181';

interface Channel {
  id: string;
  device_name: string;
  gb28181_device_id: string;
  status: string;
  manufacturer: string;
  model: string;
}

export default function ChannelList() {
  const { t } = useTranslation('modules/gb28181');
  const toast = useToast();
  const [devices, setDevices] = useState<GB28181Device[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    listGB28181Devices({ page: 1, page_size: 100 }).then(res => setDevices(res.list || [])).catch(() => {});
  }, []);

  const fetchChannels = useCallback(async () => {
    if (!selectedDeviceId) return;
    setLoading(true);
    try {
      const res = await getGB28181Channels(selectedDeviceId);
      setChannels(res || []);
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    } finally {
      setLoading(false);
    }
  }, [selectedDeviceId, toast]);

  useEffect(() => { fetchChannels(); }, [fetchChannels]);

  const statusColor = (status: string) => status === 'online' ? 'green' : 'gray';

  return (
    <Box p={6}>
      <Text fontSize="2xl" fontWeight="bold" mb={4}>{t('channels.title')}</Text>

      <HStack mb={4} spacing={4}>
        <Select placeholder={t('live.selectDevice')} value={selectedDeviceId} onChange={(e) => setSelectedDeviceId(e.target.value)} maxW="400px">
          {devices.map(d => (
            <option key={d.id} value={d.id}>{d.device_code} ({d.manufacturer})</option>
          ))}
        </Select>
        <Button onClick={fetchChannels} isLoading={loading} leftIcon={<MdRefresh />}>刷新</Button>
      </HStack>

      <TableContainer>
        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>{t('channels.fields.channelId')}</Th>
              <Th>{t('channels.fields.channelName')}</Th>
              <Th>{t('channels.fields.status')}</Th>
              <Th>{t('channels.fields.manufacturer')}</Th>
              <Th>{t('channels.fields.model')}</Th>
              <Th>操作</Th>
            </Tr>
          </Thead>
          <Tbody>
            {channels.map((ch) => (
              <Tr key={ch.id}>
                <Td>{ch.gb28181_device_id}</Td>
                <Td>{ch.device_name}</Td>
                <Td><Badge colorScheme={statusColor(ch.status)}>{ch.status}</Badge></Td>
                <Td>{ch.manufacturer}</Td>
                <Td>{ch.model}</Td>
                <Td>
                  <HStack spacing={1}>
                    <IconButton aria-label="Play" icon={<MdPlayArrow />} size="sm" colorScheme="green" title={t('channels.actions.play')} />
                    <IconButton aria-label="Playback" icon={<MdVideoLibrary />} size="sm" colorScheme="blue" title={t('channels.actions.playback')} />
                  </HStack>
                </Td>
              </Tr>
            ))}
            {channels.length === 0 && (
              <Tr><Td colSpan={6} textAlign="center">暂无通道数据</Td></Tr>
            )}
          </Tbody>
        </Table>
      </TableContainer>
    </Box>
  );
}
