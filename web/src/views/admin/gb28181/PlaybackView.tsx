import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, Select, Button, HStack, VStack, useToast, useColorModeValue, Flex,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import {
  listGB28181Devices, startGB28181Playback, stopGB28181Playback,
  controlGB28181Playback, type GB28181Device,
} from '../../../services/gb28181';

export default function PlaybackView() {
  const { t } = useTranslation('modules/gb28181');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const toast = useToast();
  const [devices, setDevices] = useState<GB28181Device[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  const [playUrl, setPlayUrl] = useState('');
  const [streamId, setStreamId] = useState('');
  const [loading, setLoading] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const [speed, setSpeed] = useState(1);

  useEffect(() => {
    listGB28181Devices({ page: 1, page_size: 100 })
      .then(res => setDevices(res.list || []))
      .catch(() => {});
    const now = new Date();
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    setEndTime(now.toISOString().slice(0, 16));
    setStartTime(yesterday.toISOString().slice(0, 16));
  }, []);

  const handleStart = async () => {
    if (!selectedDeviceId || !startTime || !endTime) return;
    setLoading(true);
    try {
      const res = await startGB28181Playback(selectedDeviceId, startTime + ':00Z', endTime + ':00Z');
      setPlayUrl(res.url);
      setStreamId(res.stream_id);
      toast({ title: '回放已启动', status: 'success', duration: 3000 });
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    if (!selectedDeviceId || !streamId) return;
    try {
      await stopGB28181Playback(selectedDeviceId, streamId);
      setPlayUrl('');
      setStreamId('');
      toast({ title: '回放已停止', status: 'info', duration: 3000 });
    } catch (err: any) {
      toast({ title: err.message, status: 'error', duration: 3000 });
    }
  };

  const handlePauseResume = async () => {
    if (!streamId) return;
    const action = isPaused ? 'resume' : 'pause';
    await controlGB28181Playback(streamId, action);
    setIsPaused(!isPaused);
  };

  const handleSpeedChange = async (newSpeed: number) => {
    if (!streamId) return;
    setSpeed(newSpeed);
    await controlGB28181Playback(streamId, 'scale', newSpeed);
  };

  useEffect(() => {
    return () => {
      if (selectedDeviceId && streamId) {
        stopGB28181Playback(selectedDeviceId, streamId).catch(() => {});
      }
    };
  }, [selectedDeviceId, streamId]);

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('playback.title')}</Text>
      </Flex>

      <Card p="20px" mb={4}>
        <VStack align="stretch" spacing={4}>
          <HStack spacing={4}>
            <Select placeholder={t('live.selectDevice')} value={selectedDeviceId} onChange={(e) => setSelectedDeviceId(e.target.value)} maxW="400px" bg={bgCard}>
              {devices.map(d => (
                <option key={d.id} value={d.id}>{d.device_code} ({d.manufacturer})</option>
              ))}
            </Select>
          </HStack>
          <HStack spacing={4}>
            <Box>
              <Text fontSize="sm" mb={1}>{t('playback.startTime')}</Text>
              <input type="datetime-local" value={startTime} onChange={(e) => setStartTime(e.target.value)} style={{ padding: '8px', border: '1px solid #ccc', borderRadius: '4px' }} />
            </Box>
            <Box>
              <Text fontSize="sm" mb={1}>{t('playback.endTime')}</Text>
              <input type="datetime-local" value={endTime} onChange={(e) => setEndTime(e.target.value)} style={{ padding: '8px', border: '1px solid #ccc', borderRadius: '4px' }} />
            </Box>
            <Button colorScheme="green" onClick={handleStart} isLoading={loading} isDisabled={!selectedDeviceId || !!playUrl} alignSelf="flex-end">
              {t('playback.startPlayback')}
            </Button>
            <Button colorScheme="red" onClick={handleStop} isDisabled={!playUrl} alignSelf="flex-end">
              {t('playback.stopPlayback')}
            </Button>
          </HStack>
        </VStack>
      </Card>

      {playUrl && (
        <Card p="20px">
          <Box as="video" src={playUrl} controls autoPlay w="100%" maxH="400px" bg="black" borderRadius="md" />
          <HStack mt={3} spacing={4}>
            <Button size="sm" variant="outline" onClick={handlePauseResume}>{isPaused ? t('playback.resume') : t('playback.pause')}</Button>
            <Text fontSize="sm">{t('playback.speed')}:</Text>
            {[0.5, 1, 2, 4].map(s => (
              <Button key={s} size="sm" variant={speed === s ? 'solid' : 'outline'} colorScheme={speed === s ? 'brand' : 'gray'} onClick={() => handleSpeedChange(s)}>
                {s}x
              </Button>
            ))}
          </HStack>
        </Card>
      )}
    </Box>
  );
}
