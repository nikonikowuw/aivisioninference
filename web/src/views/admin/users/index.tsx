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
  FormHelperText,
  Input,
  Select,
  useDisclosure,
  HStack,
  Badge,
  Spinner,
  Center,
  Switch,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { usersApi, rolesApi, type User, type Role } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import AvatarUploader from 'components/avatar-upload/AvatarUploader';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useAuth } from 'contexts/AuthContext';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { parseOptionalNumber } from 'utils/convert';

export default function Users() {
  const { user: currentUser } = useAuth();
  const { t } = useTranslation('modules/users');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();

  const { filters, setFilter, resetFilters, searchTrigger } = useFilter();

  const fetchUsers = useCallback((p: number, ps: number) => {
    return usersApi.list({
      page: p,
      page_size: ps,
      keyword: filters.keyword,
      status: parseOptionalNumber(filters.status),
    });
  }, [filters]);

  const { list: users, total, page, pageSize, initialLoading, pageLoading, load: loadUsers, changePage, changePageSize } = usePagination<User>(fetchUsers);

  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState({ username: '', display_name: '', email: '', password: '', status: 1 });
  const [avatarUrl, setAvatarUrl] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [toggleTarget, setToggleTarget] = useState<User | null>(null);
  const [isToggling, setIsToggling] = useState(false);

  useEffect(() => {
    loadUsers({ page: 1 }).catch(() => {
      toast({ title: tCommon('message.loadFailed'), status: 'error' });
    });
  }, [searchTrigger, loadUsers]);

  useEffect(() => {
    rolesApi.list({ page: 1, page_size: 100 }).then((d) => setAllRoles(d.list)).catch(() => {
      toast({ title: tCommon('message.loadFailed'), status: 'error' });
    });
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({ username: '', display_name: '', email: '', password: '', status: 1 });
    setAvatarUrl('');
    onOpen();
  };

  const openEdit = (user: User) => {
    setEditing(user);
    setForm({ username: user.username, display_name: user.display_name, email: user.email, password: '', status: user.status });
    setAvatarUrl(user.avatar_url || '');
    onOpen();
  };

  const handleSave = async () => {
    try {
      if (editing) {
        const updateData: Partial<User> & { password?: string } = {
          username: form.username,
          display_name: form.display_name,
          email: form.email,
          password: form.password,
        };
        if (!form.password) delete updateData.password;
        await usersApi.update(editing.id, updateData);
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

  const handleToggleStatus = async (user: User) => {
    const newStatus = user.status === 1 ? 0 : 1;
    setIsToggling(true);
    try {
      await usersApi.update(user.id, { status: newStatus });
      toast({ title: t('message.updateSuccess'), status: 'success' });
      loadUsers();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsToggling(false);
      setToggleTarget(null);
    }
  };

  const handleToggleClick = (user: User) => {
    if (isToggling || toggleTarget) return;
    setToggleTarget(user);
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>{t('button.create')}</Button>
      </Flex>
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        selects={[
          {
            name: 'status',
            label: t('table.columns.status'),
            options: [
              { value: '1', label: t('table.status.active') },
              { value: '0', label: t('table.status.inactive') },
            ],
          },
        ]}
      />
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="700px">
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
                  <HStack spacing={2}>
                    <Switch
                      aria-label={u.status === 1 ? t('actions.disable') : t('actions.enable')}
                      isChecked={u.status === 1}
                      isDisabled={u.id === currentUser?.id || isToggling || toggleTarget?.id === u.id}
                      onChange={() => handleToggleClick(u)}
                      colorScheme="green"
                    />
                    <Badge colorScheme={u.status === 1 ? 'green' : 'red'}>{u.status === 1 ? t('table.status.active') : t('table.status.inactive')}</Badge>
                  </HStack>
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
        <Pagination
          page={page}
          pageSize={pageSize}
          total={total}
          onChange={changePage}
          onPageSizeChange={changePageSize}
          isLoading={pageLoading}
        />
      </Box>
      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title={t('actions.delete')}
        message={t('message.deleteConfirm')}
        isLoading={isDeleting}
      />
      <ConfirmDialog
        isOpen={toggleTarget !== null}
        onClose={() => setToggleTarget(null)}
        onConfirm={() => toggleTarget && handleToggleStatus(toggleTarget)}
        title={toggleTarget?.status === 1 ? t('actions.disable') : t('actions.enable')}
        message={toggleTarget?.status === 1 ? t('message.disableConfirm') : t('message.enableConfirm')}
        isLoading={isToggling}
      />

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editing ? t('modal.editTitle') : t('modal.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Box textAlign="center" mb={4}>
              <AvatarUploader
                value={avatarUrl}
                onChange={(url) => setAvatarUrl(url)}
                userId={editing?.id}
                name={form.display_name || form.username}
                size={80}
              />
            </Box>
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
            {!editing ? (
              <FormControl mb={4}>
                <FormLabel>{t('form.status.label')}</FormLabel>
                <Select value={form.status} onChange={(e) => setForm({ ...form, status: Number(e.target.value) })}>
                  <option value={1}>{t('form.status.active')}</option>
                  <option value={0}>{t('form.status.inactive')}</option>
                </Select>
              </FormControl>
            ) : (
              <FormControl mb={4} isReadOnly>
                <FormLabel>{t('form.status.label')}</FormLabel>
                <Badge colorScheme={form.status === 1 ? 'green' : 'red'}>{form.status === 1 ? t('form.status.active') : t('form.status.inactive')}</Badge>
                <FormHelperText>{t('form.status.readOnlyHint')}</FormHelperText>
              </FormControl>
            )}
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
