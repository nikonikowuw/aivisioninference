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
  Checkbox,
  VStack,
  Badge,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon, SettingsIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { rolesApi, permissionsApi, type Role, type Permission } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';

function getDescendantIds(nodes: Permission[], id: string): string[] {
  const ids: string[] = [];
  const find = (list: Permission[]): boolean => {
    for (const n of list) {
      if (n.id === id) {
        if (n.children) {
          const walk = (children: Permission[]) => {
            for (const c of children) {
              ids.push(c.id);
              if (c.children) walk(c.children);
            }
          };
          walk(n.children);
        }
        return true;
      }
      if (n.children && find(n.children)) return true;
    }
    return false;
  };
  find(nodes);
  return ids;
}

export default function Roles() {
  const { t } = useTranslation('modules/roles');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const { isOpen: isPermOpen, onOpen: onPermOpen, onClose: onPermClose } = useDisclosure();
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
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

  const loadRoles = useCallback(async () => {
    try {
      const data = await rolesApi.list({ page: 1, page_size: 100 });
      setRoles(data.list);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    }
  }, [toast, t]);

  useEffect(() => {
    loadRoles().finally(() => setLoading(false));
  }, [loadRoles]);

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

  const togglePerm = (id: string) => {
    setSelectedPerms((prev) => {
      const isChecked = prev.includes(id);
      const descendantIds = getDescendantIds(permTree, id);

      if (isChecked) {
        return prev.filter((p) => p !== id && !descendantIds.includes(p));
      } else {
        const newSet = new Set([...prev, id, ...descendantIds]);
        return Array.from(newSet);
      }
    });
  };

  const renderPermTree = (nodes: Permission[], depth = 0) =>
    nodes.map((p) => {
      const isLeaf = !p.children || p.children.length === 0;
      const descendantIds = isLeaf ? [] : getDescendantIds(permTree, p.id);
      const checkedDescendants = descendantIds.filter((id) => selectedPerms.includes(id));
      const allChecked = isLeaf ? selectedPerms.includes(p.id) : checkedDescendants.length === descendantIds.length;
      const someChecked = checkedDescendants.length > 0;

      return (
        <Box key={p.id} ml={depth * 6}>
          <Checkbox
            isChecked={allChecked}
            isIndeterminate={someChecked && !allChecked}
            onChange={() => togglePerm(p.id)}
            mb={1}
          >
            {p.name} ({p.code})
          </Checkbox>
          {p.children && renderPermTree(p.children, depth + 1)}
        </Box>
      );
    });

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
                <Td><Badge colorScheme={r.level <= 1 ? 'red' : r.level <= 50 ? 'orange' : 'gray'}>{r.level ?? 100}</Badge></Td>
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
        <ModalContent>
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
            <Button variant="brand" onClick={handleSave} isLoading={isSaving} isDisabled={isSaving}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Permissions Modal */}
      <Modal isOpen={isPermOpen} onClose={onPermClose} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('modal.assignPermissionsTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack align="start" spacing={1} maxH="400px" overflowY="auto">
              {permTree.length > 0 ? renderPermTree(permTree) : <Text>{t('permissions.empty')}</Text>}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onPermClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleAssignPerms} isLoading={isAssigningPerms} isDisabled={isAssigningPerms}>{tCommon('button.save')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
