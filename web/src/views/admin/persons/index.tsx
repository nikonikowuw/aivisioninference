import {
  AddIcon,
  ChevronDownIcon,
  DeleteIcon,
  RepeatIcon,
  DownloadIcon
} from '@chakra-ui/icons';
import { MdFileUpload } from 'react-icons/md';
import {
  Box,
  Button,
  Flex,
  HStack,
  IconButton,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Stack,
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
import { useCallback, useEffect, useState } from 'react';
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
    changePageSize 
  } = usePagination<Person>(fetchPersons);

  useEffect(() => {
    loadPersons({ page: 1 });
  }, [searchTrigger, loadPersons]);

  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [editingPerson, setEditingPerson] = useState<Person | null>(null);
  const [allGroups, setAllGroups] = useState<PersonGroup[]>([]);

  useEffect(() => {
    personGroupsApi.list().then(setAllGroups);
  }, []);

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
      if (editingPerson) {
        await personsApi.update(editingPerson.id, data);
      } else {
        await personsApi.create(data);
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

  const handleRetry = async (id: string) => {
    try {
      await personsApi.retryEmbedding(id);
      toast({
        title: tCommon('message.success'),
        status: 'success',
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

      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md">
          <Thead>
            <Tr>
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
                <Td colSpan={8} textAlign="center" py="40px">
                  <Text color="gray.500">{tCommon('status.loading')}</Text>
                </Td>
              </Tr>
            ) : persons.length === 0 ? (
              <Tr>
                <Td colSpan={8}>
                  <EmptyState onClearFilters={resetFilters} />
                </Td>
              </Tr>
            ) : (
              persons.map((p) => (
                <Tr key={p.id}>
                  <Td>
                    <HStack>
                      <Image
                        src={getFileUrl(p.image_url)}
                        boxSize="40px"
                        borderRadius="full"
                        objectFit="cover"
                        fallbackSrc="/img/placeholder-avatar.png"
                      />
                      <Text fontWeight="600">{p.person_name}</Text>
                    </HStack>
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
                    <Badge colorScheme={statusColorMap[p.embedding_status] || 'gray'} borderRadius="full" px="2">
                      {t(`embedding.status.${p.embedding_status}`)}
                    </Badge>
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
                    <HStack justify="end" spacing="4px">
                      <IconButton
                        aria-label={t('embedding.retry')}
                        icon={<RepeatIcon />}
                        size="sm"
                        variant="ghost"
                        onClick={() => handleRetry(p.id)}
                        isDisabled={p.embedding_status !== 'failed' || !p.embedding_retryable}
                      />
                      <Menu>
                        <MenuButton as={IconButton} icon={<ChevronDownIcon />} size="sm" variant="ghost" />
                        <MenuList>
                          <MenuItem onClick={() => openEdit(p)}>
                            {t('actions.edit')}
                          </MenuItem>
                          <MenuItem color="red.500" onClick={() => setDeleteTarget(p.id)}>
                            {t('actions.delete')}
                          </MenuItem>
                        </MenuList>
                      </Menu>
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
        onSuccess={loadPersons}
      />
    </Box>
  );
}
