import {
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  HStack,
  Icon,
  Image,
  Input,
  Text,
  VStack,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useBrand } from 'contexts/BrandContext';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { MdDelete, MdUpload } from 'react-icons/md';
import { brandConfigApi, type BrandConfig } from 'services/api';

type BrandForm = Pick<BrandConfig, 'system_name' | 'logo_url'>;

const defaultForm: BrandForm = {
  system_name: 'Niko Admin',
  logo_url: '/favicon.ico',
};

export default function BrandConfigPage() {
  const { t } = useTranslation('modules/brand-config');
  const toast = useToast();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const uploadBg = useColorModeValue('gray.50', 'whiteAlpha.50');
  const uploadHoverBg = useColorModeValue('gray.100', 'whiteAlpha.200');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { setBrand } = useBrand();
  const [form, setForm] = useState<BrandForm>(defaultForm);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    loadConfig();
  }, []);

  async function loadConfig() {
    setLoading(true);
    try {
      const config = await brandConfigApi.get();
      setForm({ system_name: config.system_name, logo_url: config.logo_url });
      setBrand(config);
    } catch (err) {
      toast({ title: t('message.loadFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setLoading(false);
    }
  }

  const setField = (key: keyof BrandForm, value: string) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  };

  async function save() {
    setSaving(true);
    try {
      const next = await brandConfigApi.save(form);
      setForm({ system_name: next.system_name, logo_url: next.logo_url });
      setBrand(next);
      toast({ title: t('message.saved'), status: 'success' });
    } catch (err) {
      toast({ title: t('message.saveFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setSaving(false);
    }
  }

  async function uploadLogo(file?: File) {
    if (!file) return;
    setUploading(true);
    try {
      const res = await brandConfigApi.uploadLogo(file);
      setField('logo_url', res.logo_url);
      toast({ title: t('message.logoUploaded'), status: 'success' });
    } catch (err) {
      toast({ title: t('message.logoUploadFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <HStack justify="space-between" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button variant="brand" onClick={save} isLoading={saving || loading}>{t('actions.save')}</Button>
      </HStack>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} p={6}>
        <VStack align="stretch" spacing={6} maxW={{ base: '100%', md: '500px' }}>
          <FormControl isRequired>
            <FormLabel>{t('fields.systemName')}</FormLabel>
            <Input value={form.system_name} onChange={(e) => setField('system_name', e.target.value)} />
          </FormControl>
          
          <FormControl>
            <FormLabel>{t('fields.logo')}</FormLabel>
            <Input
              ref={fileInputRef}
              type="file"
              accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml"
              display="none"
              onChange={(e) => uploadLogo(e.target.files?.[0])}
            />
            <Flex gap={4} align="center">
              <Box
                w="80px"
                h="80px"
                border="1px dashed"
                borderColor={borderColor}
                borderRadius="8px"
                overflow="hidden"
                bg={uploadBg}
                flexShrink={0}
              >
                <Image 
                  src={form.logo_url || '/favicon.ico'} 
                  fallbackSrc="/favicon.ico"
                  w="100%" 
                  h="100%" 
                  objectFit="contain"
                  p={2}
                  opacity={uploading ? 0.5 : 1}
                />
              </Box>
              <VStack align="start" spacing={2} flex={1}>
                <HStack>
                  <Button 
                    size="sm" 
                    leftIcon={<Icon as={MdUpload} />} 
                    onClick={() => !uploading && fileInputRef.current?.click()}
                    isLoading={uploading}
                    variant="outline"
                  >
                    {t('actions.uploadLogo')}
                  </Button>
                  {form.logo_url && form.logo_url !== '/favicon.ico' && (
                    <Button 
                      size="sm" 
                      variant="ghost" 
                      colorScheme="red"
                      leftIcon={<Icon as={MdDelete} />}
                      onClick={() => setField('logo_url', '/favicon.ico')}
                      isDisabled={uploading}
                    >
                      {t('actions.remove')}
                    </Button>
                  )}
                </HStack>
                <Text color="gray.500" fontSize="sm">
                  {t('message.logoHint')}
                </Text>
              </VStack>
            </Flex>
          </FormControl>
        </VStack>
      </Box>
    </Box>
  );
}
