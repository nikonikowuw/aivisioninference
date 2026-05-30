import { Box, SimpleGrid, Text, Icon, useColorModeValue, useToast, Spinner, Center } from '@chakra-ui/react';
import type { IconType } from 'react-icons/lib';
import { useTranslation } from 'react-i18next';
import { useEffect, useState } from 'react';
import { MdPerson, MdFolder, MdAssignment } from 'react-icons/md';
import MiniStatistics from 'components/card/MiniStatistics';
import IconBox from 'components/icons/IconBox';
import { dashboardApi, type DashboardStats } from 'services/api';
import UserGrowthChart from 'views/admin/default/components/UserGrowthChart';
import AuditTable from 'views/admin/default/components/AuditTable';

export default function Dashboard() {
  const { t } = useTranslation('modules/dashboard');
  const brandColor = useColorModeValue('brand.500', 'white');
  const boxBg = useColorModeValue('secondaryGray.300', 'whiteAlpha.100');
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  const toast = useToast();

  const statCards: Array<{ name: string; value: number; icon: IconType; gradient?: boolean }> = [
    { name: t('stats.totalUsers'), value: stats?.total_users ?? 0, icon: MdPerson },
    { name: t('stats.totalFiles'), value: stats?.total_files ?? 0, icon: MdFolder },
    { name: t('stats.activeTasks'), value: stats?.active_tasks ?? 0, icon: MdAssignment, gradient: true },
  ];

  useEffect(() => {
    let cancelled = false;

    setLoading(true);
    dashboardApi
      .stats()
      .then((data) => {
        if (!cancelled) setStats(data);
      })
      .catch(() => {
        if (!cancelled) toast({ title: t('message.loadFailed'), status: 'error' });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [toast, t]);

  if (loading) {
    return (
      <Center h="400px">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Text fontSize="2xl" fontWeight="bold" mb="20px">
        {t('title')}
      </Text>
      <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap="20px" mb="20px">
        {statCards.map((card) => (
          <MiniStatistics
            key={card.name}
            startContent={
              <IconBox
                w="56px"
                h="56px"
                bg={card.gradient ? 'linear-gradient(90deg, #4481EB 0%, #04BEFE 100%)' : boxBg}
                icon={
                  <Icon
                    w={card.gradient ? '28px' : '32px'}
                    h={card.gradient ? '28px' : '32px'}
                    as={card.icon}
                    color={card.gradient ? 'white' : brandColor}
                  />
                }
              />
            }
            name={card.name}
            value={String(card.value)}
          />
        ))}
      </SimpleGrid>

      <Box mb="20px">
        <UserGrowthChart chartData={stats?.user_stats} />
      </Box>
      <Box mb="20px">
        <AuditTable tableData={stats?.audit_logs} />
      </Box>
    </Box>
  );
}
