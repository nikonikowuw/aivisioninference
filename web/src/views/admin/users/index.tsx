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
  Checkbox,
  CheckboxGroup,
  Radio,
  RadioGroup,
  Stack,
  Switch,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, DownloadIcon, EditIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useRef, useState, useCallback } from 'react';
import { usersApi, rolesApi, type User, type Role, type BatchItemResult } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import AvatarUploader from 'components/avatar-upload/AvatarUploader';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useAuth } from 'contexts/AuthContext';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { parseOptionalNumber } from 'utils/convert';

const MAX_CSV_IMPORT_SIZE = 10 * 1024 * 1024;
const CSV_MIME_TYPES = new Set(['', 'text/csv', 'application/csv', 'application/vnd.ms-excel']);

function validateCsvFile(file: File): string | null {
  const lowerName = file.name.toLowerCase();
  if (!lowerName.endsWith('.csv') || !CSV_MIME_TYPES.has(file.type)) return 'invalidCsvFile';
  if (file.size > MAX_CSV_IMPORT_SIZE) return 'csvFileTooLarge';
  return null;
}

export default function Users() {
  const { user: currentUser, refreshUser } = useAuth();
  const { t } = useTranslation('modules/users');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchUsers = useCallback((page: number, pageSize: number) => usersApi.list({
    page,
    page_size: pageSize,
    keyword: filters.keyword,
    status: parseOptionalNumber(filters.status),
  }), [filters]);

  const { list: users, total, page, pageSize, initialLoading, pageLoading, load: loadUsers, changePage, changePageSize } = usePagination<User>(fetchUsers);

  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [editing, setEditing] = useState<User | null>(null);
  const [avatarUrl, setAvatarUrl] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [toggleTarget, setToggleTarget] = useState<User | null>(null);
  const [isToggling, setIsToggling] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | 'enable' | 'disable' | null>(null);
  const [isBatching, setIsBatching] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [isImporting, setIsImporting] = useState(false);
  const [importFailures, setImportFailures] = useState<BatchItemResult[]>([]);
  const importInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    loadUsers({ page: 1 }).catch(() => {
      toast({ title: tCommon('message.loadFailed'), status: 'error' });
    });
  }, [searchTrigger, loadUsers, toast, tCommon]);

  useEffect(() => {
    rolesApi.list({ page: 1, page_size: 100 }).then((d) => setAllRoles(d.list)).catch(() => {
      toast({ title: tCommon('message.loadFailed'), status: 'error' });
    });
  }, [toast, tCommon]);

  const [form, setForm] = useState({
    username: '',
    display_name: '',
    email: '',
    password: '',
    status: 1,
    role_ids: [] as string[],
  });

  const resetForm = (user?: User) => {
    setEditing(user || null);
    setForm({
      username: user?.username || '',
      display_name: user?.display_name || '',
      email: user?.email || '',
      password: '',
      status: user?.status ?? 1,
      role_ids: user?.roles?.map((r) => r.id) || [],
    });
    setAvatarUrl(user?.avatar_url || '');
    onOpen();
  };

  const openCreate = () => resetForm();
  const openEdit = (user: User) => resetForm(user);

  const handleSave = async () => {
    try {
      if (editing) {
        const { password, ...updateData } = form;
        await usersApi.update(editing.id, updateData);

        if (password) {
          await usersApi.resetPassword(editing.id, password);
        }
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await usersApi.create(form as any);
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

  const pageUserIds = users.map((user) => user.id);
  const selectedOnPage = pageUserIds.filter((id) => selectedIds.includes(id));
  const isAllPageSelected = pageUserIds.length > 0 && selectedOnPage.length === pageUserIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    if (isAllPageSelected) {
      setSelectedIds((prev) => prev.filter((id) => !pageUserIds.includes(id)));
      return;
    }
    setSelectedIds((prev) => Array.from(new Set([...prev, ...pageUserIds])));
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
      const result = batchAction === 'delete'
        ? await usersApi.batchDelete(selectedIds)
        : await usersApi.batchUpdateStatus(selectedIds, batchAction === 'enable' ? 1 : 0);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      setSelectedIds([]);
      await loadUsers();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  const handleImport = async (file?: File) => {
    if (!file) return;
    const validationKey = validateCsvFile(file);
    if (validationKey) {
      toast({ title: t(`message.${validationKey}`), status: 'error' });
      if (importInputRef.current) importInputRef.current.value = '';
      return;
    }

    setIsImporting(true);
    setImportFailures([]);
    try {
      const result = await usersApi.importCsv(file);
      const failures = result.items.filter((item) => !item.success);
      setImportFailures(failures);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        description: failures.slice(0, 3).map((item) => item.message).filter(Boolean).join('\n'),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      await loadUsers({ page: 1 });
    } catch (err) {
      toast({ title: t('message.importFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsImporting(false);
      if (importInputRef.current) importInputRef.current.value = '';
    }
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await usersApi.exportCsv({
        keyword: filters.keyword,
        status: parseOptionalNumber(filters.status),
        ids: selectedIds.length > 0 ? selectedIds.join(',') : undefined,
      });
    } catch (err) {
      toast({ title: t('message.exportFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <HStack spacing={2}>
          <Input
            ref={importInputRef}
            type="file"
            accept=".csv,text/csv"
            display="none"
            onChange={(e) => handleImport(e.target.files?.[0])}
          />
          <Button variant="outline" onClick={() => importInputRef.current?.click()} isLoading={isImporting}>{tCommon('button.import')}</Button>
          <Button
            leftIcon={<DownloadIcon />}
            variant="outline"
            onClick={handleExport}
            isLoading={isExporting}
          >
            {selectedIds.length > 0 ? t('actions.exportSelected') : tCommon('button.export')}
          </Button>
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
      {importFailures.length > 0 && (
        <Box mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px">
          <Text fontSize="sm" fontWeight="bold" color={textColor} mb={2}>{t('message.importPartialFailed')}</Text>
          {importFailures.slice(0, 5).map((item) => (
            <Text key={item.id} fontSize="sm" color="red.500">{item.message}</Text>
          ))}
        </Box>
      )}
      {selectedIds.length > 0 && (
        <Flex mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{t('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <Button size="sm" onClick={() => setBatchAction('enable')}>{t('actions.enable')}</Button>
            <Button size="sm" onClick={() => setBatchAction('disable')}>{t('actions.disable')}</Button>
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
                  aria-label={t('batch.selectPage')}
                />
              </Th>
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
            {users.map((user) => (
              <Tr key={user.id}>
                <Td>
                  <Checkbox
                    isChecked={selectedIds.includes(user.id)}
                    onChange={() => toggleRowSelection(user.id)}
                    aria-label={t('batch.selectRow', { username: user.username })}
                  />
                </Td>
                <Td>{user.id}</Td>
                <Td fontWeight="600">{user.username}</Td>
                <Td>{user.display_name}</Td>
                <Td>{user.email}</Td>
                <Td>
                  <HStack spacing={2}>
                    <Switch
                      aria-label={user.status === 1 ? t('actions.disable') : t('actions.enable')}
                      isChecked={user.status === 1}
                      isDisabled={user.id === currentUser?.id || isToggling || toggleTarget?.id === user.id}
                      onChange={() => setToggleTarget(user)}
                      colorScheme="green"
                    />
                    <Badge colorScheme={user.status === 1 ? 'green' : 'red'}>
                      {user.status === 1 ? t('table.status.active') : t('table.status.inactive')}
                    </Badge>
                  </HStack>
                </Td>
                <Td>{user.roles?.map((r) => r.name).join(', ') || '-'}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label={t('actions.edit')} icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(user)} />
                    <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(user.id)} />
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
      <ConfirmDialog
        isOpen={batchAction !== null}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchConfirm}
        title={batchAction ? t(`batch.${batchAction}`) : ''}
        message={batchAction ? t(`message.batch${batchAction.charAt(0).toUpperCase()}${batchAction.slice(1)}Confirm`, { count: selectedIds.length }) : ''}
        isLoading={isBatching}
      />

      <Modal isOpen={isOpen} onClose={onClose} size="lg">
        <ModalOverlay backdropFilter="blur(4px)" />
        <ModalContent borderRadius="20px">
          <ModalHeader fontSize="22px" fontWeight="800" color={textColor} pt="25px" px="25px">
            {editing ? t('modal.editTitle') : t('modal.createTitle')}
          </ModalHeader>
          <ModalCloseButton top="25px" right="25px" />
          <ModalBody px="25px" pb="25px">
            {editing && (
              <Box textAlign="center" mb={4}>
                <AvatarUploader
                  value={avatarUrl}
                  onChange={async (url) => {
                    setAvatarUrl(url);
                    // 编辑的是当前用户时，同步 auth context 使侧边栏头像立即更新
                    if (editing.id === currentUser?.id) {
                      await refreshUser();
                    }
                  }}
                  userId={editing.id}
                  name={form.display_name || form.username}
                  size={80}
                />
              </Box>
            )}
            <FormControl isRequired mb="24px">
              <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                {t('form.username.label')}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                type="text"
                placeholder={t('form.username.placeholder')}
                fontWeight="500"
                size="lg"
                h="50px"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
              />
            </FormControl>
            <FormControl mb="24px">
              <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                {t('form.displayName.label')}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                type="text"
                placeholder={t('form.displayName.placeholder')}
                fontWeight="500"
                size="lg"
                h="50px"
                value={form.display_name}
                onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              />
            </FormControl>
            <FormControl isRequired mb="24px">
              <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                {t('form.email.label')}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                type="email"
                placeholder={t('form.email.placeholder')}
                fontWeight="500"
                size="lg"
                h="50px"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
              />
            </FormControl>
            <FormControl isRequired={!editing} mb="24px">
              <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                {t('form.password.label')}
                {editing && <Text as="span" fontWeight="400" ms="1">（{t('form.password.hint')}）</Text>}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                type="password"
                placeholder={t('form.password.placeholder')}
                fontWeight="500"
                size="lg"
                h="50px"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
              />
            </FormControl>
            <FormControl mb="24px">
              <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                {t('table.columns.roles')}
              </FormLabel>
              <CheckboxGroup
                colorScheme="brand"
                value={form.role_ids}
                onChange={(values) => setForm({ ...form, role_ids: values as string[] })}
              >
                <Stack spacing={[2, 4]} direction="row" wrap="wrap" p="4px">
                  {allRoles.map((role) => (
                    <Checkbox key={role.id} value={role.id} fontWeight="500" fontSize="sm">
                      {role.name}
                    </Checkbox>
                  ))}
                </Stack>
              </CheckboxGroup>
            </FormControl>
            {!editing && (
              <FormControl mb="24px">
                <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                  {t('form.status.label')}
                </FormLabel>
                <RadioGroup
                  onChange={(val) => setForm({ ...form, status: Number(val) })}
                  value={String(form.status)}
                >
                  <Stack direction="row" spacing={5} ms="4px">
                    <Radio value="1" colorScheme="green">
                      <Text fontSize="sm" fontWeight="500">{t('form.status.active')}</Text>
                    </Radio>
                    <Radio value="0" colorScheme="red">
                      <Text fontSize="sm" fontWeight="500">{t('form.status.inactive')}</Text>
                    </Radio>
                  </Stack>
                </RadioGroup>
              </FormControl>
            )}
          </ModalBody>
          <ModalFooter pb="25px" px="25px">
            <Button
              variant="no-effects"
              mr={3}
              onClick={onClose}
              fontWeight="600"
              fontSize="sm"
              h="44px"
            >
              {tCommon('button.cancel')}
            </Button>
            <Button
              variant="brand"
              onClick={handleSave}
              fontWeight="600"
              fontSize="sm"
              h="44px"
              px="24px"
            >
              {tCommon('button.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
