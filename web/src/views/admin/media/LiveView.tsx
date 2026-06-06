import {
  Box,
  Button,
  ButtonGroup,
  Center,
  Grid,
  GridItem,
  HStack,
  IconButton,
  Spinner,
  useToast,
} from '@chakra-ui/react';
import VideoPlayer from 'components/VideoPlayer';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { MdClose, MdRefresh } from 'react-icons/md';
import { request } from 'services/api';

interface StreamTile {
  deviceId: string;
  url?: string;
  loading: boolean;
  error?: string;
}

interface PlayResponse {
  url: string;
  protocol: string;
  expires: number;
}

const LAYOUTS: Record<number, { cols: number }> = {
  1: { cols: 1 },
  4: { cols: 2 },
  6: { cols: 3 },
  8: { cols: 4 },
  9: { cols: 3 },
};

const LiveView: React.FC = () => {
  const { t } = useTranslation('modules/media');
  const [tiles, setTiles] = useState<StreamTile[]>([]);
  const [layout, setLayout] = useState<number>(4);
  const toast = useToast();

  const addTile = async (deviceId: string) => {
    const idx = tiles.length;
    setTiles(prev => [...prev, { deviceId, loading: true }]);

    try {
      const data = await request<PlayResponse>(
        `/media/play?device_id=${encodeURIComponent(deviceId)}&protocol=auto`
      );
      setTiles(prev => {
        const updated = [...prev];
        if (updated[idx]) {
          updated[idx] = { ...updated[idx], url: data.url, loading: false };
        }
        return updated;
      });
    } catch (err: any) {
      toast({ title: `获取播放地址失败: ${deviceId}`, status: 'error', duration: 3000 });
      setTiles(prev => {
        const updated = [...prev];
        if (updated[idx]) {
          updated[idx] = { ...updated[idx], loading: false, error: err.message };
        }
        return updated;
      });
    }
  };

  const removeTile = (index: number) => {
    setTiles(prev => prev.filter((_, i) => i !== index));
  };

  const layoutConfig = LAYOUTS[layout] || LAYOUTS[4];

  return (
    <Box p={4} h="calc(100vh - 80px)">
      <HStack mb={4} spacing={4}>
        <ButtonGroup size="sm" isAttached variant="outline">
          {[1, 4, 6, 8, 9].map(n => (
            <Button
              key={n}
              colorScheme={layout === n ? 'blue' : 'gray'}
              onClick={() => setLayout(n)}
            >
              {t('layout.channels', { count: n, defaultValue: `${n}路` })}
            </Button>
          ))}
        </ButtonGroup>
        <IconButton
          aria-label={t('actions.refresh', { defaultValue: 'Refresh' })}
          icon={<MdRefresh />}
          size="sm"
          onClick={() => addTile(String(Date.now()))}
          title={t('actions.addDeviceHint', { defaultValue: '添加临时设备（请输入真实设备ID）' })}
        />
      </HStack>

      <Grid
        templateColumns={`repeat(${layoutConfig.cols}, 1fr)`}
        templateRows={`repeat(${layoutConfig.cols}, 1fr)`}
        gap={2}
        h="calc(100% - 50px)"
      >
        {tiles.length === 0 ? (
          <Center gridColumn="1 / -1" gridRow="1 / -1" color="gray.500">
            {t('empty.hint', { defaultValue: '点击刷新按钮添加设备预览' })}
          </Center>
        ) : (
          tiles.map((tile, idx) => (
            <GridItem key={idx} w="100%" h="100%" position="relative">
              <IconButton
                aria-label={t('close', { defaultValue: 'Close' })}
                icon={<MdClose />}
                size="xs"
                position="absolute"
                top={1}
                right={1}
                zIndex={1}
                colorScheme="red"
                onClick={() => removeTile(idx)}
              />
              {tile.loading ? (
                <Center h="100%" bg="gray.800" borderRadius="md">
                  <Spinner color="white" />
                </Center>
              ) : tile.url ? (
                <VideoPlayer url={tile.url} protocol="hls" />
              ) : (
                <Center h="100%" bg="gray.800" borderRadius="md" color="red.300" fontSize="sm">
                  {tile.error || t('empty.loadFailed', { defaultValue: '加载失败' })}
                </Center>
              )}
            </GridItem>
          ))
        )}
      </Grid>
    </Box>
  );
};

export default LiveView;
