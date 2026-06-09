import { ChevronDownIcon, ChevronRightIcon } from '@chakra-ui/icons';
import {
  Badge,
  Box, Button, ButtonGroup,
  Center,
  Collapse,
  Flex, Grid, GridItem, HStack,
  Icon,
  IconButton,
  Input, InputGroup, InputLeftElement,
  Spinner, Text,
  useColorModeValue, useToast
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import VideoPlayer from 'components/VideoPlayer';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  MdCheckCircle, MdCircle,
  MdClose, MdError,
  MdFolder, MdFolderOpen,
  MdPlayCircle,
  MdSearch, MdVideocam,
} from 'react-icons/md';
import { deviceGroupsApi, devicesApi, request } from 'services/api';
import { startGB28181Live } from 'services/gb28181';

// ── Types ──
interface Device { id: string; device_name: string; access_type: string; status: string; groups?: { id: string }[] }
interface DeviceGroup { id: string; group_name: string; parent_id?: string | null }
interface PlayResponse { url: string; protocol: string; expires?: number; stream_id?: string }
interface Tile { deviceId: string; deviceName: string; url?: string; protocol?: string; streamId?: string; loading: boolean; error?: string }
interface TreeNode { id: string; name: string; type: 'group' | 'device'; children: TreeNode[]; device?: Device }

const LAYOUTS: Record<number, { cols: number; rows: number }> = { 1: { cols: 1, rows: 1 }, 4: { cols: 2, rows: 2 }, 9: { cols: 3, rows: 3 } };
const STATUS_MAP: Record<string, { color: string; icon: typeof MdCheckCircle }> = {
  online: { color: 'green', icon: MdCheckCircle }, offline: { color: 'red', icon: MdCircle },
  error: { color: 'orange', icon: MdCircle }, unknown: { color: 'gray', icon: MdCircle },
};

