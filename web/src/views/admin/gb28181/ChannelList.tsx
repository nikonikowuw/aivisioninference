import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, Badge, Table, Thead, Tbody, Tr, Th, Td,
  Button, HStack, Select, useToast, IconButton, Flex,
  Center, Spinner, useColorModeValue,
} from '@chakra-ui/react';
import { MdRefresh, MdPlayArrow, MdVideoLibrary } from 'react-icons/md';
import Card from 'components/card/Card';
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
  const textColor = useColorModeValue('navy.700', 'white');
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
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('channels.title')}</Text>
        <HStack spacing={2}>
          <Button leftIcon={<MdRefresh />} variant="outline" onClick={fetchChannels} isLoading={loading}>{t('common.refresh')}</Button>
        </HStack>
      </Flex>

      <HStack mb={4} spacing={4}>
        <Select placeholder={t('live.selectDevice')} value={selectedDeviceId} onChange={(e) => setSelectedDeviceId(e.target.value)} maxW="400px" bg={useColorModeValue('white', 'navy.800')}>
          {devices.map(d => (
            <option key={d.id} value={d.id}>{d.device_code} ({d.manufacturer})</option>
          ))}
        </Select>
      </HStack>

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th>{t('channels.fields.channelId')}</Th>
                <Th>{t('channels.fields.channelName')}</Th>
                <Th>{t('channels.fields.status')}</Th>
                <Th>{t('channels.fields.manufacturer')}</Th>
                <Th>{t('channels.fields.model')}</Th>
                <Th textAlign="right">{t('common.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {loading ? (
                <Tr><Td colSpan={6}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : channels.length === 0 ? (
                <Tr><Td colSpan={6}><Center py="20px">{t('channels.noData')}</Center></Td></Tr>
              ) : (
                channels.map((ch) => (
                  <Tr key={ch.id}>
                    <Td><Text fontSize="sm" fontFamily="mono" color={textColor}>{ch.gb28181_device_id}</Text></Td>
                    <Td><Text fontSize="sm" fontWeight="700" color={textColor}>{ch.device_name}</Text></Td>
                    <Td><Badge colorScheme={statusColor(ch.status)} variant="solid">{ch.status}</Badge></Td>
                    <Td><Text fontSize="sm">{ch.manufacturer}</Text></Td>
                    <Td><Text fontSize="sm">{ch.model}</Text></Td>
                    <Td textAlign="right">
                      <HStack justify="flex-end" spacing={1}>
                        <IconButton aria-label="Play" icon={<MdPlayArrow />} size="sm" variant="ghost" colorScheme="green" title={t('channels.actions.play')} />
                        <IconButton aria-label="Playback" icon={<MdVideoLibrary />} size="sm" variant="ghost" colorScheme="blue" title={t('channels.actions.playback')} />
                      </HStack>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>
      </Card>
    </Box>
  );
}
