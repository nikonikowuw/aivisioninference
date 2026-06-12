import {
  AddIcon,
  DeleteIcon,
  EditIcon,
  DownloadIcon,
  RepeatIcon
} from '@chakra-ui/icons';
import { MdFileUpload } from 'react-icons/md';
import {
  Box,
  Button,
  Checkbox,
  Flex,
  HStack,
  IconButton,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalBody,
  ModalCloseButton,
  Stack,
  Spinner,
  Switch,
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
  Image,
  Badge,
  Icon
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import { EmptyState } from 'components/empty/EmptyState';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { TableSkeleton } from 'components/skeleton/Skeleton';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  personsApi,
  personGroupsApi,
  type Person,
  type PersonGroup,
  getFileUrl,
} from 'services/api';
import { PersonFormModal } from './components/PersonFormModal';
import { PersonImportModal } from './components/PersonImportModal';

const statusColorMap: Record<string, string> = {
  pending: 'yellow',
  extracting: 'blue',
  active: 'green',
  failed: 'red',
  disabled: 'gray',
};

const batchActionTitleKeys: Record<string, string> = {
  'delete': 'dialog.delete.title',
  'retry-embedding': 'dialog.retryEmbedding.title',
  'enable': 'dialog.enable.title',
  'disable': 'dialog.disable.title',
};

function getBatchMessageKey(action: string): string {
  switch (action) {
    case 'retry-embedding': return 'embedding.batchRetryConfirm';
    case 'enable': return 'message.batchEnableConfirm';
    case 'disable': return 'message.batchDisableConfirm';
    default: return 'message.batchDeleteConfirm';
  }
}

