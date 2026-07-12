import {
  Box,
  Center,
  Flex,
  SimpleGrid,
  Spinner,
  Stat,
  StatHelpText,
  StatLabel,
  StatNumber,
  Text,
  useColorModeValue,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { edgeNodeApi, type NodeOverview } from 'services/edgeNode';

export default function EdgeNodeOverview() {
  const { t } = useTranslation('modules/edge-nodes');
  const textColor = useColorModeValue('navy.700', 'white');
  const textColorSecondary = useColorModeValue('gray.600', 'gray.400');

  const [overview, setOverview] = useState<NodeOverview | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchOverview = useCallback(async () => {
    setLoading(true);
    try {
      const data = await edgeNodeApi.getOverview();
      setOverview(data);
    } catch {
      // Silently fail; overview data is non-critical
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOverview();
  }, [fetchOverview]);

  if (loading) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px">
          <Spinner size="xl" color="brand.500" />
        </Center>
      </Box>
    );
  }

  if (!overview) {
    return (
      <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
        <Center h="400px">
          <Text color={textColorSecondary}>{t('message.noData')}</Text>
        </Center>
      </Box>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" gap="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>
          {t('overview')}
        </Text>

        <SimpleGrid columns={{ base: 1, sm: 2, md: 4 }} spacing="20px">
          {/* Total Nodes */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewTotal')}</StatLabel>
              <StatNumber fontSize="3xl" color={textColor}>
                {overview.total}
              </StatNumber>
              <StatHelpText>{t('fields.node')}</StatHelpText>
            </Stat>
          </Card>

          {/* Online */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewOnline')}</StatLabel>
              <StatNumber fontSize="3xl" color="green.500">
                {overview.online}
              </StatNumber>
              <StatHelpText color="green.500">{t('status.online')}</StatHelpText>
            </Stat>
          </Card>

          {/* Offline */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewOffline')}</StatLabel>
              <StatNumber fontSize="3xl" color="gray.500">
                {overview.offline}
              </StatNumber>
              <StatHelpText color="gray.500">{t('status.offline')}</StatHelpText>
            </Stat>
          </Card>

          {/* Error Count */}
          <Card p={4}>
            <Stat>
              <StatLabel color={textColorSecondary}>{t('overviewError')}</StatLabel>
              <StatNumber fontSize="3xl" color="red.500">
                {overview.error_count}
              </StatNumber>
              <StatHelpText color="red.500">{t('status.error')}</StatHelpText>
            </Stat>
          </Card>
        </SimpleGrid>
      </Flex>
    </Box>
  );
}
