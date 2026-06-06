import {
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Spinner,
  Switch,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from 'services/api';

interface Webhook {
  id: string;
  name: string;
  url: string;
  enabled: boolean;
  tags: string[];
  max_retries: number;
  created_at: string;
}

export default function WebhookConfigTab() {
  const { t } = useTranslation(['modules/system', 'common']);
  const toast = useToast();
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const { isOpen, onOpen, onClose } = useDisclosure();

  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({
    name: '',
    url: '',
    enabled: true,
    tags: '',
    max_retries: 3,
  });

  useEffect(() => {
    loadWebhooks();
  }, []);

  async function loadWebhooks() {
    setLoading(true);
    try {
      const data = await request<Webhook[]>('/system/webhook');
      setWebhooks(data || []);
    } catch (err) {
      console.error('Failed to load webhooks:', err);
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate() {
    try {
      await request('/system/webhook', {
        method: 'POST',
        body: JSON.stringify({
          ...form,
          tags: form.tags.split(',').map(s => s.trim()).filter(Boolean),
        }),
      });
      toast({ title: t('webhook.created'), status: 'success', duration: 3000 });
      onClose();
      setForm({ name: '', url: '', enabled: true, tags: '', max_retries: 3 });
      loadWebhooks();
    } catch (err: any) {
      toast({ title: err.message || t('webhook.create_failed'), status: 'error', duration: 3000 });
    }
  }

  async function handleDelete(id: string) {
    try {
      await request(`/system/webhook/${id}`, { method: 'DELETE' });
      toast({ title: t('webhook.deleted'), status: 'success', duration: 3000 });
      loadWebhooks();
    } catch (err: any) {
      toast({ title: err.message || t('webhook.delete_failed'), status: 'error', duration: 3000 });
    }
  }

  async function handleTest(id: string) {
    try {
      const data = await request<{ success: boolean }>(`/system/webhook/${id}/test`, { method: 'POST' });
      toast({
        title: data?.success ? t('webhook.test_success') : t('webhook.test_failed'),
        status: data?.success ? 'success' : 'error',
        duration: 3000,
      });
    } catch (err: any) {
      toast({ title: err.message || t('webhook.test_failed'), status: 'error', duration: 3000 });
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
      <Flex justify="space-between" align="center" mb={4}>
        <Text fontWeight="600">{t('webhook.list')}</Text>
        <Button colorScheme="brand" onClick={onOpen}>{t('webhook.create')}</Button>
      </Flex>

      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>{t('webhook.name')}</Th>
              <Th>URL</Th>
              <Th>{t('webhook.tags')}</Th>
              <Th>{t('webhook.enabled')}</Th>
              <Th>{t('common:actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {webhooks?.map((webhook) => (
              <Tr key={webhook.id}>
                <Td>{webhook.name}</Td>
                <Td maxW="300px" isTruncated>{webhook.url}</Td>
                <Td>{webhook.tags?.join(', ')}</Td>
                <Td>
                  <Switch isChecked={webhook.enabled} isReadOnly />
                </Td>
                <Td>
                  <Flex gap={2}>
                    <Button size="sm" onClick={() => handleTest(webhook.id)}>
                      {t('webhook.test')}
                    </Button>
                    <Button size="sm" colorScheme="red" variant="ghost" onClick={() => handleDelete(webhook.id)}>
                      {t('common:delete')}
                    </Button>
                  </Flex>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('webhook.create')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Flex direction="column" gap={4}>
              <FormControl>
                <FormLabel>{t('webhook.name')}</FormLabel>
                <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
              </FormControl>
              <FormControl>
                <FormLabel>URL</FormLabel>
                <Input value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://example.com/webhook" />
              </FormControl>
              <FormControl>
                <FormLabel>{t('webhook.tags')}</FormLabel>
                <Input value={form.tags} onChange={(e) => setForm({ ...form, tags: e.target.value })} placeholder="alarm, recognition" />
              </FormControl>
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>{t('webhook.enabled')}</FormLabel>
                <Switch isChecked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
              </FormControl>
            </Flex>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>{t('common:cancel')}</Button>
            <Button colorScheme="brand" onClick={handleCreate}>{t('common:save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
