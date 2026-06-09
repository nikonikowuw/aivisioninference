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
      toast({ status: 'success', title: editingTag ? 'Updated' : 'Created' });
      onFormClose();
      fetchTags();
    } catch {
      toast({ status: 'error', title: 'Operation failed' });
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
      toast({ status: 'success', title: 'Deleted' });
      onDeleteClose();
      fetchTags();
    } catch {
      toast({ status: 'error', title: 'Delete failed' });
    }
  };

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text color={textColor} fontSize="22px" fontWeight="700">
          {t('groups.title').replace('分组', '标签').replace('Group', 'Tag')}
        </Text>
        <Button
          leftIcon={<AddIcon />}
          colorScheme="brand"
          onClick={handleCreate}
        >
          {t('groups.create').replace('分组', '标签').replace('Group', 'Tag')}
        </Button>
      </Flex>

      <Box bg={bgCard} borderRadius="16px" border={`1px solid ${borderColor}`} overflow="hidden">
        {loading ? (
          <TableSkeleton columns={5} rows={5} />
        ) : tags.length === 0 ? (
          <EmptyState message="No tags found" />
        ) : (
          <Table variant="simple">
            <Thead>
              <Tr>
                <Th>{t('groups.form.groupName.label').replace('分组名称', '标签名称')}</Th>
                <Th>Color</Th>
                <Th isNumeric>Sort Order</Th>
                <Th isNumeric>Persons</Th>
                <Th>Actions</Th>
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
                        aria-label="Edit"
                        icon={<EditIcon />}
                        size="sm"
                        variant="ghost"
                        onClick={() => handleEdit(tag)}
                      />
                      <IconButton
                        aria-label="Delete"
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
      <Modal isOpen={isFormOpen} onClose={onFormClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {editingTag ? t('groups.edit').replace('分组', '标签') : t('groups.create').replace('分组', '标签')}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('groups.form.groupName.label').replace('分组名称', '标签名称')}</FormLabel>
                <Input
                  value={formData.tag_name}
                  onChange={(e) => setFormData({ ...formData, tag_name: e.target.value })}
                  placeholder="Enter tag name"
                />
              </FormControl>
              <FormControl>
                <FormLabel>Color</FormLabel>
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
                <FormLabel>{t('groups.form.sortOrder.label')}</FormLabel>
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
              Cancel
            </Button>
            <Button
              colorScheme="brand"
              onClick={handleSubmit}
              isLoading={submitting}
              isDisabled={!formData.tag_name.trim()}
            >
              {editingTag ? 'Update' : 'Create'}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmDialog
        isOpen={isDeleteOpen}
        onClose={onDeleteClose}
        onConfirm={confirmDelete}
        title={t('groups.delete').replace('分组', '标签')}
        message={t('groups.deleteConfirm', { name: deleteTarget?.tag_name || '' })}
      />
    </Box>
  );
}
