import { useRef, useEffect } from 'react';
import { useColorModeValue } from '@chakra-ui/react';
import * as echarts from 'echarts/core';
import { GaugeChart } from 'echarts/charts';
import {
  TooltipComponent,
  TooltipComponentOption,
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

echarts.use([GaugeChart, TooltipComponent, CanvasRenderer]);

// 基础色值定义
const C_GREEN = '#00FFA3'; // 最健康（青绿）
const D_GREEN = '#276749'; // 深绿
const O_RANGE = '#FF6B00'; // 橙
const L_ORANGE = '#FFA833'; // 浅橙
const RED = '#E53E3E';    // 红色

function getColorPair(percent: number): [string, string] {
  if (percent > 80) return [RED, '#FC8181'];
  if (percent > 50) return [O_RANGE, L_ORANGE];
  if (percent > 25) return [D_GREEN, '#48BB78'];
  return [C_GREEN, '#7AFFB8'];
}

interface EChartsRadialProps {
  percent: number;
  detail?: string;
}

export default function EChartsRadial({ percent, detail }: EChartsRadialProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const instanceRef = useRef<echarts.ECharts | null>(null);
  const [mainColor] = getColorPair(percent);
  const trackColor = useColorModeValue('#E2E8F0', '#2D3748');
  const valueColor = useColorModeValue('#1A202C', 'white');

  useEffect(() => {
    if (!chartRef.current) return;
    if (!instanceRef.current) {
      instanceRef.current = echarts.init(chartRef.current, undefined, {
        renderer: 'canvas',
      });
    }
    const chart = instanceRef.current;

    const [startColor, endColor] = getColorPair(percent);

    chart.setOption({
      series: [{
        type: 'gauge',
        startAngle: 180,
        endAngle: 0,
        min: 0,
        max: 100,
        pointer: { show: false },
        progress: {
          show: true,
          roundCap: true,
          width: 12,
          itemStyle: {
            color: {
              type: 'linear',
              x: 0, y: 0, x2: 1, y2: 0,
              colorStops: [
                { offset: 0, color: startColor },
                { offset: 1, color: endColor },
              ],
              global: false,
            },
          },
        },
        axisLine: {
          lineStyle: {
            width: 12,
            color: [[1, trackColor]],
            roundCap: true,
          },
        },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        detail: {
          show: true,
          offsetCenter: [0, '30%'],
          valueAnimation: true,
          fontSize: 20,
          fontWeight: 700,
          color: valueColor,
          formatter: () => `${percent.toFixed(1)}%`,
        },
        data: [{ value: percent }],
      }],
    }, true);

    return () => {
      if (instanceRef.current) {
        instanceRef.current.dispose();
        instanceRef.current = null;
      }
    };
  }, [percent, trackColor, valueColor]);

  useEffect(() => {
    return () => {
      if (instanceRef.current) {
        instanceRef.current.dispose();
        instanceRef.current = null;
      }
    };
  }, []);

  return <div ref={chartRef} style={{ width: '100%', height: '100%' }} />;
}
