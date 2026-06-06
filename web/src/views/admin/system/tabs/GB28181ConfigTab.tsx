import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box, Text, VStack, Table, Thead, Tbody, Tr, Th, Td, TableContainer,
  Badge, Spinner, Alert, AlertIcon,
} from '@chakra-ui/react';
import { getGB28181Config } from '../../../../services/gb28181';

export default function GB28181ConfigTab() {
  const { t } = useTranslation('modules/gb28181');
  const [config, setConfig] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getGB28181Config()
      .then(data => setConfig(data || {}))
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spinner />;
  if (error) return <Alert status="error"><AlertIcon />{error}</Alert>;

  const configKeys = Object.keys(config);

  return (
    <Box>
      <Text fontSize="lg" fontWeight="bold" mb={4}>{t('config.title')}</Text>
      <Alert status="info" mb={4}>
        <AlertIcon />
        {t('config.readOnly')}
      </Alert>
      <TableContainer>
        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>{t('config.configItem')}</Th>
              <Th>{t('config.configValue')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {configKeys.map(key => (
              <Tr key={key}>
                <Td fontWeight="medium">{key}</Td>
                <Td>{String(config[key])}</Td>
              </Tr>
            ))}
            {configKeys.length === 0 && (
              <Tr><Td colSpan={2} textAlign="center">{t('config.noData')}</Td></Tr>
            )}
          </Tbody>
        </Table>
      </TableContainer>
    </Box>
  );
}
