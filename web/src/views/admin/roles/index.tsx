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
  FormHelperText,
  FormLabel,
  Input,
  Textarea,
  useDisclosure,
  HStack,
  Spinner,
  Center,
  Badge,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon, SettingsIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { rolesApi, permissionsApi, type Role, type Permission } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import PermissionTree from 'components/permission-tree/PermissionTree';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { parseOptionalNumber } from 'utils/convert';



export default function Roles() {
  const { t } = useTranslation('modules/roles');
  const { t: tCommon } = useTranslation('common');
  const { t: tMenu } = useTranslation('menu');

  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const { isOpen: isPermOpen, onOpen: onPermOpen, onClose: onPermClose } = useDisclosure();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchRoles = useCallback((p: number, ps: number) => {
    return rolesApi.list({
      page: p,
      page_size: ps,
      keyword: filters.keyword,
      status: parseOptionalNumber(filters.status),
    });
  }, [filters]);

  const { list: roles, total, page, pageSize, initialLoading, pageLoading, load: loadRoles, changePage, changePageSize } = usePagination<Role>(fetchRoles);

  const [editing, setEditing] = useState<Role | null>(null);
  const [form, setForm] = useState({ name: '', description: '', level: 100 });
  const [isSaving, setIsSaving] = useState(false);
  const roleLevelMin = 1;
  const roleLevelMax = 99999;
  const [permTree, setPermTree] = useState<Permission[]>([]);
  const [selectedPerms, setSelectedPerms] = useState<string[]>([]);
  const [permRoleId, setPermRoleId] = useState<string>('');
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isAssigningPerms, setIsAssigningPerms] = useState(false);

  useEffect(() => {
    loadRoles({ page: 1 });
  }, [searchTrigger, loadRoles]);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', description: '', level: 100 });
    onOpen();
  };

  const openEdit = (role: Role) => {
    setEditing(role);
    setForm({ name: role.name, description: role.description, level: role.level ?? 100 });
    onOpen();
  };

  const handleSave = async () => {
    if (!Number.isInteger(form.level) || form.level < roleLevelMin || form.level > roleLevelMax) {
      toast({
        title: t('message.levelInvalidTitle'),
        description: t('message.levelInvalidDescription', { min: roleLevelMin, max: roleLevelMax }),
        status: 'error',
      });
      return;
    }

    setIsSaving(true);
    try {
      if (editing) {
        await rolesApi.update(editing.id, form);
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await rolesApi.create(form);
        toast({ title: t('message.createSuccess'), status: 'success' });
      }
      onClose();
      await loadRoles();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await rolesApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      loadRoles();
    } catch (err) {
      toast({ title: t('message.deleteFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  const openPermissions = async (roleId: string) => {
    setPermRoleId(roleId);
    try {
      const [tree, rolePerms] = await Promise.all([
        permissionsApi.tree(),
        rolesApi.getPermissions(roleId),
      ]);
      setPermTree(tree);
      setSelectedPerms(rolePerms.map((p) => p.id));
      onPermOpen();
    } catch (err) {
      toast({ title: t('message.loadPermissionsFailed'), status: 'error' });
    }
  };

  const handleAssignPerms = async () => {
    setIsAssigningPerms(true);
    try {
      await rolesApi.assignPermissions(permRoleId, selectedPerms);
      toast({ title: t('message.assignPermissionsSuccess'), status: 'success' });
      onPermClose();
      await loadRoles();
    } catch (err) {
      toast({ title: t('message.assignPermissionsFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsAssigningPerms(false);
    }
  };

  const getLevelColorScheme = (level: number | undefined): string => {
    if (level == null) return 'gray';
    if (level <= 1) return 'red';
    if (level <= 50) return 'orange';
    return 'gray';
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
        onRefresh={refresh}
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
              <Th>{t('table.columns.name')}</Th>
              <Th>{t('table.columns.description')}</Th>
              <Th>{t('form.level.label')}</Th>
              <Th>{t('table.columns.permissionCount')}</Th>
              <Th>{t('table.columns.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {roles.map((r) => (
              <Tr key={r.id}>
                <Td>{r.id}</Td>
                <Td fontWeight="600">{r.name}</Td>
                <Td>{r.description || '-'}</Td>
                <Td><Badge colorScheme={getLevelColorScheme(r.level)}>{r.level ?? 100}</Badge></Td>
                <Td>{r.permissions?.length ?? 0}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label={t('button.assignPermissions')} icon={<SettingsIcon />} size="sm" variant="ghost" colorScheme="blue" onClick={() => openPermissions(r.id)} />
                    <IconButton aria-label={t('actions.edit')} icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(r)} />
                    <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(r.id)} />
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

      {/* Create/Edit Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent as="form" onSubmit={(e) => { e.preventDefault(); handleSave(); }}>
          <ModalHeader>{editing ? t('modal.editTitle') : t('modal.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb={4}>
              <FormLabel>{t('form.name.label')}</FormLabel>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={t('form.name.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.description.label')}</FormLabel>
              <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder={t('form.description.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.level.label')}</FormLabel>
              <Input
                type="number"
                min={roleLevelMin}
                max={roleLevelMax}
                value={Number.isFinite(form.level) ? form.level : ''}
                onChange={(e) => {
                  const value = e.target.value;
                  if (value === '') {
                    setForm({ ...form, level: Number.NaN });
                    return;
                  }
                  setForm({ ...form, level: Number(value) });
                }}
                placeholder={t('form.level.placeholder')}
              />
              <FormHelperText>{t('form.level.helper')}</FormHelperText>
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" type="submit" isLoading={isSaving}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Permissions Modal */}
      <Modal isOpen={isPermOpen} onClose={onPermClose} size="xl">
        <ModalOverlay backdropFilter="blur(4px)" />
        <ModalContent borderRadius="20px">
          <ModalHeader fontSize="22px" fontWeight="800" color={textColor} pt="25px" px="25px">
            {t('modal.assignPermissionsTitle')}
          </ModalHeader>
          <ModalCloseButton top="25px" right="25px" />
          <ModalBody px="25px" pb="25px">
            <Box maxH="60vh" overflowY="auto" w="100%" pr="2">
              {permTree.length > 0 ? (
                <PermissionTree
                  tree={permTree}
                  selectedIds={selectedPerms}
                  onChange={setSelectedPerms}
                />
              ) : (
                <Text>{t('permissions.empty')}</Text>
              )}
            </Box>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onPermClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleAssignPerms} isLoading={isAssigningPerms}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
