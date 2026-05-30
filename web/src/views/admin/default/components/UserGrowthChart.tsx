import { ApexOptions } from 'apexcharts';
import { Box, Flex, Icon, Text, useColorModeValue } from '@chakra-ui/react';
import Card from 'components/card/Card';
import LineChart from 'components/charts/LineChart';
import { useTranslation } from 'react-i18next';
import { MdTimeline } from 'react-icons/md';
import type { DashboardUserStat } from 'services/api';

export default function UserGrowthChart(props: { chartData?: DashboardUserStat[] }) {
  const { chartData = [] } = props;
  const { t } = useTranslation('modules/dashboard');
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const iconColor = useColorModeValue('brand.500', 'white');

  const lineChartData = [
    {
      name: t('charts.userNew'),
      data: chartData.map((d) => d.new)
    },
    {
      name: t('charts.userActive'),
      data: chartData.map((d) => d.active)
    }
  ];

  const lineChartOptions: ApexOptions = {
    chart: {
      type: 'area',
      toolbar: {
        show: false
      },
      dropShadow: {
        enabled: true,
        top: 13,
        left: 0,
        blur: 10,
        opacity: 0.1,
        color: '#4318FF'
      }
    },
    colors: ['#4318FF', '#05CD99'],
    markers: {
      size: 4,
      colors: 'white',
      strokeColors: ['#4318FF', '#05CD99'],
      strokeWidth: 2,
      strokeOpacity: 0.9,
      strokeDashArray: 0,
      fillOpacity: 1,
      discrete: [],
      shape: 'circle',
      radius: 2,
      offsetX: 0,
      offsetY: 0,
      showNullDataPoints: true
    },
    tooltip: {
      theme: 'dark'
    },
    dataLabels: {
      enabled: false
    },
    stroke: {
      curve: 'smooth',
      type: 'line',
      width: 3
    },
    xaxis: {
      categories: chartData.map((d) => d.date),
      labels: {
        style: {
          colors: '#A3AED0',
          fontSize: '12px',
          fontWeight: '500'
        }
      },
      axisBorder: {
        show: false
      },
      axisTicks: {
        show: false
      }
    },
    yaxis: {
      show: true,
      labels: {
        style: {
          colors: '#A3AED0',
          fontSize: '12px',
          fontWeight: '500'
        }
      }
    },
    legend: {
      show: true,
      position: 'top',
      horizontalAlign: 'right',
      fontFamily: 'inherit',
      fontWeight: 500,
      labels: {
        colors: '#A3AED0'
      }
    },
    grid: {
      show: true,
      borderColor: 'rgba(163, 174, 208, 0.3)',
      strokeDashArray: 5,
      yaxis: {
        lines: {
          show: true
        }
      },
      xaxis: {
        lines: {
          show: false
        }
      }
    },
    fill: {
      type: 'gradient',
      gradient: {
        shade: 'light',
        type: 'vertical',
        shadeIntensity: 0.5,
        inverseColors: true,
        opacityFrom: 0.8,
        opacityTo: 0.1,
        stops: [0, 100]
      }
    }
  };

  return (
    <Card p='20px' alignItems='center' flexDirection='column' w='100%'>
      <Flex align='center' justify='space-between' w='100%' mb='20px'>
        <Flex align='center'>
          <Icon as={MdTimeline} color={iconColor} w='24px' h='24px' me='10px' />
          <Text color={textColor} fontSize='lg' fontWeight='700'>
            {t('charts.userStats')}
          </Text>
        </Flex>
      </Flex>
      <Box minH='260px' w='100%'>
        <LineChart chartData={lineChartData} chartOptions={lineChartOptions} />
      </Box>
    </Card>
  );
}
