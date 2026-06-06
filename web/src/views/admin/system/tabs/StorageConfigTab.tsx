import {
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Input,
  NumberInput,
  NumberInputField,
  Radio,
  RadioGroup,
  Spinner,
  Stack,
  Switch,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from 'services/api';

type CleanupMode = 'cron' | 'threshold';

interface StorageConfig {
  enabled: boolean;
  cleanup_mode: CleanupMode;
  retention_days: number;
  cron_expression: string;
  threshold_value: number;
  target_percentage: number;
  threshold_check_cron: string;
}

interface CleanupLog {
  id: string;
  started_at: string;
  completed_at?: string;
  status: string;
  cleaned_records: number;
  cleaned_files: number;
  freed_space: number;
  error_message?: string;
}

interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

export default function StorageConfigTab() {
  const { t } = useTranslation(['modules/system', 'common']);
  const toast = useToast();
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const [config, setConfig] = useState<StorageConfig | null>(null);
  const [logs, setLogs] = useState<CleanupLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  async function loadData() {
    setLoading(true);
    try {
      await Promise.all([loadConfig(), loadLogs()]);
    } finally {
      setLoading(false);
    }
  }

  async function loadConfig() {
    try {
      const data = await request<StorageConfig>('/system/storage/config');
      setConfig(data);
    } catch (err) {
      console.error('Failed to load storage config:', err);
    }
  }

  async function loadLogs() {
    try {
      const data = await request<PaginatedResponse<CleanupLog>>('/system/storage/cleanup-logs?page_size=10');
      setLogs(data?.list || []);
    } catch (err) {
      console.error('Failed to load cleanup logs:', err);
    }
  }

  function update(fields: Partial<StorageConfig>) {
    if (!config) return;
    setConfig({ ...config, ...fields });
  }

  async function handleSave() {
    if (!config) return;

    // 阈值模式校验
    if (config.cleanup_mode === 'threshold') {
      if (config.threshold_value <= config.target_percentage) {
        toast({
          title: t('storage.error_threshold_overlap', { defaultValue: '触发阈值必须大于目标水位' }),
          status: 'warning',
          duration: 5000,
          isClosable: true,
        });
        return;
      }
    }

    setSaving(true);
    try {
      await request('/system/storage/config', {
        method: 'PUT',
        body: JSON.stringify(config),
      });
      toast({ title: t('storage.saved'), status: 'success', duration: 3000 });
    } catch (err: any) {
      toast({ title: err.message || t('storage.save_failed'), status: 'error', duration: 3000 });
    } finally {
      setSaving(false);
    }
  }

  async function handleRunCleanup() {
    try {
      await request('/system/storage/cleanup/run', { method: 'POST' });
      toast({ title: t('storage.cleanup_started'), status: 'success', duration: 3000 });
      loadLogs();
    } catch (err: any) {
      toast({ title: err.message || t('storage.cleanup_failed'), status: 'error', duration: 3000 });
    }
  }

  if (loading) {
    return (
      <Flex justify="center" align="center" h="200px">
        <Spinner size="xl" />
      </Flex>
    );
  }

  return (
    <Box>
      {/* 配置表单 */}
      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor} mb={6}>
        <Text fontWeight="600" mb={4}>{t('storage.config')}</Text>

        <Flex direction="column" gap={4}>
          {/* 启用总开关 */}
          <FormControl display="flex" alignItems="center">
            <FormLabel mb={0}>{t('storage.enabled')}</FormLabel>
            <Switch
              isChecked={config?.enabled}
              onChange={(e) => update({ enabled: e.target.checked })}
            />
          </FormControl>

          {config?.enabled && (
            <>
              {/* 清理策略选择（互斥双模式） */}
              <FormControl>
                <FormLabel>{t('storage.cleanup_mode')}</FormLabel>
                <RadioGroup value={config?.cleanup_mode} onChange={(v: CleanupMode) => update({ cleanup_mode: v })}>
                  <Stack direction="row" spacing={6}>
                    <Radio value="cron">{t('storage.mode_cron')}</Radio>
                    <Radio value="threshold">{t('storage.mode_threshold')}</Radio>
                  </Stack>
                </RadioGroup>
              </FormControl>

              {/* CRON 定时清理参数 */}
              {config?.cleanup_mode === 'cron' && (
                <FormControl>
                  <FormLabel>{t('storage.cron_expression')}</FormLabel>
                  <Input
                    value={config?.cron_expression || '0 2 * * *'}
                    placeholder="0 2 * * *"
                    onChange={(e) => update({ cron_expression: e.target.value })}
                  />
                  <Text fontSize="xs" color="gray.500" mt={1}>
                    {t('storage.cron_expression_hint')}
                  </Text>
                </FormControl>
              )}

              {/* 阈值清理参数 */}
              {config?.cleanup_mode === 'threshold' && (
                <>
                  <Flex gap={4}>
                    <FormControl>
                      <FormLabel>{t('storage.threshold_value')}</FormLabel>
                      <NumberInput
                        value={config?.threshold_value || 90}
                        min={1}
                        max={100}
                        onChange={(_, v) => config && setConfig({ ...config, threshold_value: v })}
                      >
                        <NumberInputField />
                      </NumberInput>
                    </FormControl>
                    <FormControl>
                      <FormLabel>{t('storage.target_percentage')}</FormLabel>
                      <NumberInput
                        value={config?.target_percentage || 70}
                        min={0}
                        max={100}
                        onChange={(_, v) => update({ target_percentage: v })}
                      >
                        <NumberInputField />
                      </NumberInput>
                    </FormControl>
                  </Flex>
                  <FormControl>
                    <FormLabel>{t('storage.threshold_check_cron')}</FormLabel>
                    <Input
                      value={config?.threshold_check_cron || '*/5 * * * *'}
                      placeholder="*/5 * * * *"
                      onChange={(e) => update({ threshold_check_cron: e.target.value })}
                    />
                    <Text fontSize="xs" color="gray.500" mt={1}>
                      {t('storage.threshold_check_cron_hint')}
                    </Text>
                  </FormControl>
                </>
              )}

              {/* 保留天数 */}
              <FormControl>
                <FormLabel>{t('storage.retention_days')}</FormLabel>
                <NumberInput
                  value={config?.retention_days || 0}
                  min={0}
                  onChange={(_, v) => update({ retention_days: v })}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>
            </>
          )}

          <Flex gap={4}>
            <Button colorScheme="brand" onClick={handleSave} isLoading={saving}>
              {t('common:button.save')}
            </Button>
            <Button onClick={handleRunCleanup}>
              {t('storage.run_cleanup')}
            </Button>
          </Flex>
        </Flex>
      </Box>

      {/* 清理日志 */}
      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
        <Text fontWeight="600" mb={4}>{t('storage.cleanup_logs')}</Text>
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>{t('storage.time')}</Th>
              <Th>{t('storage.status')}</Th>
              <Th>{t('storage.cleaned_records')}</Th>
              <Th>{t('storage.freed_space')}</Th>
              <Th>{t('storage.error')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {logs?.map((log) => (
              <Tr key={log.id}>
                <Td>{new Date(log.started_at).toLocaleString()}</Td>
                <Td>
                  <Text color={log.status === 'success' ? 'green.500' : 'red.500'}>
                    {log.status === 'success' ? t('status.running') : t('status.failed')}
                  </Text>
                </Td>
                <Td>{log.cleaned_records}</Td>
                <Td>{(log.freed_space / 1024 / 1024).toFixed(2)} MB</Td>
                <Td maxW="200px" isTruncated>{log.error_message}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </Box>
  );
}