function buildTree(groups: DeviceGroup[], devices: Device[]): TreeNode[] {
  const map = new Map<string, TreeNode>();
  groups.forEach(g => map.set(g.id, { id: g.id, name: g.group_name, type: 'group', children: [] }));
  const roots: TreeNode[] = [];
  groups.forEach(g => {
    const node = map.get(g.id)!;
    if (g.parent_id && map.has(g.parent_id)) {
      map.get(g.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  });
  devices.forEach(d => {
    const deviceNode: TreeNode = { id: d.id, name: d.device_name, type: 'device', children: [], device: d };
    let attached = false;
    for (const g of d.groups || []) {
      const groupNode = map.get(g.id);
      if (groupNode) { groupNode.children.push(deviceNode); attached = true; }
    }
    if (!attached) roots.push(deviceNode);
  });
  return roots;
}

const TreeNodeView: React.FC<{ node: TreeNode; depth: number; search: string; onPlay: (d: Device) => void; t: any }> =
  ({ node, depth, search, onPlay, t }) => {
    const [expanded, setExpanded] = useState(depth < 1);
    const isDevice = node.type === 'device';
    const device = node.device;
    const statusInfo = device ? STATUS_MAP[device.status] || STATUS_MAP.unknown : null;
    const textColor = useColorModeValue('navy.700', 'white');
    const hoverBg = useColorModeValue('gray.50', 'whiteAlpha.100');
    if (search && isDevice && !node.name.toLowerCase().includes(search.toLowerCase())) return null;
    if (search && !isDevice && !node.children.some(c => c.name.toLowerCase().includes(search.toLowerCase()))) return null;
    return (
      <Box pl={`${depth * 12}px`}>
        {isDevice ? (
          <Flex p="2" pr="3" borderRadius="md" cursor="grab" _hover={{ bg: hoverBg }} align="center" gap="2"
            draggable onDragStart={(e) => { e.dataTransfer.setData('application/json', JSON.stringify({ id: device!.id, name: device!.device_name, access_type: device!.access_type })); e.dataTransfer.effectAllowed = 'copy'; }}
            onClick={() => device && onPlay(device)}>
            <IconButton aria-label={t('play')} icon={<MdPlayCircle />} size="xs" colorScheme="green" variant="ghost" flexShrink={0}
              onClick={(e) => { e.stopPropagation(); device && onPlay(device); }} />
            <Icon as={MdVideocam} color="blue.500" boxSize="14px" flexShrink={0} />
            <Text fontSize="sm" color={textColor} flex={1} noOfLines={1}>{node.name}</Text>
            {statusInfo && <Icon as={statusInfo.icon} color={`${statusInfo.color}.500`} boxSize="10px" flexShrink={0} />}
          </Flex>
        ) : (
          <Flex p="2" pr="3" borderRadius="md" cursor="pointer" _hover={{ bg: hoverBg }} align="center" gap="2" onClick={() => setExpanded(!expanded)}>
            <Icon as={expanded ? MdFolderOpen : MdFolder} color={expanded ? 'blue.500' : 'gray.500'} boxSize="14px" flexShrink={0} />
            <Text fontSize="sm" fontWeight="medium" color={textColor} flex={1}>{node.name}</Text>
            <Badge size="sm" colorScheme="gray" variant="subtle">{node.children.length}</Badge>
            <Icon as={expanded ? ChevronDownIcon : ChevronRightIcon} color="gray.400" boxSize="14px" flexShrink={0} />
          </Flex>
        )}
        {node.children.length > 0 && (
          <Collapse in={expanded}><Box pt="1">
            {node.children.map(c => <TreeNodeView key={c.id} node={c} depth={depth + 1} search={search} onPlay={onPlay} t={t} />)}
          </Box></Collapse>
        )}
      </Box>
    );
  };

const GridCell: React.FC<{ index: number; tile: Tile | null; onDrop: (i: number, device: Device) => void; onRemove: (i: number) => void; t: any }> =
  ({ index, tile, onDrop, onRemove, t }) => {
    const [dragOver, setDragOver] = useState(false);
    const textColor = useColorModeValue('navy.700', 'white');
    return (
      <GridItem w="100%" h="100%" position="relative" borderRadius="md" overflow="hidden"
        border={dragOver ? '2px solid' : '1px solid'} borderColor={dragOver ? 'blue.400' : 'gray.200'}
        bg={dragOver ? 'blue.50' : 'black'} transition="all 0.15s"
        onDragOver={(e) => { e.preventDefault(); e.dataTransfer.dropEffect = 'copy'; setDragOver(true); }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => { e.preventDefault(); setDragOver(false); try { const d = JSON.parse(e.dataTransfer.getData('application/json')); onDrop(index, { id: d.id, device_name: d.name, access_type: d.access_type || 'rtsp', status: 'unknown' }); } catch { } }}>
        {tile ? (<>
          <Flex position="absolute" top={0} left={0} right={0} zIndex={2} bg="linear-gradient(180deg, rgba(0,0,0,0.7) 0%, transparent 100%)" p={1.5} px={2} justify="space-between" align="center">
            <HStack spacing={1}>
              <MdVideocam size="12px" color="#68D391" />
              <Text fontSize="xs" color="white" fontWeight="medium">{tile.deviceName}</Text>
              {tile.protocol && <Badge size="xs" colorScheme="blue" variant="subtle">{tile.protocol}</Badge>}
            </HStack>
            <IconButton aria-label={t('close')} icon={<MdClose />} size="2xs" variant="ghost" color="white" onClick={() => onRemove(index)} />
          </Flex>
          {tile.loading ? <Center h="100%"><Spinner color="white" size="sm" /></Center>
            : tile.url ? <VideoPlayer url={tile.url} protocol={tile.protocol === 'webrtc' ? 'webrtc' : 'hls'} />
              : <Center h="100%" flexDirection="column" gap={2}><MdError size="24px" color="#FC8181" /><Text color="red.300" fontSize="xs">{tile.error || t('playFailed')}</Text></Center>}
        </>) : (
          <Center h="100%" flexDirection="column" gap={2}>
            <MdVideocam size="32px" opacity={dragOver ? 0.5 : 0.15} />
            {!dragOver && <Text color="gray.500" fontSize="sm">{t('dragToPlay')}</Text>}
            {dragOver && <Text color="blue.400" fontSize="sm" fontWeight="medium">{t('releaseToPlay')}</Text>}
          </Center>
        )}
      </GridItem>
    );
  };

export default function LiveView() {
  const { t } = useTranslation('modules/media');
  const textColor = useColorModeValue('navy.700', 'white');
  const toast = useToast();
  const [devices, setDevices] = useState<Device[]>([]);
  const [groups, setGroups] = useState<DeviceGroup[]>([]);
  const [tiles, setTiles] = useState<(Tile | null)[]>([null]);
  const [layout, setLayout] = useState(1);
  const [search, setSearch] = useState('');

  useEffect(() => {
    devicesApi.list({ page: 1, page_size: 500 }).then(r => setDevices(r.list)).catch(() => toast({ title: t('loadDeviceFailed'), status: 'error', duration: 3000 }));
    deviceGroupsApi.list({ page: 1, page_size: 100 }).then(r => setGroups(r.list)).catch(() => { });
  }, []);

  const treeData = useMemo(() => buildTree(groups, devices), [groups, devices]);

  const startPlayRequest = useCallback((device: Device): Promise<PlayResponse> => {
    if (device.access_type === 'gb28181') {
      return startGB28181Live(device.id);
    }
    return request<PlayResponse>(`/media/play?device_id=${device.id}&protocol=auto`);
  }, []);

  const playDevice = useCallback((device: Device) => {
    setTiles(prev => {
      const emptyIdx = prev.findIndex(t => t === null);
      if (emptyIdx === -1) return prev;
      const updated = [...prev];
      updated[emptyIdx] = { deviceId: device.id, deviceName: device.device_name, loading: true };
      startPlayRequest(device)
        .then(data => {
          setTiles(cur => {
            const u = [...cur];
            if (u[emptyIdx]) Object.assign(u[emptyIdx]!, { url: data.url, protocol: data.protocol, streamId: data.stream_id, loading: false });
            return u;
          });
        })
        .catch(() => {
          setTiles(cur => {
            const u = [...cur];
            if (u[emptyIdx]) Object.assign(u[emptyIdx]!, { loading: false, error: t('playFailed') });
            return u;
          });
        });
      return updated;
    });
  }, [startPlayRequest, t]);

  const handleDrop = useCallback((index: number, device: Device) => {
    setTiles(prev => {
      const u = [...prev];
      u[index] = { deviceId: device.id, deviceName: device.device_name, loading: true };
      startPlayRequest(device)
        .then(data => {
          setTiles(cur => {
            const u = [...cur];
            if (u[index]) Object.assign(u[index]!, { url: data.url, protocol: data.protocol, streamId: data.stream_id, loading: false });
            return u;
          });
        })
        .catch(() => {
          setTiles(cur => {
            const u = [...cur];
            if (u[index]) Object.assign(u[index]!, { loading: false, error: t('playFailed') });
            return u;
          });
        });
      return u;
    });
  }, [startPlayRequest, t]);

  const handleRemove = useCallback((index: number) => setTiles(prev => { const u = [...prev]; u[index] = null; return [...u]; }), []);
  const handleLayoutChange = useCallback((n: number) => { setLayout(n); setTiles(prev => { const u = [...prev]; while (u.length < n) u.push(null); return u.slice(0, n); }); }, []);
  const lc = LAYOUTS[layout] || LAYOUTS[4];

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('liveTitle')}</Text>
        <HStack spacing={3}>
          <ButtonGroup size="xs" isAttached>
            {[1, 4, 9].map(n => <Button key={n} colorScheme={layout === n ? 'blue' : 'gray'} onClick={() => handleLayoutChange(n)}>{t('layout', { count: n })}</Button>)}
          </ButtonGroup>
        </HStack>
      </Flex>
      <Flex gap="20px" h="calc(100vh - 200px)" minH="500px">
        <Card px="0" py="3" w="300px" minW="300px" overflow="hidden" display="flex" flexDirection="column">
          <Box px="3" pb="2">
            <Text fontSize="sm" fontWeight="bold" color={textColor} mb="2">{t('deviceList')}</Text>
            <InputGroup size="sm">
              <InputLeftElement><MdSearch color="gray.400" /></InputLeftElement>
              <Input placeholder={t('searchDevice')} value={search} onChange={e => setSearch(e.target.value)} />
            </InputGroup>
          </Box>
          <Box flex={1} overflowY="auto" px="2" pt="1">
            {treeData.length === 0 ? <Center py="8"><Spinner color="gray.400" size="sm" /></Center>
              : treeData.map(n => <TreeNodeView key={n.id} node={n} depth={0} search={search} onPlay={playDevice} t={t} />)}
          </Box>
        </Card>
        <Card px="4" py="3" flex={1} display="flex" flexDirection="column" overflow="hidden">
          <Box flex={1} minH={0}>
            <Grid templateColumns={`repeat(${lc.cols}, 1fr)`} templateRows={`repeat(${lc.rows}, 1fr)`} gap={3} h="100%">
              {Array.from({ length: layout }, (_, i) => (
                <GridCell key={i} index={i} tile={tiles[i] || null} onDrop={handleDrop} onRemove={handleRemove} t={t} />
              ))}
            </Grid>
          </Box>
        </Card>
      </Flex>
    </Box>
  );
}
