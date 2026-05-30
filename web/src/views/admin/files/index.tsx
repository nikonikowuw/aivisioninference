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
  HStack,
  Center,
  Spinner,
  Progress,
  Badge,
  Checkbox,
} from '@chakra-ui/react';
import { DeleteIcon, DownloadIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback, useRef } from 'react';
import { filesApi, type FileItem } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';

function formatSize(bytes: number, t: (key: string) => string): string {
  if (bytes < 1024) return `${bytes} ${t('size.bytes')}`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} ${t('size.kilobytes')}`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} ${t('size.megabytes')}`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} ${t('size.gigabytes')}`;
}

export default function Files() {
  const { t } = useTranslation('modules/files');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<'delete' | null>(null);
  const [isBatching, setIsBatching] = useState(false);

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();

  const fetchFiles = useCallback((p: number, ps: number) => {
    return filesApi.list({
      page: p,
      page_size: ps,
      keyword: filters.keyword,
      storage_type: filters.storage_type,
      start_time: filters.start_time,
      end_time: filters.end_time,
    });
  }, [filters]);

  const { list: files, total, page, pageSize, initialLoading, pageLoading, load, changePage, changePageSize } = usePagination<FileItem>(fetchFiles);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  const pageFileIds = files.map((f) => f.id);
  const selectedOnPage = pageFileIds.filter((id) => selectedIds.includes(id));
  const isAllPageSelected = pageFileIds.length > 0 && selectedOnPage.length === pageFileIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    if (isAllPageSelected) {
      setSelectedIds((prev) => prev.filter((id) => !pageFileIds.includes(id)));
      return;
    }
    setSelectedIds((prev) => Array.from(new Set([...prev, ...pageFileIds])));
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
      const result = await filesApi.batchDelete(selectedIds);
      toast({
        title: t('message.batchDone', { success: result.success, failed: result.failed }),
        status: result.failed > 0 ? 'warning' : 'success',
      });
      setSelectedIds([]);
      await load();
    } catch (err) {
      toast({ title: t('message.operationFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsBatching(false);
      setBatchAction(null);
    }
  };

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setProgress(0);
    try {
      await filesApi.upload(file, setProgress);
      toast({ title: t('message.uploadSuccess'), status: 'success' });
      load();
    } catch (err) {
      toast({ title: t('message.uploadFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setUploading(false);
      setProgress(0);
      if (inputRef.current) inputRef.current.value = '';
    }
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await filesApi.exportCsv({
        keyword: filters.keyword,
        storage_type: filters.storage_type,
        start_time: filters.start_time,
        end_time: filters.end_time,
      });
    } catch (err) {
      toast({ title: t('message.exportFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await filesApi.delete(deleteTarget);
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
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <HStack spacing={2}>
          <Button leftIcon={<DownloadIcon />} variant="outline" onClick={handleExport} isLoading={isExporting}>{t('actions.export')}</Button>
          <Button variant="brand" onClick={() => inputRef.current?.click()} isLoading={uploading}>
            {t('button.upload')}
          </Button>
        </HStack>
        <input ref={inputRef} type="file" hidden onChange={handleUpload} />
      </Flex>
      {uploading && (
        <Box mb={4}>
          <Text fontSize="sm" mb={1}>{t('upload.uploading')} {progress}%</Text>
          <Progress value={progress} colorScheme="brand" borderRadius="full" />
        </Box>
      )}
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          {
            name: 'storage_type',
            label: t('table.columns.storageType'),
            options: [
              { value: 'local', label: t('filter.storageTypes.local') },
              { value: 'oss', label: t('filter.storageTypes.oss') },
              { value: 'pg', label: t('filter.storageTypes.pg') },
            ],
          },
        ]}
        dateRange
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
              <Th>{t('table.columns.type')}</Th>
              <Th>{t('table.columns.size')}</Th>
              <Th>{t('table.columns.uploadTime')}</Th>
              <Th>{t('table.columns.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {files.map((f) => (
              <Tr key={f.id}>
                <Td>
                  <Checkbox
                    isChecked={selectedIds.includes(f.id)}
                    onChange={() => toggleRowSelection(f.id)}
                  />
                </Td>
                <Td>{f.id}</Td>
                <Td fontWeight="600">{f.original_name}</Td>
                <Td><Badge>{f.mime_type}</Badge></Td>
                <Td>{formatSize(f.size, t)}</Td>
                <Td>{formatDateTime(f.created_at)}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label={t('actions.download')} icon={<DownloadIcon />} size="sm" variant="ghost" colorScheme="blue" onClick={() => filesApi.download(f.id, f.original_name)} />
                    <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(f.id)} />
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
    </Box>
  );
}
