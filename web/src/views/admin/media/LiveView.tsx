import {
  Badge,
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
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { MdClose, MdFullscreen, MdFullscreenExit, MdRefresh } from 'react-icons/md';
import { mediaApi } from 'services/api';

interface StreamTile {
  deviceId: string;
  url?: string;
  prevUrl?: string;
  streamType: 'main' | 'sub';
  codec?: string;
  loading: boolean;
  error?: string;
}

const LAYOUTS: Record<number, { cols: number }> = {
  1: { cols: 1 },
  4: { cols: 2 },
  9: { cols: 3 },
  16: { cols: 4 },
};

const LiveView: React.FC = () => {
  const { t } = useTranslation('modules/media');
  const [tiles, setTiles] = useState<StreamTile[]>([]);
  const tilesRef = useRef<StreamTile[]>(tiles);
  const [layout, setLayout] = useState<number>(4);
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null);
  const toast = useToast();

  const stopTilePlay = useCallback((tile?: StreamTile) => {
    if (!tile || !tile.url) return;
    mediaApi.stopPlay(tile.deviceId, tile.streamType).catch(() => {});
  }, []);

  useEffect(() => {
    tilesRef.current = tiles;
  }, [tiles]);

  useEffect(() => () => {
    tilesRef.current.forEach(stopTilePlay);
  }, [stopTilePlay]);

  const loadTileStream = useCallback(async (deviceId: string, targetStreamType: 'main' | 'sub', currentTile?: StreamTile): Promise<StreamTile> => {
    try {
      const res = await mediaApi.getPlayUrl({
        device_id: deviceId,
        stream_type: targetStreamType,
        protocol: 'auto',
      });
      return {
        deviceId,
        url: res.url,
        prevUrl: undefined,
        streamType: targetStreamType,
        codec: res.codec || 'h264',
        loading: false,
      };
    } catch (err: any) {
      return {
        deviceId,
        url: currentTile?.url,
        prevUrl: undefined,
        streamType: targetStreamType,
        codec: currentTile?.codec,
        loading: false,
        error: err.message || '获取播放地址失败',
      };
    }
  }, []);

  const addTile = async (deviceId: string) => {
    const defaultStreamType: 'main' | 'sub' = (layout === 1 || focusedIndex !== null) ? 'main' : 'sub';
    const newTile: StreamTile = {
      deviceId,
      streamType: defaultStreamType,
      loading: true,
    };

    setTiles(prev => [...prev, newTile]);

    const loaded = await loadTileStream(deviceId, defaultStreamType);
    setTiles(prev => prev.map(t => (t.deviceId === deviceId && t.loading ? loaded : t)));
  };

  const removeTile = (index: number) => {
    setTiles(prev => {
      stopTilePlay(prev[index]);
      return prev.filter((_, i) => i !== index);
    });
    if (focusedIndex === index) {
      setFocusedIndex(null);
    } else if (focusedIndex !== null && focusedIndex > index) {
      setFocusedIndex(focusedIndex - 1);
    }
  };

  // Switch stream type for all tiles when layout changes or single tile focused
  const updateTileStreams = useCallback(async (targetLayout: number, targetFocusedIndex: number | null) => {
    setTiles(prevTiles => {
      const isSingleView = targetLayout === 1 || targetFocusedIndex !== null;
      
      prevTiles.forEach(async (tile, idx) => {
        const desiredType: 'main' | 'sub' = isSingleView && (targetFocusedIndex === null || targetFocusedIndex === idx)
          ? 'main'
          : 'sub';

        if (tile.streamType !== desiredType && tile.url) {
          // Release previous stream
          mediaApi.stopPlay(tile.deviceId, tile.streamType).catch(() => {});

          // Pre-roll frame transition: keep prevUrl active while fetching new stream
          setTiles(curr => curr.map((t, i) => i === idx ? {
            ...t,
            loading: true,
            prevUrl: t.url,
            streamType: desiredType,
          } : t));

          const updated = await loadTileStream(tile.deviceId, desiredType, tile);
          setTiles(curr => curr.map((t, i) => i === idx ? updated : t));
        }
      });

      return prevTiles;
    });
  }, [loadTileStream]);

  const handleLayoutChange = (n: number) => {
    setLayout(n);
    setFocusedIndex(null);
    updateTileStreams(n, null);
  };

  const toggleTileFocus = (index: number) => {
    const nextFocused = focusedIndex === index ? null : index;
    setFocusedIndex(nextFocused);
    updateTileStreams(layout, nextFocused);
  };

  const layoutConfig = LAYOUTS[layout] || LAYOUTS[4];

  const visibleTiles = focusedIndex !== null && tiles[focusedIndex]
    ? [{ tile: tiles[focusedIndex], originalIndex: focusedIndex }]
    : tiles.map((t, i) => ({ tile: t, originalIndex: i }));

  return (
    <Box p={4} h="calc(100vh - 80px)">
      <HStack mb={4} spacing={4} justify="space-between">
        <HStack spacing={4}>
          <ButtonGroup size="sm" isAttached variant="outline">
            {[1, 4, 9, 16].map(n => (
              <Button
                key={n}
                colorScheme={layout === n && focusedIndex === null ? 'blue' : 'gray'}
                onClick={() => handleLayoutChange(n)}
              >
                {t('layout.channels', { count: n, defaultValue: `${n}分屏` })}
              </Button>
            ))}
          </ButtonGroup>
          {focusedIndex !== null && (
            <Button
              size="sm"
              colorScheme="yellow"
              leftIcon={<MdFullscreenExit />}
              onClick={() => {
                setFocusedIndex(null);
                updateTileStreams(layout, null);
              }}
            >
              {t('layout.restoreGrid', { defaultValue: '恢复分屏' })}
            </Button>
          )}
        </HStack>
        <IconButton
          aria-label={t('actions.refresh', { defaultValue: 'Refresh' })}
          icon={<MdRefresh />}
          size="sm"
          onClick={() => addTile(`DEV_${String(Date.now()).slice(-4)}`)}
          title={t('actions.addDeviceHint', { defaultValue: '添加测试设备' })}
        />
      </HStack>

      <Grid
        templateColumns={focusedIndex !== null ? '1fr' : `repeat(${layoutConfig.cols}, 1fr)`}
        templateRows={focusedIndex !== null ? '1fr' : `repeat(${layoutConfig.cols}, 1fr)`}
        gap={2}
        h="calc(100% - 50px)"
      >
        {tiles.length === 0 ? (
          <Center gridColumn="1 / -1" gridRow="1 / -1" color="gray.500">
            {t('empty.hint', { defaultValue: '点击右上角按钮添加设备预览' })}
          </Center>
        ) : (
          visibleTiles.map(({ tile, originalIndex }) => (
            <GridItem
              key={originalIndex}
              w="100%"
              h="100%"
              position="relative"
              onDoubleClick={() => toggleTileFocus(originalIndex)}
            >
              <HStack position="absolute" top={1} right={1} zIndex={2} spacing={1}>
                <IconButton
                  aria-label={focusedIndex === originalIndex ? 'Exit Single' : 'Single View'}
                  icon={focusedIndex === originalIndex ? <MdFullscreenExit /> : <MdFullscreen />}
                  size="xs"
                  colorScheme="blue"
                  onClick={() => toggleTileFocus(originalIndex)}
                />
                <IconButton
                  aria-label={t('close', { defaultValue: 'Close' })}
                  icon={<MdClose />}
                  size="xs"
                  colorScheme="red"
                  onClick={() => removeTile(originalIndex)}
                />
              </HStack>

              <HStack position="absolute" top={1} left={1} zIndex={2} spacing={1}>
                <Badge colorScheme={tile.streamType === 'main' ? 'green' : 'cyan'} fontSize="xs">
                  {tile.streamType === 'main' ? '主流 (HD)' : '子流 (SD)'}
                </Badge>
                {tile.codec && (
                  <Badge colorScheme={tile.codec.toLowerCase() === 'h265' ? 'purple' : 'gray'} fontSize="xs">
                    {tile.codec.toUpperCase()}
                  </Badge>
                )}
              </HStack>

              {tile.loading && !tile.prevUrl ? (
                <Center h="100%" bg="gray.800" borderRadius="md">
                  <Spinner color="white" />
                </Center>
              ) : (tile.url || tile.prevUrl) ? (
                <Box position="relative" w="100%" h="100%">
                  <VideoPlayer url={tile.url || tile.prevUrl!} deviceId={tile.deviceId} />
                  {tile.loading && (
                    <Center position="absolute" top={0} left={0} w="100%" h="100%" bg="blackAlpha.600" zIndex={1}>
                      <Spinner color="white" size="md" />
                    </Center>
                  )}
                </Box>
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
