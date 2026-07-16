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
  Checkbox,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback } from 'react';
import { deviceGroupsApi, type DeviceGroup } from 'services/api';
import Card from 'components/card/Card';
import { EmptyState } from 'components/empty/EmptyState';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';
import { MdFolder } from 'react-icons/md';

export default function DeviceGroups() {
  const { t } = useTranslation('modules/devices');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const modalBg = useColorModeValue('white', 'navy.700');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchGroups = useCallback(
    (p: number, ps: number) =>
      deviceGroupsApi.list({ page: p, page_size: ps, keyword: filters.keyword }),
    [filters],
  );

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

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const [editing, setEditing] = useState<DeviceGroup | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeviceGroup | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | null>(null);
  const [isBatching, setIsBatching] = useState(false);

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

  const pageIds = groups.map(g => g.id);
  const selectedOnPage = pageIds.filter(id => selectedIds.includes(id));
  const isAllSelected = pageIds.length > 0 && selectedOnPage.length === pageIds.length;
  const isIndeterminate = selectedOnPage.length > 0 && !isAllSelected;

  const toggleAll = () => {
    setSelectedIds(prev => {
      const allSelected = pageIds.every(id => prev.includes(id));
      if (allSelected) {
        return prev.filter(id => !pageIds.includes(id));
      }
      return Array.from(new Set([...prev, ...pageIds]));
    });
  };

  const toggleOne = (id: string) => {
    setSelectedIds(prev =>
      prev.includes(id) ? prev.filter(sid => sid !== id) : [...prev, id]
    );
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

  const handleBatchDelete = async () => {
    setIsBatching(true);
    try {
      const result = await deviceGroupsApi.batchDelete(selectedIds);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      setSelectedIds([]);
      load();
    } catch (err) {
      toast({ title: tCommon('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  if (initialLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('groupFields.groupName')}</Text>
        <HStack spacing={2}>
          <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>
            {t('groupActions.create')}
          </Button>
        </HStack>
      </Flex>

      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
      />

      {selectedIds.length > 0 && (
        <Flex
          bg={useColorModeValue('white', 'navy.800')}
          p="4"
          borderRadius="lg"
          border="1px solid"
          borderColor={useColorModeValue('gray.200', 'whiteAlpha.100')}
          mb="4"
          justify="space-between"
          align="center"
        >
          <Text fontSize="sm" color={textColor}>{tCommon('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <Button size="sm" colorScheme="red" onClick={() => setBatchAction('delete')}>{t('actions.batchDelete')}</Button>
          </HStack>
        </Flex>
      )}

      <Card px="0px" pb="20px">
        <Box overflowX="auto">
          <Table variant="simple" color="gray.500" mb="24px">
            <Thead>
              <Tr>
                <Th pe="10px" w="48px">
                  <Checkbox isChecked={isAllSelected} isIndeterminate={isIndeterminate} onChange={toggleAll} />
                </Th>
                <Th>{t('groupFields.groupName')}</Th>
                <Th>{t('groupFields.description')}</Th>
                <Th>{t('groupFields.deviceCount')}</Th>
                <Th textAlign="right">{t('groupFields.actions')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {pageLoading ? (
                <Tr><Td colSpan={5}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : groups.length === 0 ? (
                <Tr><Td colSpan={5}><EmptyState /></Td></Tr>
              ) : (
                groups.map((group) => (
                  <Tr key={group.id}>
                    <Td pe="10px">
                      <Checkbox isChecked={selectedIds.includes(group.id)} onChange={() => toggleOne(group.id)} />
                    </Td>
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
                        <IconButton aria-label={t('actions.edit')} icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(group)} />
                        <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(group)} />
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
      <Modal isOpen={isOpen} onClose={onClose} isCentered>
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

      {/* Batch Delete Confirmation */}
      <ConfirmDialog
        isOpen={batchAction === 'delete'}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchDelete}
        title={t('actions.batchDelete')}
        message={t('message.batchDeleteConfirm', { count: selectedIds.length })}
        isLoading={isBatching}
      />
    </Box>
  );
}
