import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Alert,
  AlertIcon,
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Input,
  Spinner,
  Text,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { getGB28181Config, updateGB28181Config, type GB28181Config } from '../../../../services/gb28181';

const DEFAULT_CONFIG_KEYS = ['sip.port', 'sip.id', 'sip.domain', 'sip.password'];

export default function GB28181ConfigTab() {
  const { t } = useTranslation('modules/gb28181');
  const toast = useToast();
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const [config, setConfig] = useState<GB28181Config>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    getGB28181Config()
      .then(data => setConfig(data || {}))
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  function update(key: string, value: string) {
    setConfig({ ...config, [key]: value });
  }

  async function handleSave() {
    setSaving(true);
    try {
      await updateGB28181Config(config);
      toast({ title: t('config.saved'), status: 'success', duration: 3000 });
    } catch (err: any) {
      toast({ title: err.message || t('config.saveFailed'), status: 'error', duration: 3000 });
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <Flex justify="center" align="center" h="200px">
        <Spinner size="xl" />
      </Flex>
    );
  }

  if (error) return <Alert status="error"><AlertIcon />{error}</Alert>;

  const configKeys = Array.from(new Set([...DEFAULT_CONFIG_KEYS, ...Object.keys(config)]));

  return (
    <Box>
      <Text fontSize="lg" fontWeight="bold" mb={4}>{t('config.title')}</Text>
      <Alert status="info" mb={4}>
        <AlertIcon />
        {t('config.restartHint')}
      </Alert>

      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
        <Flex direction="column" gap={4}>
          {configKeys.map(key => (
            <FormControl key={key}>
              <FormLabel>{t(`config.fields.${key}`, { defaultValue: key })}</FormLabel>
              <Input
                value={String(config[key] ?? '')}
                type={key.includes('password') ? 'password' : 'text'}
                placeholder={key}
                onChange={(event) => update(key, event.target.value)}
              />
            </FormControl>
          ))}

          {configKeys.length === 0 && <Text color="gray.500">{t('config.noData')}</Text>}

          <Button colorScheme="blue" alignSelf="flex-start" isLoading={saving} onClick={handleSave}>
            {t('common.save')}
          </Button>
        </Flex>
      </Box>
    </Box>
  );
}
