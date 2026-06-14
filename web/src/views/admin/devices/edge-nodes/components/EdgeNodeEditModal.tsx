import {
  Button,
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
  Stack,
  Switch,
  Textarea,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { edgeNodeApi, type EdgeNode, type UpdateEdgeNodeRequest } from 'services/edgeNode';

interface EdgeNodeEditModalProps {
  isOpen: boolean;
  onClose: () => void;
  node: EdgeNode;
  onSuccess: (updatedNode: EdgeNode) => void;
}

export default function EdgeNodeEditModal({ isOpen, onClose, node, onSuccess }: EdgeNodeEditModalProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const textColorSecondary = 'gray.400';
  const toast = useToast();

  const [form, setForm] = useState<UpdateEdgeNodeRequest>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen && node) {
      setForm({
        name: node.name,
        description: node.description || '',
        endpoint: node.endpoint,
        ipc_addr: node.ipc_addr || '',
        max_load: node.max_load,
        enabled: node.enabled,
        remark: node.remark || '',
      });
    }
  }, [isOpen, node]);

  const handleChange = useCallback((field: keyof UpdateEdgeNodeRequest, value: string | number | boolean | undefined) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!form.name || !form.endpoint) {
      toast({ title: tCommon('message.requiredFields'), status: 'warning' });
      return;
    }
    
    // 乐观更新：先把表单数据和原 node 合并，立即关闭弹窗
    const prevNode = node;
    const optimisticNode: EdgeNode = {
      ...prevNode,
      ...form,
      name: form.name!,
      endpoint: form.endpoint!,
      max_load: form.max_load!,
    };
    onSuccess(optimisticNode);
    onClose();

    // 后台异步提交 API
    try {
      await edgeNodeApi.update(node.id, form);
      toast({ title: t('message.updateSuccess'), status: 'success' });
    } catch {
      // 失败时回滚到更新前的数据
      onSuccess(prevNode);
      toast({ title: t('message.updateFailed'), status: 'error' });
    }
  }, [node, form, toast, t, tCommon, onSuccess, onClose]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="xl" isCentered>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader color={textColor}>{t('actions.editNode')}</ModalHeader>
        <ModalCloseButton />
        <ModalBody pb={6}>
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

              <FormControl>
                <FormLabel fontSize="sm" color={textColorSecondary}>{t('fields.ipcAddr')}</FormLabel>
                <Input
                  value={form.ipc_addr || ''}
                  onChange={(e) => handleChange('ipc_addr', e.target.value)}
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
              <Button variant="ghost" onClick={onClose}>
                {tCommon('button.cancel')}
              </Button>
              <Button
                colorScheme="brand"
                type="button"
                onClick={handleSubmit}
                isLoading={isSubmitting}
              >
                {tCommon('button.submit')}
              </Button>
            </HStack>
          </Stack>
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}
