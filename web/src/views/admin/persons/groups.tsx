import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
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
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import { EmptyState } from 'components/empty/EmptyState';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { personGroupsApi, type PersonGroup } from 'services/api';
import { GroupFormModal } from './components/GroupFormModal';

export default function PersonGroupsPage() {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');
  
  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();

  const [groups, setGroups] = useState<PersonGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<PersonGroup | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [editingGroup, setEditingGroup] = useState<PersonGroup | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await personGroupsApi.list();
      setGroups(res);
    } catch (error: any) {
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
      });
    } finally {
      setLoading(false);
    }
  }, [toast, tCommon]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const openCreate = () => {
    setEditingGroup(null);
    onOpen();
  };

  const openEdit = (group: PersonGroup) => {
    setEditingGroup(group);
    onOpen();
  };

  const handleSave = async (data: any) => {
    try {
      if (editingGroup) {
        await personGroupsApi.update(editingGroup.id, data);
      } else {
        await personGroupsApi.create(data);
      }
      toast({
        title: tCommon('message.success'),
        status: 'success',
      });
      onClose();
      fetchData();
    } catch (error: any) {
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
      });
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await personGroupsApi.delete(deleteTarget.id);
      toast({
        title: tCommon('message.success'),
        status: 'success',
      });
      setDeleteTarget(null);
      fetchData();
    } catch (error: any) {
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
      });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" mb="20px">
        <Flex
          direction={{ base: 'column', md: 'row' }}
          justify="space-between"
          align={{ base: 'start', md: 'center' }}
          mb="20px"
        >
          <Text color={textColor} fontSize="2xl" ms="24px" fontWeight="bold">
            {t('groups.title')}
          </Text>
          <HStack spacing="12px" mt={{ base: '10px', md: '0' }}>
            <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>
              {t('groups.create')}
            </Button>
          </HStack>
        </Flex>
      </Flex>

      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md">
          <Thead>
            <Tr>
              <Th>{t('groups.form.groupName.label')}</Th>
              <Th>{t('groups.form.description.label')}</Th>
              <Th>{t('table.columns.personCount')}</Th>
              <Th>{t('groups.form.sortOrder.label')}</Th>
              <Th isNumeric>{tCommon('actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {loading ? (
              <Tr>
                <Td colSpan={5} textAlign="center" py="40px">
                  <Text color="gray.500">{tCommon('status.loading')}</Text>
                </Td>
              </Tr>
            ) : groups.length === 0 ? (
              <Tr>
                <Td colSpan={5}>
                  <EmptyState />
                </Td>
              </Tr>
            ) : (
              groups.map((g) => (
                <Tr key={g.id}>
                  <Td fontWeight="500">{g.group_name}</Td>
                  <Td>{g.description || '-'}</Td>
                  <Td>{g.person_count ?? 0}</Td>
                  <Td>{g.sort_order}</Td>
                  <Td isNumeric>
                    <HStack justify="end" spacing="4px">
                      <IconButton
                        aria-label={t('actions.edit')}
                        icon={<EditIcon />}
                        size="sm"
                        variant="ghost"
                        onClick={() => openEdit(g)}
                      />
                      <IconButton
                        aria-label={t('actions.delete')}
                        icon={<DeleteIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => setDeleteTarget(g)}
                      />
                    </HStack>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Box>

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        isLoading={isDeleting}
        title={tCommon('dialog.delete.title')}
        message={t('groups.deleteConfirm', { name: deleteTarget?.group_name })}
      />

      <GroupFormModal
        isOpen={isOpen}
        onClose={onClose}
        onSave={handleSave}
        editingGroup={editingGroup}
      />
    </Box>
  );
}
