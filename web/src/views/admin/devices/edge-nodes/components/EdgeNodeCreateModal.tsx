import { CopyIcon } from '@chakra-ui/icons';
import {
  Button,
  Code,
  Flex,
  FormControl,
  FormLabel,
  HStack,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  NumberDecrementStepper,
  NumberIncrementStepper,
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  SimpleGrid,
  Text,
  Textarea,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { edgeNodeApi, type CreateEdgeNodeRequest, type CreateNodeResponse } from 'services/edgeNode';

interface EdgeNodeCreateModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export default function EdgeNodeCreateModal({ isOpen, onClose, onSuccess }: EdgeNodeCreateModalProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const textColorSecondary = 'gray.400';
  const toast = useToast();

  const [form, setForm] = useState<CreateEdgeNodeRequest>({
    name: '',
    description: '',
    endpoint: '',
    max_load: 4,
    remark: '',
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [result, setResult] = useState<CreateNodeResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const handleChange = useCallback((field: keyof CreateEdgeNodeRequest, value: string | number) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!form.name || !form.endpoint) {
      toast({ title: tCommon('message.requiredFields'), status: 'warning' });
      return;
    }
    setIsSubmitting(true);
    try {
      const res = await edgeNodeApi.create(form);
      setResult(res);
      onSuccess();
      toast({ title: t('message.createSuccess'), status: 'success' });
    } catch {
      toast({ title: t('message.createFailed'), status: 'error' });
    } finally {
      setIsSubmitting(false);
    }
  }, [form, toast, t, tCommon, onSuccess]);

  const handleCopyToken = useCallback(async () => {
    if (!result?.token) return;
    try {
      await navigator.clipboard.writeText(result.token);
      setCopied(true);
      toast({ title: t('message.copySuccess'), status: 'success' });
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast({ title: tCommon('message.copyFailed'), status: 'error' });
    }
  }, [result, toast, t, tCommon]);

  const handleClose = useCallback(() => {
    setForm({ name: '', description: '', endpoint: '', max_load: 4, remark: '' });
    setResult(null);
    setCopied(false);
    onClose();
  }, [onClose]);

  const handleCloseResult = useCallback(() => {
    handleClose();
  }, [handleClose]);

  return (
    <Modal isOpen={isOpen} onClose={handleClose} size={result ? 'lg' : 'xl'} isCentered>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader color={textColor}>
          {result ? t('message.createSuccess') : t('actions.create')}
        </ModalHeader>
        <ModalCloseButton />

        {result ? (
          <ModalBody pb={6}>
            <Text color={textColor} fontSize="md" fontWeight="bold" mb={3}>
              {t('fields.authToken')}
            </Text>
            <Flex gap={2} mb={4}>
              <Code
                p={3}
                borderRadius="md"
                fontSize="sm"
                wordBreak="break-all"
                whiteSpace="pre-wrap"
                flex={1}
                bg="gray.50"
                _dark={{ bg: 'gray.900' }}
              >
                {result.token}
              </Code>
              <Button
                leftIcon={<CopyIcon />}
                onClick={handleCopyToken}
                colorScheme={copied ? 'green' : 'blue'}
                size="sm"
                flexShrink={0}
              >
                {copied ? t('actions.copied') : t('actions.copyToken')}
              </Button>
            </Flex>

            <Text color="orange.500" fontSize="sm" fontWeight="medium" mb={4}>
              ⚠ {t('message.tokenNotice')}
            </Text>

            <Text color={textColor} fontSize="md" fontWeight="bold" mb={3}>
              {t('message.configGuide')}
            </Text>
            <Code
              p={4}
              borderRadius="md"
              whiteSpace="pre-wrap"
              wordBreak="break-all"
              fontSize="sm"
              display="block"
              bg="gray.50"
              _dark={{ bg: 'gray.900' }}
            >
{`# .env 引擎配置文件
NIKO_ENGINE_NODE_ID=${result.node.id}
NIKO_ENGINE_AUTH_TOKEN=${result.token}
NIKO_ENGINE_HTTP_PORT=8080
NIKO_ENGINE_PLATFORM_URL=http://your-platform-server:8080`}
            </Code>

            <Button mt={4} w="full" colorScheme="brand" onClick={handleCloseResult}>
              {tCommon('button.close')}
            </Button>
          </ModalBody>
        ) : (
          <ModalBody pb={6}>
            <SimpleGrid columns={{ base: 1, md: 2 }} spacing="20px" mb={4}>
              <FormControl isRequired>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.name')}</FormLabel>
                <Input
                  value={form.name}
                  onChange={(e) => handleChange('name', e.target.value)}
                  placeholder="node-001"
                />
              </FormControl>

              <FormControl isRequired>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.endpoint')}</FormLabel>
                <Input
                  value={form.endpoint}
                  onChange={(e) => handleChange('endpoint', e.target.value)}
                  placeholder="http://192.168.1.100:8080"
                  type="url"
                />
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.description')}</FormLabel>
                <Input
                  value={form.description || ''}
                  onChange={(e) => handleChange('description', e.target.value)}
                  placeholder={t('fields.description')}
                />
              </FormControl>


              <FormControl isRequired>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.maxLoad')}</FormLabel>
                <NumberInput
                  min={1}
                  max={100}
                  value={form.max_load}
                  onChange={(_, v) => handleChange('max_load', v)}
                >
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>
            </SimpleGrid>

            <FormControl mb={4}>
              <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.remark')}</FormLabel>
              <Textarea
                value={form.remark || ''}
                onChange={(e) => handleChange('remark', e.target.value)}
                rows={3}
              />
            </FormControl>

            <HStack justifyContent="flex-end" pt={4}>
              <Button variant="ghost" onClick={handleClose}>
                {tCommon('button.cancel')}
              </Button>
              <Button colorScheme="brand" onClick={handleSubmit} isLoading={isSubmitting}>
                {tCommon('button.submit')}
              </Button>
            </HStack>
          </ModalBody>
        )}
      </ModalContent>
    </Modal>
  );
}
