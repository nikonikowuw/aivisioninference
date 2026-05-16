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
  Spinner,
  Center,
  Progress,
  Badge,
} from '@chakra-ui/react';
import { DeleteIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback, useRef } from 'react';
import { filesApi, type FileItem } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';

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
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const loadFiles = useCallback(async () => {
    try {
      const data = await filesApi.list({ page: 1, page_size: 100 });
      setFiles(data.list);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    }
  }, [toast, t]);

  useEffect(() => {
    loadFiles().finally(() => setLoading(false));
  }, [loadFiles]);

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setProgress(0);
    try {
      await filesApi.upload(file, setProgress);
      toast({ title: t('message.uploadSuccess'), status: 'success' });
      loadFiles();
    } catch (err) {
      toast({ title: t('message.uploadFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setUploading(false);
      setProgress(0);
      if (inputRef.current) inputRef.current.value = '';
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await filesApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      loadFiles();
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
        <Button variant="brand" onClick={() => inputRef.current?.click()} isLoading={uploading}>
          {t('button.upload')}
        </Button>
        <input ref={inputRef} type="file" hidden onChange={handleUpload} />
      </Flex>
      {uploading && (
        <Box mb={4}>
          <Text fontSize="sm" mb={1}>{t('upload.uploading')} {progress}%</Text>
          <Progress value={progress} colorScheme="brand" borderRadius="full" />
        </Box>
      )}
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr>
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
                <Td>{f.id}</Td>
                <Td fontWeight="600">{f.name}</Td>
                <Td><Badge>{f.mime_type}</Badge></Td>
                <Td>{formatSize(f.size, t)}</Td>
                <Td>{formatDateTime(f.created_at)}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label={t('actions.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(f.id)} />
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
    </Box>
  );
}
