import {
  Box,
  Button,
  Flex,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  useColorModeValue,
  IconButton,
  HStack,
  Center,
  Spinner,
  useToast,
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
  useDisclosure,
  Icon,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { deviceGroupsApi, type DeviceGroup } from 'services/api';
import Card from 'components/card/Card';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { usePagination } from 'hooks/usePagination';
import { MdFolder } from 'react-icons/md';

export default function DeviceGroups() {
  const { t } = useTranslation('modules/devices');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const modalBg = useColorModeValue('white', 'navy.700');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();

  const fetchGroups = useCallback((p: number, ps: number) => deviceGroupsApi.list({ page: p, page_size: ps }), []);

  const {
    list: groups,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
  } = usePagination<DeviceGroup>(fetchGroups);

  const [editing, setEditing] = useState<DeviceGroup | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeviceGroup | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    load({ page: 1 });
  }, [load]);

  const [form, setForm] = useState({
    group_name: '',
    description: '',
  });

  const resetForm = (group?: DeviceGroup) => {
    setEditing(group || null);
    setForm({
      group_name: group?.group_name || '',
      description: group?.description || '',
    });
    onOpen();
  };

  const openCreate = () => resetForm();
  const openEdit = (group: DeviceGroup) => resetForm(group);

  const handleSave = async () => {
    try {
      if (editing) {
        await deviceGroupsApi.update(editing.id, form);
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await deviceGroupsApi.create(form as any);
        toast({ title: t('message.createSuccess'), status: 'success' });
      }
      onClose();
      load();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await deviceGroupsApi.delete(deleteTarget.id);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      load();
    } catch (err) {
      toast({ title: t('message.deleteFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('groupFields.groupName')}</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>
          {t('groupActions.create')}
        </Button>
      </Flex>

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th>{t('groupFields.groupName')}</Th>
                <Th>{t('groupFields.description')}</Th>
                <Th>{t('groupFields.deviceCount')}</Th>
                <Th textAlign="right">{t('groupFields.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {pageLoading ? (
                <Tr><Td colSpan={4}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : groups.length === 0 ? (
                <Tr><Td colSpan={4}><Center py="20px">{tCommon('noData')}</Center></Td></Tr>
              ) : (
                groups.map((group) => (
                  <Tr key={group.id}>
                    <Td>
                      <HStack>
                        <Icon as={MdFolder} color="brand.500" />
                        <Text color={textColor} fontSize="sm" fontWeight="700">{group.group_name}</Text>
                      </HStack>
                    </Td>
                    <Td><Text fontSize="sm">{group.description || '-'}</Text></Td>
                    <Td><Text fontSize="sm">{group.device_count}</Text></Td>
                    <Td textAlign="right">
                      <HStack justify="flex-end">
                        <IconButton aria-label="edit" icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(group)} />
                        <IconButton aria-label="delete" icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(group)} />
                      </HStack>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>

        <Box px="25px">
          <Pagination
            page={page}
            pageSize={pageSize}
            total={total}
            onChange={changePage}
            onPageSizeChange={changePageSize}
          />
        </Box>
      </Card>

      {/* Create/Edit Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent bg={modalBg}>
          <ModalHeader color={textColor}>{editing ? t('actions.edit') : t('groupActions.create')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb="4" isRequired>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('groupFields.groupName')}</FormLabel>
              <Input variant="main" value={form.group_name} onChange={(e) => setForm({ ...form, group_name: e.target.value })} maxLength={128} />
            </FormControl>
            <FormControl mb="4">
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>{t('groupFields.description')}</FormLabel>
              <Input variant="main" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr="3" onClick={onClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleSave}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation */}
      <ConfirmDialog
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title={t('actions.delete')}
        message={t('message.deleteGroupConfirm') + (deleteTarget?.device_count ? ` (${deleteTarget.device_count} ${t('groupFields.deviceCount')})` : '')}
        isLoading={isDeleting}
      />
    </Box>
  );
}
