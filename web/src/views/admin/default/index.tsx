import { Box, SimpleGrid, Text, Icon, Flex, useColorModeValue, useToast, Spinner, Center } from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { useEffect, useState } from 'react';
import { MdPerson, MdSecurity, MdFolder, MdAssignment } from 'react-icons/md';
import MiniStatistics from 'components/card/MiniStatistics';
import IconBox from 'components/icons/IconBox';
import { dashboardApi, type DashboardStats } from 'services/api';

export default function Dashboard() {
  const { t } = useTranslation('modules/dashboard');
  const brandColor = useColorModeValue('brand.500', 'white');
  const boxBg = useColorModeValue('secondaryGray.300', 'whiteAlpha.100');
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  const toast = useToast();

  useEffect(() => {
    dashboardApi
      .stats()
      .then(setStats)
      .catch(() => {
        toast({ title: t('message.loadFailed'), status: 'error' });
      })
      .finally(() => setLoading(false));
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
      <SimpleGrid columns={{ base: 1, md: 2, lg: 4 }} gap="20px" mb="20px">
        <MiniStatistics
          startContent={
            <IconBox
              w="56px"
              h="56px"
              bg={boxBg}
              icon={
                <Icon w="32px" h="32px" as={MdPerson} color={brandColor} />
              }
            />
          }
          name={t('stats.totalUsers')}
          value={String(stats?.total_users ?? 0)}
        />
        <MiniStatistics
          startContent={
            <IconBox
              w="56px"
              h="56px"
              bg={boxBg}
              icon={
                <Icon w="32px" h="32px" as={MdSecurity} color={brandColor} />
              }
            />
          }
          name={t('stats.totalRoles')}
          value={String(stats?.total_roles ?? 0)}
        />
        <MiniStatistics
          startContent={
            <IconBox
              w="56px"
              h="56px"
              bg={boxBg}
              icon={
                <Icon w="32px" h="32px" as={MdFolder} color={brandColor} />
              }
            />
          }
          name={t('stats.totalFiles')}
          value={String(stats?.total_files ?? 0)}
        />
        <MiniStatistics
          startContent={
            <IconBox
              w="56px"
              h="56px"
              bg="linear-gradient(90deg, #4481EB 0%, #04BEFE 100%)"
              icon={
                <Icon w="28px" h="28px" as={MdAssignment} color="white" />
              }
            />
          }
          name={t('stats.activeTasks')}
          value={String(stats?.active_tasks ?? 0)}
        />
      </SimpleGrid>
    </Box>
  );
}
