import { Box, Text, useColorModeValue } from '@chakra-ui/react';
import { useTheme } from '@chakra-ui/react';
import { useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { MetricDataPoint } from 'services/edgeNodeMetrics';

export interface MetricsTimeSeriesProps {
  /** Chart title */
  title: string;
  /** Time-series data points */
  data: MetricDataPoint[];
  /** Loading state */
  loading?: boolean;
  /** Unit suffix for the Y-axis */
  unit?: string;
  /** Chart color (Chakra color scheme name, e.g. 'blue', 'green') */
  colorScheme?: string;
  /** Height of the chart in pixels */
  height?: number;
  /** Format Y-axis value */
  formatValue?: (value: number) => string;
  /** Format tooltip label */
  formatTooltipLabel?: (value: number) => string;
}

/**
 * MetricsTimeSeries — 可重用的时间序列折线图组件
 * 基于 Recharts 实现，支持面积图展示历史指标趋势。
 */
export default function MetricsTimeSeries({
  title,
  data,
  loading = false,
  unit = '',
  colorScheme = 'blue',
  height = 300,
  formatValue,
  formatTooltipLabel,
}: MetricsTimeSeriesProps) {
  const { t } = useTranslation('common');
  const theme = useTheme();
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const textColor = useColorModeValue('gray.600', 'gray.300');

  // Map Chakra color scheme to actual color values
  const chartColor = useMemo(() => {
    const colors: Record<string, string> = {
      blue: theme.colors.blue?.[400] || '#3182CE',
      green: theme.colors.green?.[400] || '#38A169',
      red: theme.colors.red?.[400] || '#E53E3E',
      orange: theme.colors.orange?.[400] || '#DD6B20',
      purple: theme.colors.purple?.[400] || '#805AD5',
      teal: theme.colors.teal?.[400] || '#319795',
    };
    return colors[colorScheme] || colors.blue;
  }, [colorScheme, theme]);

  const chartGradientId = useMemo(
    () => `gradient-${colorScheme}-${title.replace(/\s+/g, '-')}`,
    [colorScheme, title],
  );

  const formatXAxis = useCallback((tick: string) => {
    try {
      const d = new Date(tick);
      if (isNaN(d.getTime())) return tick;
      return `${d.getHours().toString().padStart(2, '0')}:${d.getMinutes().toString().padStart(2, '0')}`;
    } catch {
      return tick;
    }
  }, []);

  const formatYAxis = useCallback(
    (value: number) => {
      if (formatValue) return formatValue(value);
      if (unit === '%') return `${Math.round(value)}%`;
      if (unit === 'B/s' || unit === 'bytes/s') {
        if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)} MB/s`;
        if (value >= 1_000) return `${(value / 1_000).toFixed(1)} KB/s`;
        return `${Math.round(value)} B/s`;
      }
      if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}G`;
      if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
      if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
      return value.toFixed(1);
    },
    [formatValue, unit],
  );

  const renderTooltip = useCallback(
    ({ active, payload }: any) => {
      if (!active || !payload?.length) return null;
      const point = payload[0].payload as MetricDataPoint;
      const time = new Date(point.t).toLocaleString();
      const val = formatTooltipLabel
        ? formatTooltipLabel(point.v)
        : `${point.v.toFixed(2)}${unit ? ` ${unit}` : ''}`;
      return (
        <Box
          bg={useColorModeValue('white', 'gray.800')}
          border="1px solid"
          borderColor={borderColor}
          borderRadius="md"
          p={2}
          shadow="md"
          fontSize="sm"
        >
          <Text>{time}</Text>
          <Text fontWeight="bold" color={chartColor}>
            {val}
          </Text>
        </Box>
      );
    },
    [borderColor, chartColor, formatTooltipLabel, unit],
  );

  if (loading) {
    return (
      <Box h={`${height}px`} display="flex" alignItems="center" justifyContent="center">
        <Text color="gray.400">{t('loading')}</Text>
      </Box>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Box h={`${height}px`} display="flex" alignItems="center" justifyContent="center">
        <Text color="gray.400">{t('noData')}</Text>
      </Box>
    );
  }

  return (
    <Box>
      <Text fontSize="sm" fontWeight="500" color={textColor} mb={1}>
        {title}
      </Text>
      <Box h={`${height}px`}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
            <defs>
              <linearGradient id={chartGradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={chartColor} stopOpacity={0.3} />
                <stop offset="95%" stopColor={chartColor} stopOpacity={0.0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke={borderColor} />
            <XAxis
              dataKey="t"
              tickFormatter={formatXAxis}
              stroke={textColor}
              fontSize={11}
              tickLine={false}
              axisLine={false}
              minTickGap={30}
            />
            <YAxis
              tickFormatter={formatYAxis}
              stroke={textColor}
              fontSize={11}
              tickLine={false}
              axisLine={false}
              width={60}
            />
            <Tooltip content={renderTooltip} />
            <Area
              type="monotone"
              dataKey="v"
              stroke={chartColor}
              strokeWidth={2}
              fill={`url(#${chartGradientId})`}
              dot={false}
              activeDot={{ r: 4, strokeWidth: 0 }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </Box>
    </Box>
  );
}
