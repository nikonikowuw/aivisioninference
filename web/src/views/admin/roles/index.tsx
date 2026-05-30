import { AddIcon, DeleteIcon, EditIcon, SettingsIcon } from '@chakra-ui/icons';
import {
  Badge,
  Box,
  Button,
  Center,
  Checkbox,
  Flex,
  FormControl,
  FormHelperText,
  FormLabel,
  HStack,
  IconButton,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  NumberDecrementStepper,
  NumberIncrementStepper,
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  Spinner,
  Switch,
  Table,
  Tbody,
  Td,
  Text,
  Textarea,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import PermissionTree from 'components/permission-tree/PermissionTree';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { permissionsApi, rolesApi, type Permission, type Role } from 'services/api';
import { parseOptionalNumber } from 'utils/convert';

const defaultRoleForm = { name: '', description: '', level: 100, status: 1 };
const roleLevelMin = 1;
const roleLevelMax = 99999;

export default function Roles() {
  const { t } = useTranslation('modules/roles');
  const { t: tCommon } = useTranslation('common');

  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const modalBorderColor = useColorModeValue('gray.100', 'whiteAlpha.100');
  const inputBorderColor = useColorModeValue('gray.200', 'whiteAlpha.200');
  const statusControlBg = useColorModeValue('gray.50', 'whiteAlpha.50');
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
  const [form, setForm] = useState(defaultRoleForm);
  const [isSaving, setIsSaving] = useState(false);
  const [permTree, setPermTree] = useState<Permission[]>([]);
  const [selectedPerms, setSelectedPerms] = useState<string[]>([]);
  const [permRoleId, setPermRoleId] = useState<string>('');
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isAssigningPerms, setIsAssigningPerms] = useState(false);

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | null>(null);
  const [isBatching, setIsBatching] = useState(false);

  useEffect(() => {
    loadRoles({ page: 1 });
  }, [searchTrigger, loadRoles]);

  const selectedIdSet = useMemo(() => new Set(selectedIds), [selectedIds]);
  const pageRoleIds = useMemo(() => roles.map((role) => role.id), [roles]);
  const selectedOnPage = pageRoleIds.filter((id) => selectedIdSet.has(id));
  const isAllPageSelected = pageRoleIds.length > 0 && selectedOnPage.length === pageRoleIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    if (isAllPageSelected) {
      setSelectedIds((prev) => prev.filter((id) => !pageRoleIds.includes(id)));
      return;
    }
    setSelectedIds((prev) => Array.from(new Set([...prev, ...pageRoleIds])));
  };

  const toggleRowSelection = (id: string) => {
    setSelectedIds((prev) => (
      prev.includes(id) ? prev.filter((selectedId) => selectedId !== id) : [...prev, id]
    ));
  };

  const handleBatchConfirm = async () => {
    if (!batchAction || selectedIds.length === 0) return;
    setIsBatching(true);
    try {
      const result = await rolesApi.batchDelete(selectedIds);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      setSelectedIds([]);
      await loadRoles();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  const openCreate = () => {
    setEditing(null);
    setForm(defaultRoleForm);
    onOpen();
  };

  const openEdit = (role: Role) => {
    setEditing(role);
    setForm({
      name: role.name,
      description: role.description,
      level: role.level ?? defaultRoleForm.level,
      status: role.status ?? defaultRoleForm.status,
    });
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

  const changeRoleStatus = async (role: Role, status: number) => {
    try {
      await rolesApi.update(role.id, { ...role, status });
      toast({ title: t('message.updateSuccess'), status: 'success' });
      loadRoles();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
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
        <HStack spacing={2}>
          <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>{t('button.create')}</Button>
        </HStack>
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
      {selectedIds.length > 0 && (
        <Flex mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{t('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <Button size="sm" colorScheme="red" onClick={() => setBatchAction('delete')}>{t('actions.delete')}</Button>
          </HStack>
        </Flex>
      )}
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="700px">
          <Thead>
            <Tr>
              <Th w="48px">
                <Checkbox
                  isChecked={isAllPageSelected}
                  isIndeterminate={isPageSelectionIndeterminate}
                  onChange={togglePageSelection}
                />
              </Th>
              <Th>{t('table.columns.id')}</Th>
              <Th>{t('table.columns.name')}</Th>
              <Th>{t('table.columns.description')}</Th>
              <Th>{t('form.level.label')}</Th>
              <Th>{t('table.columns.status')}</Th>
              <Th>{t('table.columns.permissionCount')}</Th>
              <Th>{t('table.columns.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {roles.map((r) => (
              <Tr key={r.id}>
                <Td>
                  <Checkbox
                    isChecked={selectedIdSet.has(r.id)}
                    onChange={() => toggleRowSelection(r.id)}
                  />
                </Td>
                <Td>{r.id}</Td>
                <Td fontWeight="600">{r.name}</Td>
                <Td>{r.description || '-'}</Td>
                <Td><Badge colorScheme={getLevelColorScheme(r.level)}>{r.level ?? 100}</Badge></Td>
                <Td>
                  <Switch
                    colorScheme="green"
                    isChecked={r.status === 1}
                    onChange={(e) => changeRoleStatus(r, e.target.checked ? 1 : 0)}
                  />
                </Td>
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
      <ConfirmDialog
        isOpen={batchAction !== null}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchConfirm}
        title={t('actions.delete')}
        message={t('message.batchDeleteConfirm', { count: selectedIds.length })}
        isLoading={isBatching}
      />

      {/* Create/Edit Modal */}
      <Modal isOpen={isOpen} onClose={onClose} size="md">
        <ModalOverlay backdropFilter="blur(8px)" />
        <ModalContent
          as="form"
          onSubmit={(e) => { e.preventDefault(); handleSave(); }}
          borderRadius="24px"
          border="1px solid"
          borderColor={modalBorderColor}
          boxShadow="2xl"
          overflow="hidden"
        >
          <ModalHeader
            fontSize="22px"
            fontWeight="800"
            color={textColor}
            pt="30px"
            px="30px"
            pb="15px"
          >
            {editing ? t('modal.editTitle') : t('modal.createTitle')}
          </ModalHeader>
          <ModalCloseButton top="25px" right="30px" borderRadius="12px" />
          <ModalBody px="30px" py="10px">
            <FormControl mb="20px" isRequired>
              <FormLabel fontSize="sm" fontWeight="600" color={textColor} mb="8px">
                {t('form.name.label')}
              </FormLabel>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder={t('form.name.placeholder')}
                borderRadius="16px"
                h="46px"
                fontSize="sm"
                variant="outline"
                borderColor={inputBorderColor}
                _focus={{
                  borderColor: 'brand.500',
                  boxShadow: '0 0 0 1px var(--chakra-colors-brand-500)',
                }}
              />
            </FormControl>

            <FormControl mb="20px">
              <FormLabel fontSize="sm" fontWeight="600" color={textColor} mb="8px">
                {t('form.description.label')}
              </FormLabel>
              <Textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder={t('form.description.placeholder')}
                borderRadius="16px"
                fontSize="sm"
                minH="90px"
                py="12px"
                variant="outline"
                borderColor={inputBorderColor}
                _focus={{
                  borderColor: 'brand.500',
                  boxShadow: '0 0 0 1px var(--chakra-colors-brand-500)',
                }}
              />
            </FormControl>

            <FormControl mb="20px">
              <FormLabel fontSize="sm" fontWeight="600" color={textColor} mb="8px">
                {t('form.level.label')}
              </FormLabel>
              <NumberInput
                min={roleLevelMin}
                max={roleLevelMax}
                value={Number.isFinite(form.level) ? form.level : defaultRoleForm.level}
                variant="outline"
                onChange={(valueStr, valueNum) => {
                  if (valueStr === '') {
                    setForm({ ...form, level: Number.NaN });
                    return;
                  }
                  setForm({ ...form, level: valueNum });
                }}
              >
                <NumberInputField
                  placeholder={t('form.level.placeholder')}
                  borderRadius="16px"
                  h="46px"
                  fontSize="sm"
                  borderColor={inputBorderColor}
                  _focus={{
                    borderColor: 'brand.500',
                    boxShadow: '0 0 0 1px var(--chakra-colors-brand-500)',
                  }}
                />
                <NumberInputStepper mr="6px">
                  <NumberIncrementStepper border="none" color="gray.500" _active={{ color: 'brand.500' }} />
                  <NumberDecrementStepper border="none" color="gray.500" _active={{ color: 'brand.500' }} />
                </NumberInputStepper>
              </NumberInput>
              <FormHelperText fontSize="xs" color="gray.500" mt="6px">
                {t('form.level.helper')}
              </FormHelperText>
            </FormControl>

            <FormControl
              display="flex"
              alignItems="center"
              justifyContent="space-between"
              mb="10px"
              p="14px 18px"
              border="1px solid"
              borderColor={borderColor}
              borderRadius="16px"
              bg={statusControlBg}
            >
              <Box>
                <FormLabel mb="0" fontSize="sm" fontWeight="600" color={textColor}>
                  {t('table.columns.status')}
                </FormLabel>
                <Text fontSize="xs" color="gray.500" mt="2px">
                  {form.status === 1 ? t('table.status.active') : t('table.status.inactive')}
                </Text>
              </Box>
              <Switch
                colorScheme="brand"
                isChecked={form.status === 1}
                onChange={(e) => setForm({ ...form, status: e.target.checked ? 1 : 0 })}
              />
            </FormControl>
          </ModalBody>
          <ModalFooter px="30px" pt="15px" pb="30px">
            <Button
              variant="ghost"
              mr="12px"
              onClick={onClose}
              borderRadius="16px"
              h="46px"
              px="24px"
              fontSize="sm"
            >
              {tCommon('button.cancel')}
            </Button>
            <Button
              variant="brand"
              type="submit"
              isLoading={isSaving}
              borderRadius="16px"
              h="46px"
              px="24px"
              fontSize="sm"
            >
              {tCommon('button.save')}
            </Button>
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
