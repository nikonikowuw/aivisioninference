import {
  Box,
  Flex,
  Text,
  Tag,
  Progress,
  CircularProgress,
  Icon,
  HStack,
  useColorModeValue,
} from '@chakra-ui/react';
import {
  FiCpu,
  FiZap,
  FiDatabase,
  FiClock,
  FiArrowUp,
  FiPlay,
  FiChevronRight,
  FiVideo,
} from 'react-icons/fi';
import { MdMemory } from 'react-icons/md';
import { useTranslation } from 'react-i18next';
import { formatUptime } from 'utils/convert';
import type { EdgeNodeCardSnapshot } from 'services/edgeNode';

interface EdgeNodeCardProps {
  node: EdgeNodeCardSnapshot;
  compact?: boolean;
  onClick?: () => void;
  onDeployClick?: (e: React.MouseEvent) => void;
}

function formatBandwidth(bps?: number): string {
  if (bps === undefined || bps === null) return '-';
  const kbps = bps / 1000;
  if (kbps < 1000) return `${kbps.toFixed(1)} Kbps`;
  const mbps = kbps / 1000;
  return `${mbps.toFixed(1)} Mbps`;
}

export default function EdgeNodeCard({ node, compact = false, onClick, onDeployClick }: EdgeNodeCardProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const cardBg = useColorModeValue('rgba(255, 255, 255, 0.67)', 'rgba(26, 32, 44, 0.67)');
  const hoverBg = useColorModeValue('rgba(255, 255, 255, 0.85)', 'rgba(45, 55, 72, 0.85)');
  const borderColor = useColorModeValue('rgba(255, 255, 255, 0.82)', 'rgba(255, 255, 255, 0.12)');
  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('gray.500', 'gray.400');

  const frozen = node.status === 'offline' || node.status === 'disabled';

  const statusColors: Record<string, string> = {
    online: 'green',
    offline: 'gray',
    error: 'red',
    disabled: 'orange',
  };

  // Helper to color metrics based on utilization thresholds
  const getMetricColor = (val?: number, normalColor = 'green') => {
    if (val === undefined || val === null) return 'gray';
    if (val >= 90) return 'red';
    if (val >= 75) return 'yellow';
    return normalColor;
  };

  // 1. Host stats
  const cpuVal = frozen ? undefined : node.cpu_usage;
  const memVal = frozen ? undefined : node.memory_usage;
  
  const diskInfo = node.disk_usage && node.disk_usage.length > 0 ? node.disk_usage[0] : null;
  const diskVal = diskInfo ? diskInfo.percent : undefined;

  // 2. Accelerator (NPU/GPU)
  const hasAcc = node.accelerator_metrics_valid;
  const accVal = frozen || !hasAcc ? undefined : (node.accelerator_utilization !== undefined && node.accelerator_utilization !== null ? Number(node.accelerator_utilization) : undefined);

  // 3. Ring calculations
  // Egress usage
  const egressUsage = node.egress_capacity_bps && node.egress_capacity_bps > 0 && node.egress_bps
    ? (Number(node.egress_bps) / Number(node.egress_capacity_bps)) * 100
    : 0;

  // Decode usage
  const decodeUsage = node.hardware_info?.cpu_cores && node.decode_slots_used
    ? (node.decode_slots_used / (node.hardware_info.cpu_cores * 2)) * 100 // default mock capacity base on cpu_cores
    : 0;

  // Active load usage
  const loadUsage = node.max_load > 0 ? (node.current_load / node.max_load) * 100 : 0;

  return (
    <Box
      className={`node-card ${frozen ? 'offline' : ''}`}
      bg={cardBg}
      border="1px solid"
      borderColor={borderColor}
      borderRadius="14px"
      boxShadow="0 12px 35px rgba(19, 32, 25, 0.08)"
      backdropFilter="blur(20px) saturate(1.12)"
      cursor="pointer"
      transition="transform 180ms ease, box-shadow 180ms ease, background 180ms ease"
      _hover={{
        transform: 'translateY(-3px)',
        boxShadow: '0 18px 42px rgba(19, 32, 25, 0.16)',
        bg: hoverBg,
      }}
      onClick={onClick}
      p="16px"
      display="flex"
      flexDirection="column"
      gap="12px"
      opacity={frozen ? 0.72 : 1}
      tabIndex={0}
      role="button"
      aria-label={`Manage ${node.name}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onClick?.();
        }
      }}
    >
      {/* Header */}
      <Flex justify="space-between" align="flex-start" gap="8px">
        <Box minW="0" flex="1">
          <Text
            color={textColor}
            fontSize="sm"
            fontWeight="bold"
            noOfLines={1}
            title={node.name}
          >
            {node.name}
          </Text>
          <Text
            color={mutedColor}
            fontSize="9.5px"
            noOfLines={1}
            mt="2px"
            title={`${node.hardware_info?.cpu_model || ''} · ${node.hardware_info?.gpu_model || ''}`}
          >
            {node.hardware_info?.cpu_model ? `${node.hardware_info.cpu_model} · ${node.hardware_info.gpu_model || ''}` : node.endpoint}
          </Text>
        </Box>
        <Tag
          colorScheme={statusColors[node.status] || 'gray'}
          variant="solid"
          size="sm"
          borderRadius="full"
          fontSize="9px"
          fontWeight="bold"
          px="8px"
          py="2px"
        >
          {t(`status.${node.status}`)}
        </Tag>
      </Flex>

      {/* Progress Bars */}
      <Flex direction="column" gap="8px" mt="4px">
        {/* CPU */}
        <Box>
          <Flex justify="space-between" align="center" fontSize="9.5px">
            <HStack spacing="4px" color={mutedColor}>
              <Icon as={FiCpu} w="13px" h="13px" />
              <Text>{t('fields.cpuUsage')}</Text>
            </HStack>
            <Text fontWeight="bold" color={textColor}>
              {cpuVal !== undefined ? `${cpuVal.toFixed(1)}%` : 'N/A'}
            </Text>
          </Flex>
          <Progress
            value={cpuVal ?? 0}
            size="xs"
            borderRadius="full"
            mt="4px"
            colorScheme={getMetricColor(cpuVal, 'green')}
            bg="rgba(0,0,0,0.06)"
          />
        </Box>

        {/* Memory */}
        <Box>
          <Flex justify="space-between" align="center" fontSize="9.5px">
            <HStack spacing="4px" color={mutedColor}>
              <Icon as={MdMemory} w="13px" h="13px" />
              <Text>{t('fields.memUsage')}</Text>
            </HStack>
            <Text fontWeight="bold" color={textColor}>
              {memVal !== undefined ? `${memVal.toFixed(1)}%` : 'N/A'}
            </Text>
          </Flex>
          <Progress
            value={memVal ?? 0}
            size="xs"
            borderRadius="full"
            mt="4px"
            colorScheme={getMetricColor(memVal, 'green')}
            bg="rgba(0,0,0,0.06)"
          />
        </Box>

        {/* Accelerator / NPU / GPU */}
        <Box>
          <Flex justify="space-between" align="center" fontSize="9.5px">
            <HStack spacing="4px" color={mutedColor}>
              <Icon as={FiZap} w="13px" h="13px" />
              <Text>{hasAcc ? 'NPU' : 'GPU'}</Text>
            </HStack>
            <Text fontWeight="bold" color={textColor}>
              {accVal !== undefined ? `${accVal.toFixed(1)}%` : 'N/A'}
            </Text>
          </Flex>
          <Progress
            value={accVal ?? 0}
            size="xs"
            borderRadius="full"
            mt="4px"
            colorScheme={getMetricColor(accVal, 'blue')}
            bg="rgba(0,0,0,0.06)"
          />
        </Box>

        {/* Disk */}
        <Box>
          <Flex justify="space-between" align="center" fontSize="9.5px">
            <HStack spacing="4px" color={mutedColor}>
              <Icon as={FiDatabase} w="13px" h="13px" />
              <Text>{t('fields.diskUsage') || 'Disk'}</Text>
            </HStack>
            <Text fontWeight="bold" color={textColor}>
              {diskVal !== undefined ? `${diskVal.toFixed(1)}%` : 'N/A'}
            </Text>
          </Flex>
          <Progress
            value={diskVal ?? 0}
            size="xs"
            borderRadius="full"
            mt="4px"
            colorScheme={getMetricColor(diskVal, 'cyan')}
            bg="rgba(0,0,0,0.06)"
          />
        </Box>
      </Flex>

      {/* Error Banner */}
      {!frozen && node.status === 'error' && node.description && (
        <Box
          bg="rgba(241, 70, 90, 0.11)"
          color="red.600"
          borderRadius="7px"
          p="6px 8px"
          fontSize="9px"
          noOfLines={1}
          title={node.description}
        >
          {node.description}
        </Box>
      )}

      {/* Circular Rings */}
      {!compact && (
        <Flex
          justify="space-between"
          align="center"
          pt="12px"
          borderTop="1px solid"
          borderColor={borderColor}
          mt="auto"
        >
          {/* Ring 1: Bandwidth / Egress */}
          <Flex direction="column" align="center" gap="4px" flex="1">
            <Box position="relative" display="inline-flex">
              <CircularProgress
                value={frozen ? 0 : egressUsage}
                size="40px"
                thickness="8px"
                color={getMetricColor(egressUsage, 'blue')}
                trackColor="rgba(0,0,0,0.05)"
              />
              <Box
                position="absolute"
                inset="0"
                display="flex"
                alignItems="center"
                justifyContent="center"
                color={getMetricColor(egressUsage, 'blue')}
              >
                <Icon as={FiArrowUp} w="12px" h="12px" />
              </Box>
            </Box>
            <Text color={mutedColor} fontSize="8px" fontWeight="bold" noOfLines={1}>
              {frozen ? '0 B/s' : formatBandwidth(Number(node.egress_bps))}
            </Text>
          </Flex>

          {/* Ring 2: Decoders */}
          <Flex direction="column" align="center" gap="4px" flex="1">
            <Box position="relative" display="inline-flex">
              <CircularProgress
                value={frozen ? 0 : decodeUsage}
                size="40px"
                thickness="8px"
                color="green.400"
                trackColor="rgba(0,0,0,0.05)"
              />
              <Box
                position="absolute"
                inset="0"
                display="flex"
                alignItems="center"
                justifyContent="center"
                color="green.400"
              >
                <Icon as={FiVideo} w="12px" h="12px" />
              </Box>
            </Box>
            <Text color={mutedColor} fontSize="8px" fontWeight="bold" noOfLines={1}>
              {frozen ? '-' : `${node.decode_slots_used ?? 0} / ${node.hardware_info?.cpu_cores ? node.hardware_info.cpu_cores * 2 : '-'}`}
            </Text>
          </Flex>

          {/* Ring 3: Load / Tasks */}
          <Flex direction="column" align="center" gap="4px" flex="1">
            <Box position="relative" display="inline-flex">
              <CircularProgress
                value={frozen ? 0 : loadUsage}
                size="40px"
                thickness="8px"
                color="blue.400"
                trackColor="rgba(0,0,0,0.05)"
              />
              <Box
                position="absolute"
                inset="0"
                display="flex"
                alignItems="center"
                justifyContent="center"
                color="blue.400"
              >
                <Icon as={FiPlay} w="12px" h="12px" />
              </Box>
            </Box>
            <Text color={mutedColor} fontSize="8px" fontWeight="bold" noOfLines={1}>
              {node.current_load ?? 0} / {node.max_load ?? '-'}
            </Text>
          </Flex>

          {/* Ring 4: Uptime */}
          <Flex direction="column" align="center" gap="4px" flex="1">
            <Box position="relative" display="inline-flex">
              <CircularProgress
                value={frozen ? 0 : 100}
                size="40px"
                thickness="8px"
                color="teal.400"
                trackColor="rgba(0,0,0,0.05)"
              />
              <Box
                position="absolute"
                inset="0"
                display="flex"
                alignItems="center"
                justifyContent="center"
                color="teal.400"
              >
                <Icon as={FiClock} w="12px" h="12px" />
              </Box>
            </Box>
            <Text color={mutedColor} fontSize="8px" fontWeight="bold" noOfLines={1}>
              {frozen ? '-' : formatUptime(Number(node.uptime))}
            </Text>
          </Flex>
        </Flex>
      )}

      {/* Footer */}
      <Flex
        justify="space-between"
        align="center"
        fontSize="8.5px"
        color={mutedColor}
        mt={compact ? '4px' : '8px'}
      >
        <Text noOfLines={1}>
          {node.metrics_received_at
            ? `${t('fields.lastHeartbeat')}: ${new Date(node.metrics_received_at).toLocaleTimeString()}`
            : (node.uptime ? `${t('fields.lastHeartbeat')}: 刚刚` : '-')}
        </Text>
        <HStack spacing="2px" color={textColor} fontWeight="bold">
          <Text>{t('actions.detail') || 'Manage'}</Text>
          <Icon as={FiChevronRight} />
        </HStack>
      </Flex>
    </Box>
  );
}
