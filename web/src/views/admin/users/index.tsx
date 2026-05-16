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
  Select,
  useDisclosure,
  HStack,
  Badge,
  Spinner,
  Center,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { usersApi, rolesApi, type User, type Role } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';

export default function Users() {
  const { t } = useTranslation('modules/users');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [users, setUsers] = useState<User[]>([]);
  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState({ username: '', display_name: '', email: '', password: '', status: 1 });
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const loadUsers = useCallback(async () => {
    try {
      const data = await usersApi.list({ page: 1, page_size: 100 });
      setUsers(data.list);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    }
  }, [toast, t]);

  useEffect(() => {
    Promise.all([loadUsers(), rolesApi.list({ page: 1, page_size: 100 }).then((d) => setAllRoles(d.list))])
      .finally(() => setLoading(false));
  }, [loadUsers]);

  const openCreate = () => {
    setEditing(null);
    setForm({ username: '', display_name: '', email: '', password: '', status: 1 });
    onOpen();
  };

  const openEdit = (user: User) => {
    setEditing(user);
    setForm({ username: user.username, display_name: user.display_name, email: user.email, password: '', status: user.status });
    onOpen();
  };

  const handleSave = async () => {
    try {
      if (editing) {
        const updateData: Record<string, unknown> = { ...form };
        if (!form.password) delete updateData.password;
        await usersApi.update(editing.id, updateData as Partial<User>);
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await usersApi.create(form as Partial<User>);
        toast({ title: t('message.createSuccess'), status: 'success' });
      }
      onClose();
      loadUsers();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await usersApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      loadUsers();
    } catch (err) {
      toast({ title: t('message.deleteFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>{t('button.create')}</Button>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>{t('table.columns.id')}</Th>
              <Th>{t('table.columns.username')}</Th>
              <Th>{t('table.columns.displayName')}</Th>
              <Th>{t('table.columns.email')}</Th>
              <Th>{t('table.columns.status')}</Th>
              <Th>{t('table.columns.roles')}</Th>
              <Th>{t('table.columns.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {users.map((u) => (
              <Tr key={u.id}>
                <Td>{u.id}</Td>
                <Td fontWeight="600">{u.username}</Td>
                <Td>{u.display_name}</Td>
                <Td>{u.email}</Td>
                <Td>
                  <Badge colorScheme={u.status === 1 ? 'green' : 'red'}>{u.status === 1 ? t('table.status.active') : t('table.status.inactive')}</Badge>
                </Td>
                <Td>{u.roles?.map((r) => r.name).join(', ') || '-'}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label={t('actions.edit')} icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(u)} />
                    <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(u.id)} />
                  </HStack>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title={t('actions.delete')}
        message={t('message.deleteConfirm')}
        isLoading={isDeleting}
      />

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editing ? t('modal.editTitle') : t('modal.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb={4}>
              <FormLabel>{t('form.username.label')}</FormLabel>
              <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} placeholder={t('form.username.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.displayName.label')}</FormLabel>
              <Input value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} placeholder={t('form.displayName.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.email.label')}</FormLabel>
              <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} placeholder={t('form.email.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.password.label')}{editing && `（${t('form.password.hint')}）`}</FormLabel>
              <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder={t('form.password.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.status.label')}</FormLabel>
              <Select value={form.status} onChange={(e) => setForm({ ...form, status: Number(e.target.value) })}>
                <option value={1}>{t('form.status.active')}</option>
                <option value={0}>{t('form.status.inactive')}</option>
              </Select>
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleSave}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
