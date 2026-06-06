import {
  Box,
  Button,
  Flex,
  FormControl,
  FormHelperText,
  FormLabel,
  Input,
  Spinner,
  Stack,
  Text,
  Textarea,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import Card from 'components/card/Card';
import { systemInfoApi, type SystemInfo } from 'services/api';

export default function SystemInfoTab() {
  const { t } = useTranslation('modules/system');
  const toast = useToast();
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const inputBorderColor = useColorModeValue('gray.200', 'whiteAlpha.200');
  const textColor = useColorModeValue('navy.700', 'white');
  const hintColor = useColorModeValue('gray.500', 'gray.400');

  const [info, setInfo] = useState<SystemInfo>({
    device_model: '',
    deploy_location: '',
    description: '',
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadInfo();
  }, []);

  async function loadInfo() {
    setLoading(true);
    try {
      const data = await systemInfoApi.get();
      setInfo({
        device_model: data?.device_model || '',
        deploy_location: data?.deploy_location || '',
        description: data?.description || '',
      });
    } catch (err) {
      toast({
        title: t('info.load_failed', { defaultValue: '加载系统信息失败' }),
        status: 'error',
        duration: 3000,
      });
    } finally {
      setLoading(false);
    }
  }

  async function handleSave() {
    setSaving(true);
    try {
      await systemInfoApi.save(info);
      toast({
        title: t('info.save_success', { defaultValue: '保存成功' }),
        status: 'success',
        duration: 3000,
      });
    } catch (err) {
      toast({
        title: t('info.save_failed', { defaultValue: '保存失败' }),
        description: err instanceof Error ? err.message : '',
        status: 'error',
        duration: 3000,
      });
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <Flex justify="center" align="center" h="200px">
        <Spinner size="xl" color="brand.500" />
      </Flex>
    );
  }

  return (
    <Stack spacing={6}>
      <Card p={6} bg={bgCard} borderColor={borderColor}>
        <Stack spacing={5}>
          <Box>
            <Text fontWeight="600" fontSize="lg" color={textColor}>
              {t('info.title', { defaultValue: '系统信息' })}
            </Text>
            <Text fontSize="sm" color={hintColor} mt={1}>
              {t('info.description', {
                defaultValue: '配置可由管理员编辑的设备元数据。这些信息会显示在运行状态页的系统信息卡片中。',
              })}
            </Text>
          </Box>

          <FormControl>
            <FormLabel fontSize="sm" fontWeight="600" color={textColor}>
              {t('info.device_model', { defaultValue: '设备型号' })}
            </FormLabel>
            <Input
              value={info.device_model}
              onChange={(e) => setInfo({ ...info, device_model: e.target.value })}
              placeholder={t('info.device_model_placeholder', {
                defaultValue: '如：RK3588、昇腾 Atlas 300I Pro',
              })}
              borderColor={inputBorderColor}
              borderRadius="lg"
            />
            <FormHelperText fontSize="xs" color={hintColor}>
              {t('info.device_model_hint', {
                defaultValue: '留空则使用硬件自动检测结果（如 RK3588 / 昇腾）。',
              })}
            </FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel fontSize="sm" fontWeight="600" color={textColor}>
              {t('info.deploy_location', { defaultValue: '部署位置' })}
            </FormLabel>
            <Input
              value={info.deploy_location}
              onChange={(e) => setInfo({ ...info, deploy_location: e.target.value })}
              placeholder={t('info.deploy_location_placeholder', {
                defaultValue: '如：上海数据中心 / 厂区 A 区门岗',
              })}
              borderColor={inputBorderColor}
              borderRadius="lg"
            />
          </FormControl>

          <FormControl>
            <FormLabel fontSize="sm" fontWeight="600" color={textColor}>
              {t('info.system_description', { defaultValue: '系统描述' })}
            </FormLabel>
            <Textarea
              value={info.description}
              onChange={(e) => setInfo({ ...info, description: e.target.value })}
              placeholder={t('info.system_description_placeholder', {
                defaultValue: '该系统的用途、责任团队、备注信息……',
              })}
              rows={3}
              borderColor={inputBorderColor}
              borderRadius="lg"
            />
          </FormControl>

          <Flex justify="flex-end">
            <Button
              colorScheme="brand"
              onClick={handleSave}
              isLoading={saving}
              borderRadius="lg"
            >
              {t('info.save', { defaultValue: '保存' })}
            </Button>
          </Flex>
        </Stack>
      </Card>
    </Stack>
  );
}