export default function PersonsPage() {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');

  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const {
    isOpen: isImportOpen,
    onOpen: onImportOpen,
    onClose: onImportClose
  } = useDisclosure();
  const {
    isOpen: isImagePreviewOpen,
    onOpen: onImagePreviewOpen,
    onClose: onImagePreviewClose
  } = useDisclosure();
  const [previewImageUrl, setPreviewImageUrl] = useState<string>('');

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchPersons = useCallback((page: number, pageSize: number) => personsApi.list({
    page,
    page_size: pageSize,
    keyword: filters.keyword,
    embedding_status: filters.embedding_status,
    group_id: filters.group_id,
  }), [filters]);

  const {
    list: persons,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load: loadPersons,
    changePage,
    changePageSize,
    setList
  } = usePagination<Person>(fetchPersons);

  useEffect(() => {
    loadPersons({ page: 1 });
  }, [searchTrigger, loadPersons]);

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
const [batchAction, setBatchAction] = useState<'delete' | 'enable' | 'disable' | 'retry-embedding' | null>(null);
const [isBatching, setIsBatching] = useState(false);
const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [editingPerson, setEditingPerson] = useState<Person | null>(null);
  const [allGroups, setAllGroups] = useState<PersonGroup[]>([]);

  useEffect(() => {
    personGroupsApi.list().then(setAllGroups);
  }, []);

  const pagePersonIds = persons.map((p) => p.id);
  const selectedOnPage = pagePersonIds.filter((id) => selectedIds.includes(id));
  const isAllPageSelected = pagePersonIds.length > 0 && selectedOnPage.length === pagePersonIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    setSelectedIds((prev) => {
      const allSelected = pagePersonIds.every((id) => prev.includes(id));
      if (allSelected) {
        return prev.filter((id) => !pagePersonIds.includes(id));
      }
      return Array.from(new Set([...prev, ...pagePersonIds]));
    });
  };

  const toggleRowSelection = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((sid) => sid !== id) : [...prev, id]
    );
  };

  const openCreate = () => {
    setEditingPerson(null);
    onOpen();
  };

  const openEdit = (person: Person) => {
    setEditingPerson(person);
    onOpen();
  };

  const handleSave = async (data: FormData | Record<string, unknown>) => {
    try {
      let personId: string;
      if (editingPerson) {
        await personsApi.update(editingPerson.id, data);
        personId = editingPerson.id;
        // 乐观更新状态,让加载圈立即显示
        setList(prev => prev.map(p => p.id === personId ? { ...p, embedding_status: 'extracting' } : p));
      } else {
        const person = await personsApi.create(data);
        personId = person.id;
      }
      toast({
        title: tCommon('message.success'),
        status: 'success',
        duration: 2000,
      });
      onClose();
      loadPersons();
    } catch (error: any) {
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      });
    }
  };

  const handleBatchConfirm = async () => {
    if (!batchAction || selectedIds.length === 0) return;
    setIsBatching(true);
    try {
      if (batchAction === 'delete') {
        await personsApi.batchDelete(selectedIds);
        toast({
          title: tCommon('message.success'),
          status: 'success',
        });
      } else if (batchAction === 'retry-embedding') {
        const result = await personsApi.batchRetryEmbedding(selectedIds) as unknown as { Success: number; Failed: number; Errors?: Record<string, string> };
        if (result.Failed > 0) {
          toast({
            title: t('embedding.batchRetryResult', { success: result.Success, failed: result.Failed }),
            status: 'warning',
          });
        } else {
          toast({
            title: tCommon('message.success'),
            status: 'success',
          });
        }
        // 乐观更新
        setList(prev => prev.map(p => selectedIds.includes(p.id) ? { ...p, embedding_status: 'extracting' } : p));
      } else {
        await personsApi.batchToggle(selectedIds, batchAction === 'enable');
        toast({
          title: tCommon('message.success'),
          status: 'success',
        });
      }
      setSelectedIds([]);
      await loadPersons();
    } catch (error: any) {
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
      });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  const handleToggle = async (id: string, enabled: boolean) => {
    try {
      await personsApi.batchToggle([id], enabled);
      toast({
        title: tCommon('message.success'),
        status: 'success',
        duration: 2000,
      });
      loadPersons();
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
      await personsApi.delete(deleteTarget);
      toast({
        title: tCommon('message.success'),
        status: 'success',
      });
      setDeleteTarget(null);
      loadPersons();
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

  // 检测是否有人在待提取或提取中,只要有,就静默轮询
  const hasPending = persons.some(p => p.embedding_status === 'pending' || p.embedding_status === 'extracting');

  useEffect(() => {
    if (!hasPending) return;
    const interval = setInterval(() => {
      loadPersons({ silent: true });
    }, 2000);
    return () => clearInterval(interval);
  }, [hasPending, loadPersons]);

  const handleRetry = async (id: string) => {
    const prevStatus = persons.find(p => p.id === id)?.embedding_status;
    try {
      // 乐观更新状态,让加载圈立即显示
      setList(prev => prev.map(p => p.id === id ? { ...p, embedding_status: 'extracting' } : p));
      await personsApi.retryEmbedding(id);
      toast({
        title: tCommon('message.success'),
        status: 'success',
      });
      loadPersons();
    } catch (error: any) {
      // 失败时回滚状态
      setList(prev => prev.map(p => p.id === id ? { ...p, embedding_status: prevStatus || 'failed' } : p));
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
      });
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
            {t('title')}
          </Text>
          <HStack spacing="12px" mt={{ base: '10px', md: '0' }}>
            <Button leftIcon={<Icon as={MdFileUpload} />} variant="outline" onClick={onImportOpen}>
              {t('actions.import')}
            </Button>
            <Button leftIcon={<DownloadIcon />} variant="outline" onClick={() => personsApi.exportExcel()}>
              {tCommon('button.export')}
            </Button>
            <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>
              {tCommon('button.create')}
            </Button>
          </HStack>
        </Flex>

        <SearchBar
          filters={filters}
          onFilterChange={setFilter}
          onReset={resetFilters}
          onRefresh={refresh}
          selects={[
            {
              name: 'group_id',
              label: t('table.columns.groups'),
              options: allGroups.map(g => ({ value: g.id, label: g.group_name })),
            },
            {
              name: 'embedding_status',
              label: t('table.columns.embeddingStatus'),
              options: [
                { value: 'pending', label: t('embedding.status.pending') },
                { value: 'extracting', label: t('embedding.status.extracting') },
                { value: 'active', label: t('embedding.status.active') },
                { value: 'failed', label: t('embedding.status.failed') },
                { value: 'disabled', label: t('embedding.status.disabled') },
              ],
            },
          ]}
        />
      </Flex>

      {selectedIds.length > 0 && (
        <Flex mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{t('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            <IconButton
              aria-label={t('batch.retryEmbedding')}
              icon={<RepeatIcon />}
              size="sm"
              variant="ghost"
              onClick={() => setBatchAction('retry-embedding')}
            />
            <Button size="sm" onClick={() => setBatchAction('enable')}>{t('batch.enable')}</Button>
            <Button size="sm" onClick={() => setBatchAction('disable')}>{t('batch.disable')}</Button>
            <Button size="sm" colorScheme="red" onClick={() => setBatchAction('delete')}>{t('batch.delete')}</Button>
          </HStack>
        </Flex>
      )}

      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md">
          <Thead>
            <Tr>
              <Th w="48px">
                <Checkbox
                  isChecked={isAllPageSelected}
                  isIndeterminate={isPageSelectionIndeterminate}
                  onChange={togglePageSelection}
                />
              </Th>
              <Th>{t('table.columns.faceImage')}</Th>
              <Th>{t('table.columns.personName')}</Th>
              <Th>{t('table.columns.personCode')}</Th>
              <Th>{t('table.columns.groups')}</Th>
              <Th>{t('table.columns.phone')}</Th>
              <Th>{t('table.columns.embeddingStatus')}</Th>
              <Th>{t('table.columns.enabled')}</Th>
              <Th>{t('table.columns.createdAt')}</Th>
              <Th isNumeric>{tCommon('actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {initialLoading ? (
              <Tr>
                <Td colSpan={10} textAlign="center" py="40px">
                  <Text color="gray.500">{tCommon('status.loading')}</Text>
                </Td>
              </Tr>
            ) : persons.length === 0 ? (
              <Tr>
                <Td colSpan={10}>
                  <EmptyState onClearFilters={resetFilters} />
                </Td>
              </Tr>
            ) : (
              persons.map((p) => (
                <Tr key={p.id}>
                  <Td>
                    <Checkbox
                      isChecked={selectedIds.includes(p.id)}
                      onChange={() => toggleRowSelection(p.id)}
                    />
                  </Td>
                  <Td>
                    <Image
                      src={getFileUrl(p.image_url)}
                      boxSize="50px"
                      minW="50px"
                      objectFit="cover"
                      borderRadius="md"
                      fallbackSrc="/img/placeholder-avatar.png"
                      cursor="pointer"
                      onClick={() => {
                        setPreviewImageUrl(getFileUrl(p.image_url));
                        onImagePreviewOpen();
                      }}
                    />
                  </Td>
                  <Td>
                    <Text fontWeight="600">{p.person_name}</Text>
                  </Td>
                  <Td>{p.person_code}</Td>
                  <Td>
                    <HStack spacing={1} wrap="wrap">
                      {p.groups?.map(g => (
                        <Badge key={g.id} variant="subtle" colorScheme="brand" fontSize="xs">
                          {g.group_name}
                        </Badge>
                      ))}
                    </HStack>
                  </Td>
                  <Td>{p.phone || '-'}</Td>
                  <Td>
                    {(p.embedding_status === 'pending' || p.embedding_status === 'extracting') ? (
                      <HStack spacing="2">
                        <Spinner size="sm" color="blue.500" />
                        <Badge colorScheme="blue" borderRadius="full" px="2">
                          {t(`embedding.status.${p.embedding_status}`)}
                        </Badge>
                      </HStack>
                    ) : (
                      <Badge colorScheme={statusColorMap[p.embedding_status] || 'gray'} borderRadius="full" px="2">
                        {t(`embedding.status.${p.embedding_status}`)}
                      </Badge>
                    )}
                  </Td>
                  <Td>
                    <Switch
                      isChecked={p.enabled}
                      colorScheme="brand"
                      onChange={(e) => handleToggle(p.id, e.target.checked)}
                      size="sm"
                    />
                  </Td>
                  <Td>{new Date(p.created_at).toLocaleDateString()}</Td>
                  <Td isNumeric>
                    <HStack justify="end" spacing={2}>
                      <IconButton
                        aria-label={t('embedding.retry')}
                        icon={<RepeatIcon />}
                        size="sm"
                        variant="ghost"
                        onClick={() => handleRetry(p.id)}
                        isDisabled={p.embedding_status === 'extracting'}
                      />
                      <IconButton
                        aria-label={t('actions.edit')}
                        icon={<EditIcon />}
                        size="sm"
                        variant="ghost"
                        onClick={() => openEdit(p)}
                      />
                      <IconButton
                        aria-label={t('actions.delete')}
                        icon={<DeleteIcon />}
                        size="sm"
                        variant="ghost"
                        colorScheme="red"
                        onClick={() => setDeleteTarget(p.id)}
                      />
                    </HStack>
                  </Td>
                </Tr>
              ))
            )}
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
        isLoading={isDeleting}
        title={tCommon('dialog.delete.title')}
        message={t('message.deleteConfirm')}
      />

      <ConfirmDialog
        isOpen={batchAction !== null}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchConfirm}
        isLoading={isBatching}
        title={tCommon(batchActionTitleKeys[batchAction!] || 'dialog.delete.title')}
        message={t(getBatchMessageKey(batchAction!), { count: selectedIds.length })}
      />

      <PersonFormModal
        isOpen={isOpen}
        onClose={onClose}
        onSave={handleSave}
        editingPerson={editingPerson}
        allGroups={allGroups}
      />

      <PersonImportModal
        isOpen={isImportOpen}
        onClose={onImportClose}
        onSuccess={() => { loadPersons(); }}
      />

      {/* 头像大图预览 */}
      <Modal isOpen={isImagePreviewOpen} onClose={onImagePreviewClose} size="xl" isCentered>
        <ModalOverlay bg="blackAlpha.800" />
        <ModalContent bg="transparent" boxShadow="none">
          <ModalCloseButton color="white" bg="blackAlpha.500" borderRadius="full" />
          <ModalBody p={0}>
            <Image
              src={previewImageUrl}
              w="100%"
              h="auto"
              maxH="80vh"
              objectFit="contain"
              borderRadius="lg"
              fallbackSrc="/img/placeholder-avatar.png"
            />
          </ModalBody>
        </ModalContent>
      </Modal>
    </Box>
  );
}
