import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, Select, Button, HStack, useToast, useColorModeValue, Flex,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import { listGB28181Devices, startGB28181Live, stopGB28181Live, type GB28181Device } from '../../../services/gb28181';

export default function LiveView() {
  const { t } = useTranslation('modules/gb28181');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const toast = useToast();
  const [devices, setDevices] = useState<GB28181Device[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [playUrl, setPlayUrl] = useState('');
  const [streamId, setStreamId] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    listGB28181Devices({ page: 1, page_size: 100, status: 'online' })
      .then(res => setDevices(res.list || []))
      .catch(() => {});
  }, []);

  const handleStart = async () => {
    if (!selectedDeviceId) return;
    setLoading(true);
    try {
      const res = await startGB28181Live(selectedDeviceId);
      setPlayUrl(res.url);
      setStreamId(res.stream_id);
      toast({ title: '预览已启动', status: 'success', duration: 3000 });
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    if (!selectedDeviceId || !streamId) return;
    try {
      await stopGB28181Live(selectedDeviceId, streamId);
      setPlayUrl('');
      setStreamId('');
      toast({ title: '预览已停止', status: 'info', duration: 3000 });
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  useEffect(() => {
    return () => {
      if (selectedDeviceId && streamId) {
        stopGB28181Live(selectedDeviceId, streamId).catch(() => {});
      }
    };
  }, [selectedDeviceId, streamId]);

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('live.title')}</Text>
      </Flex>

      <Card p="20px" mb={4}>
        <HStack spacing={4}>
          <Select placeholder={t('live.selectDevice')} value={selectedDeviceId} onChange={(e) => setSelectedDeviceId(e.target.value)} maxW="400px" bg={bgCard}>
            {devices.map(d => (
              <option key={d.id} value={d.id}>{d.device_code} ({d.manufacturer})</option>
            ))}
          </Select>
          <Button colorScheme="green" onClick={handleStart} isLoading={loading} isDisabled={!selectedDeviceId || !!playUrl}>
            {t('live.startPlay')}
          </Button>
          <Button colorScheme="red" onClick={handleStop} isDisabled={!playUrl}>
            {t('live.stopPlay')}
          </Button>
        </HStack>
      </Card>

      {playUrl && (
        <Card p="20px">
          <Text mb={2} fontSize="sm" color="gray.500">{t('live.playUrl')}: <code>{playUrl}</code></Text>
          <Box as="video" src={playUrl} controls autoPlay w="100%" maxH="500px" bg="black" borderRadius="md" />
        </Card>
      )}
    </Box>
  );
}
