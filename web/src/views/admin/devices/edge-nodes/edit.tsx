import { ChevronLeftIcon } from '@chakra-ui/icons';
import {
  Box,
  Button,
  Center,
  Flex,
  FormControl,
  FormLabel,
  HStack,
  Input,
  NumberDecrementStepper,
  NumberIncrementStepper,
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  SimpleGrid,
  Spinner,
  Stack,
  Switch,
  Text,
  Textarea,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import { edgeNodeApi, type EdgeNode, type UpdateEdgeNodeRequest } from 'services/edgeNode';

export default function EditEdgeNode() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');

  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const textColorSecondary = 'gray.400';
  const toast = useToast();

  const [form, setForm] = useState<UpdateEdgeNodeRequest>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (!id) return;
    setIsLoading(true);
    edgeNodeApi.get(id)
      .then((node) => {
        setForm({
          name: node.name,
          description: node.description,
          endpoint: node.endpoint,
          max_load: node.max_load,
          enabled: node.enabled,
          remark: node.remark,
        });
      })
      .catch(() => toast({ title: t('message.loadFailed'), status: 'error' }))
      .finally(() => setIsLoading(false));
  }, [id, toast, t]);

  const handleChange = useCallback((field: keyof UpdateEdgeNodeRequest, value: string | number | boolean | undefined) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!id || !form.name || !form.endpoint) {
      toast({ title: tCommon('message.requiredFields'), status: 'warning' });
      return;
    }
    setIsSubmitting(true);
    try {
      await edgeNodeApi.update(id, form);
      toast({ title: t('message.updateSuccess'), status: 'success' });
      navigate(`/admin/devices/edge-nodes/${id}`);
    } catch {
      toast({ title: t('message.updateFailed'), status: 'error' });
    } finally {
      setIsSubmitting(false);
    }
  }, [id, form, toast, t, tCommon, navigate]);

  if (isLoading) {
    return (
      <Center h="50vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" mb="20px" maxW="700px">
        <HStack spacing={4} mb="20px">
          <Button
            leftIcon={<ChevronLeftIcon />}
            variant="ghost"
            onClick={() => navigate(`/admin/devices/edge-nodes/${id}`)}
          >
            {tCommon('button.back')}
          </Button>
          <Text color={textColor} fontSize="2xl" fontWeight="bold">
            {t('actions.edit')}
          </Text>
        </HStack>

        <Card px="24px" py="24px">
          <Stack spacing={4}>
            <SimpleGrid columns={{ base: 1, md: 2 }} spacing="20px">
              <FormControl isRequired>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.name')}</FormLabel>
                <Input
                  value={form.name || ''}
                  onChange={(e) => handleChange('name', e.target.value)}
                />
              </FormControl>

              <FormControl isRequired>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.endpoint')}</FormLabel>
                <Input
                  value={form.endpoint || ''}
                  onChange={(e) => handleChange('endpoint', e.target.value)}
                  type="url"
                />
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.description')}</FormLabel>
                <Input
                  value={form.description || ''}
                  onChange={(e) => handleChange('description', e.target.value)}
                />
              </FormControl>


              <FormControl isRequired>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.maxLoad')}</FormLabel>
                <NumberInput
                  min={1}
                  max={100}
                  value={form.max_load ?? 4}
                  onChange={(_, v) => handleChange('max_load', v)}
                >
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>

              <FormControl display="flex" alignItems="center" pt="6px">
                <FormLabel fontSize="sm" color={textColorSecondary} mb={0}>
                  {t('fields.enabled')}
                </FormLabel>
                <Switch
                  isChecked={form.enabled !== false}
                  onChange={(e) => handleChange('enabled', e.target.checked)}
                />
              </FormControl>
            </SimpleGrid>

            <FormControl>
              <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.remark')}</FormLabel>
              <Textarea
                value={form.remark || ''}
                onChange={(e) => handleChange('remark', e.target.value)}
                rows={3}
              />
            </FormControl>

            <HStack justifyContent="flex-end" pt={4}>
              <Button
                variant="ghost"
                onClick={() => navigate(`/admin/devices/edge-nodes/${id}`)}
              >
                {tCommon('button.cancel')}
              </Button>
              <Button
                colorScheme="brand"
                onClick={handleSubmit}
                isLoading={isSubmitting}
              >
                {tCommon('button.submit')}
              </Button>
            </HStack>
          </Stack>
        </Card>
      </Flex>
    </Box>
  );
}
