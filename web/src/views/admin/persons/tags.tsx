import {
  AddIcon,
  DeleteIcon,
  EditIcon,
} from '@chakra-ui/icons';
import {
  Box,
  Button,
  Flex,
  HStack,
  IconButton,
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
  Badge,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  VStack,
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import { EmptyState } from 'components/empty/EmptyState';
import { TableSkeleton } from 'components/skeleton/Skeleton';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { personTagsApi, type PersonTag } from 'services/api';

interface TagFormData {
  tag_name: string;
  color: string;
  sort_order: number;
}

const defaultFormData: TagFormData = {
  tag_name: '',
  color: '#1890ff',
  sort_order: 0,
};

export default function PersonTagsPage() {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const [tags, setTags] = useState<PersonTag[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingTag, setEditingTag] = useState<PersonTag | null>(null);
  const [formData, setFormData] = useState<TagFormData>(defaultFormData);
  const [deleteTarget, setDeleteTarget] = useState<PersonTag | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const { isOpen: isFormOpen, onOpen: onFormOpen, onClose: onFormClose } = useDisclosure();
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure();

  const fetchTags = useCallback(async () => {
    setLoading(true);
    try {
      const data = await personTagsApi.list();
      setTags(data);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    } finally {
      setLoading(false);
    }
  }, [toast, t]);

  useEffect(() => {
    fetchTags();
  }, [fetchTags]);

  const handleCreate = () => {
    setEditingTag(null);
    setFormData(defaultFormData);
    onFormOpen();
  };

  const handleEdit = (tag: PersonTag) => {
    setEditingTag(tag);
    setFormData({
      tag_name: tag.tag_name,
      color: tag.color,
      sort_order: tag.sort_order,
    });
    onFormOpen();
  };

  const handleSubmit = async () => {
    if (!formData.tag_name.trim()) return;
    setSubmitting(true);
    try {
      if (editingTag) {
        await personTagsApi.update(editingTag.id, formData);
      } else {
        await personTagsApi.create(formData);
      }
      toast({ status: 'success', title: tCommon('message.success') });
      onFormClose();
      fetchTags();
    } catch {
      toast({ status: 'error', title: tCommon('message.operationFailed') });
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = (tag: PersonTag) => {
    setDeleteTarget(tag);
    onDeleteOpen();
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    try {
      await personTagsApi.delete(deleteTarget.id);
      toast({ status: 'success', title: tCommon('message.success') });
      onDeleteClose();
      fetchTags();
    } catch {
      toast({ status: 'error', title: tCommon('message.operationFailed') });
    }
  };

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text color={textColor} fontSize="22px" fontWeight="700">
          {t('tags.title')}
        </Text>
        <Button
          leftIcon={<AddIcon />}
          colorScheme="brand"
          onClick={handleCreate}
        >
          {t('tags.create')}
        </Button>
      </Flex>

      <Box bg={bgCard} borderRadius="16px" border={`1px solid ${borderColor}`} overflow="hidden">
        {loading ? (
          <TableSkeleton columns={5} rows={5} />
        ) : tags.length === 0 ? (
          <EmptyState description={t('tags.empty')} />
        ) : (
          <Table variant="simple">
            <Thead>
              <Tr>
                <Th>{t('tags.table.columns.tagName')}</Th>
                <Th>{t('tags.table.columns.color')}</Th>
                <Th isNumeric>{t('tags.table.columns.sortOrder')}</Th>
                <Th isNumeric>{t('tags.table.columns.persons')}</Th>
                <Th>{tCommon('actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {tags.map((tag) => (
                <Tr key={tag.id}>
                  <Td>
                    <HStack>
                      <Badge bg={tag.color} color="white" px={2} py={1} borderRadius="4px">
                        {tag.tag_name}
                      </Badge>
                    </HStack>
                  </Td>
                  <Td>
                    <HStack>
                      <Box w="24px" h="24px" borderRadius="4px" bg={tag.color} />
                      <Text fontSize="sm">{tag.color}</Text>
                    </HStack>
                  </Td>
                  <Td isNumeric>{tag.sort_order}</Td>
                  <Td isNumeric>{tag.person_count}</Td>
                  <Td>
                    <HStack spacing={2}>
                      <IconButton
                        aria-label={t('actions.edit')}
                        icon={<EditIcon />}
                        size="sm"
                        variant="ghost"
                        onClick={() => handleEdit(tag)}
                      />
                      <IconButton
                        aria-label={t('actions.delete')}
                        icon={<DeleteIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => handleDelete(tag)}
                      />
                    </HStack>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Box>

      {/* Create/Edit Modal */}
      <Modal isOpen={isFormOpen} onClose={onFormClose} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {editingTag ? t('tags.edit') : t('tags.create')}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('tags.form.tagName.label')}</FormLabel>
                <Input
                  value={formData.tag_name}
                  onChange={(e) => setFormData({ ...formData, tag_name: e.target.value })}
                  placeholder={t('tags.form.tagName.placeholder')}
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('tags.form.color.label')}</FormLabel>
                <HStack>
                  <Input
                    type="color"
                    w="60px"
                    h="40px"
                    p={1}
                    value={formData.color}
                    onChange={(e) => setFormData({ ...formData, color: e.target.value })}
                  />
                  <Input
                    value={formData.color}
                    onChange={(e) => setFormData({ ...formData, color: e.target.value })}
                    placeholder="#1890ff"
                  />
                </HStack>
              </FormControl>
              <FormControl>
                <FormLabel>{t('tags.form.sortOrder.label')}</FormLabel>
                <Input
                  type="number"
                  value={formData.sort_order}
                  onChange={(e) => setFormData({ ...formData, sort_order: parseInt(e.target.value) || 0 })}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onFormClose}>
              {tCommon('button.cancel')}
            </Button>
            <Button
              colorScheme="brand"
              onClick={handleSubmit}
              isLoading={submitting}
              isDisabled={!formData.tag_name.trim()}
            >
              {editingTag ? tCommon('button.save') : tCommon('button.create')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmDialog
        isOpen={isDeleteOpen}
        onClose={onDeleteClose}
        onConfirm={confirmDelete}
        title={t('tags.delete')}
        message={t('tags.deleteConfirm', { name: deleteTarget?.tag_name || '' })}
      />
    </Box>
  );
}
