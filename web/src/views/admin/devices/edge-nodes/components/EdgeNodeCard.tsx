import {
  Box,
  Flex,
  Text,
  Tag,
  Progress,
  Icon,
  HStack,
  VStack,
  useColorModeValue,
  keyframes,
} from '@chakra-ui/react';
import {
  FiCpu,
  FiZap,
  FiDatabase,
  FiClock,
  FiArrowUp,
  FiArrowDown,
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

/** 呼吸光晕动画：用于 online 状态徽章前的指示点 */
const pulseGlow = keyframes`
  0%, 100% { box-shadow: 0 0 0 0 rgba(1, 181, 116, 0.55); }
  50%      { box-shadow: 0 0 6px 3px rgba(1, 181, 116, 0.25); }
`;

const statusConfig: Record<string, { colorScheme: string; dotColor: string; glow: boolean }> = {
  online:   { colorScheme: 'green',  dotColor: '#01B574', glow: true },
  offline:  { colorScheme: 'gray',   dotColor: '#A0AEC0', glow: false },
  error:    { colorScheme: 'red',    dotColor: '#EE5D50', glow: false },
  disabled: { colorScheme: 'orange', dotColor: '#FFB547', glow: false },
};

/**
 * 将字节/秒格式化为人类可读的网络速率字符串。
 * 输入单位为字节/秒 (bytes/sec)，使用 1024 进制（二进制前缀）。
 * IEC 标准对应名称为 KiB/s、MiB/s，保留 KB/s 作为常用别名。
 */
function formatNetworkSpeed(bytesPerSec?: number): string {
  if (bytesPerSec === undefined || bytesPerSec === null) return '-';
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`;
  const kbps = bytesPerSec / 1024;
  if (kbps < 1024) return `${kbps.toFixed(1)} KiB/s`;
  const mbps = kbps / 1024;
  return `${mbps.toFixed(1)} MiB/s`;
}

/** 阈值色：≥90 红 / ≥75 橙黄 / 默认保持各指标特征色 */
function getMetricColor(val?: number, normalColor = 'green'): string {
  if (val === undefined || val === null) return 'gray';
  if (val >= 90) return 'red';
  if (val >= 75) return 'orange';
  return normalColor;
}

/** 单条资源进度行的渲染 */
function MetricRow({
  icon,
  label,
  value,
  colorScheme,
}: {
  icon: React.ElementType;
  label: string;
  value?: number;
  colorScheme: string;
}) {
  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('secondaryGray.600', 'gray.400');
  const progressTrack = useColorModeValue('rgba(0, 0, 0, 0.05)', 'rgba(255, 255, 255, 0.06)');

  return (
    <Box>
      <Flex justify="space-between" align="center" mb="6px">
        <HStack spacing="5px" color={mutedColor}>
          <Icon as={icon} w="13px" h="13px" />
          <Text fontSize="10px" fontWeight="600">{label}</Text>
        </HStack>
        <Text
          fontSize="10px"
          fontWeight="bold"
          color={textColor}
          sx={{ fontVariantNumeric: 'tabular-nums' }}
        >
          {value !== undefined ? `${value.toFixed(1)}%` : 'N/A'}
        </Text>
      </Flex>
      <Progress
        value={value ?? 0}
        borderRadius="full"
        colorScheme={getMetricColor(value, colorScheme)}
        bg={progressTrack}
        h="6px"
        sx={{
          '& > div': {
            transition: 'width 0.6s cubic-bezier(0.22, 1, 0.36, 1)',
          },
        }}
      />
    </Box>
  );
}

/** Stats Bar 中单个 stat 单元 */
function StatCell({
  icon,
  iconColor,
  value,
  label,
}: {
  icon: React.ElementType;
  iconColor: string;
  value: string;
  label: string;
}) {
  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('secondaryGray.600', 'gray.400');
  const statsIconBg = useColorModeValue('rgba(0, 0, 0, 0.04)', 'rgba(255, 255, 255, 0.06)');

  return (
    <VStack spacing="4px" flex="1" py="10px" px="6px">
      <Flex
        w="28px"
        h="28px"
        borderRadius="8px"
        bg={statsIconBg}
        align="center"
        justify="center"
      >
        <Icon as={icon} w="14px" h="14px" color={iconColor} />
      </Flex>
      <Text
        fontSize="11px"
        fontWeight="bold"
        color={textColor}
        sx={{ fontVariantNumeric: 'tabular-nums' }}
        lineHeight="1"
      >
        {value}
      </Text>
      <Text fontSize="9px" color={mutedColor} lineHeight="1">
        {label}
      </Text>
    </VStack>
  );
}

export default function EdgeNodeCard({ node, compact = false, onClick, onDeployClick }: EdgeNodeCardProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const cardBg = useColorModeValue('rgba(255, 255, 255, 0.78)', 'rgba(22, 27, 45, 0.82)');
  const hoverBg = useColorModeValue('rgba(255, 255, 255, 0.92)', 'rgba(30, 37, 58, 0.92)');
  const borderColor = useColorModeValue('rgba(0, 0, 0, 0.06)', 'rgba(255, 255, 255, 0.08)');
  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('secondaryGray.600', 'gray.400');
  const statsBarBg = useColorModeValue('rgba(0, 0, 0, 0.025)', 'rgba(255, 255, 255, 0.035)');
  const statsDivider = useColorModeValue('rgba(0, 0, 0, 0.06)', 'rgba(255, 255, 255, 0.06)');

  const frozen = node.status === 'offline' || node.status === 'disabled';

  const status = statusConfig[node.status] || statusConfig.offline;

  // Host stats
  const cpuVal = frozen ? undefined : node.cpu_usage;
  const memVal = frozen ? undefined : node.memory_usage;

  const diskInfo = node.disk_usage && node.disk_usage.length > 0 ? node.disk_usage[0] : null;
  const diskVal = diskInfo ? diskInfo.percent : undefined;

  // Accelerator (NPU/GPU)
  const hasAcc = node.accelerator_metrics_valid;
  const accVal = frozen || !hasAcc
    ? undefined
    : (node.accelerator_utilization !== undefined && node.accelerator_utilization !== null
        ? Number(node.accelerator_utilization)
        : undefined);

  // 默认解码器容量 = CPU 核心数 × 2（常见边缘设备每核支持 2 路解码的近似估算）
  const decoderCapacity = node.hardware_info?.cpu_cores ? node.hardware_info.cpu_cores * 2 : null;

  return (
    <Box
      className={`node-card ${frozen ? 'offline' : ''}`}
      bg={cardBg}
      border="1px solid"
      borderColor={borderColor}
      borderRadius="16px"
      boxShadow="0 8px 32px rgba(0, 0, 0, 0.08)"
      backdropFilter="blur(24px) saturate(1.15)"
      cursor="pointer"
      transition="transform 220ms cubic-bezier(0.22, 1, 0.36, 1), box-shadow 220ms ease, background 180ms ease"
      _hover={{
        transform: 'translateY(-4px)',
        boxShadow: '0 16px 48px rgba(0, 0, 0, 0.14)',
        bg: hoverBg,
      }}
      onClick={onClick}
      p="18px"
      display="flex"
      flexDirection="column"
      gap="14px"
      opacity={frozen ? 0.68 : 1}
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
      {/* ─── Header ─── */}
      <Flex justify="space-between" align="flex-start" gap="8px">
        <Box minW="0" flex="1">
          <Text
            color={textColor}
            fontSize="14px"
            fontWeight="bold"
            noOfLines={1}
            title={node.name}
            letterSpacing="-0.3px"
          >
            {node.name}
          </Text>
          <Text
            color={mutedColor}
            fontSize="10px"
            noOfLines={1}
            mt="3px"
            title={`${node.hardware_info?.cpu_model || ''} · ${node.hardware_info?.gpu_model || ''}`}
          >
            {node.hardware_info?.cpu_model
              ? `${node.hardware_info.cpu_model} · ${node.hardware_info.gpu_model || ''}`
              : node.endpoint}
          </Text>
        </Box>
        <Tag
          colorScheme={status.colorScheme}
          variant="subtle"
          size="sm"
          borderRadius="full"
          fontSize="10px"
          fontWeight="bold"
          px="10px"
          py="3px"
          gap="5px"
        >
          {/* 状态指示点（online 时带呼吸动画） */}
          <Box
            w="6px"
            h="6px"
            borderRadius="full"
            bg={status.dotColor}
            animation={status.glow ? `${pulseGlow} 2s ease-in-out infinite` : undefined}
          />
          {t(`status.${node.status}`)}
        </Tag>
      </Flex>

      {/* ─── Progress Bars ─── */}
      <Flex direction="column" gap="10px">
        <MetricRow icon={FiCpu} label={t('fields.cpuUsage')} value={cpuVal} colorScheme="green" />
        <MetricRow icon={MdMemory} label={t('fields.memUsage')} value={memVal} colorScheme="blue" />
        <MetricRow icon={FiZap} label={hasAcc ? t('fields.npuUsage') : t('fields.gpuUsage')} value={accVal} colorScheme="purple" />
        <MetricRow icon={FiDatabase} label={t('fields.diskUsage')} value={diskVal} colorScheme="cyan" />
      </Flex>

      {/* ─── Error Banner ─── */}
      {!frozen && node.status === 'error' && node.description && (
        <Box
          bg="rgba(238, 93, 80, 0.1)"
          color="red.500"
          borderRadius="8px"
          px="10px"
          py="7px"
          fontSize="10px"
          noOfLines={1}
          title={node.description}
        >
          {node.description}
        </Box>
      )}

      {/* ─── Stats Bar（运行时指标横向展示） ─── */}
      {!compact && (
        <Box
          bg={statsBarBg}
          borderRadius="12px"
          border="1px solid"
          borderColor={statsDivider}
          mt="auto"
          overflow="hidden"
        >
          <Flex align="stretch">
            <StatCell
              icon={FiArrowDown}
              iconColor="blue.400"
              value={frozen ? '-' : formatNetworkSpeed(node.net_rx_speed)}
              label={t('metrics.netRx')}
            />
            <Box w="1px" bg={statsDivider} my="10px" />
            <StatCell
              icon={FiArrowUp}
              iconColor="blue.400"
              value={frozen ? '-' : formatNetworkSpeed(node.net_tx_speed)}
              label={t('metrics.netTx')}
            />
            <Box w="1px" bg={statsDivider} my="10px" />
            <StatCell
              icon={FiVideo}
              iconColor="green.400"
              value={frozen ? '-' : `${node.decode_slots_used ?? 0} / ${decoderCapacity ?? '-'}`}
              label={t('fields.decoders')}
            />
            <Box w="1px" bg={statsDivider} my="10px" />
            <StatCell
              icon={FiPlay}
              iconColor="brand.400"
              value={`${node.current_load ?? 0} / ${node.max_load ?? '-'}`}
              label={t('fields.tasks')}
            />
            <Box w="1px" bg={statsDivider} my="10px" />
            <StatCell
              icon={FiClock}
              iconColor="teal.400"
              value={frozen ? '-' : formatUptime(Number(node.uptime))}
              label={t('fields.uptime')}
            />
          </Flex>
        </Box>
      )}

      {/* ─── Footer ─── */}
      <Flex
        justify="space-between"
        align="center"
        fontSize="10px"
        color={mutedColor}
        mt={compact ? '2px' : '0'}
      >
        <Text noOfLines={1}>
          {node.metrics_received_at
            ? `${t('fields.lastHeartbeat')}: ${new Date(node.metrics_received_at).toLocaleTimeString()}`
            : (node.uptime ? `${t('fields.lastHeartbeat')}: 刚刚` : '-')}
        </Text>
        <HStack spacing="2px" color={textColor} fontWeight="bold" _hover={{ color: 'brand.400' }} transition="color 150ms">
          <Text>{t('actions.detail') || 'Manage'}</Text>
          <Icon as={FiChevronRight} />
        </HStack>
      </Flex>
    </Box>
  );
}
