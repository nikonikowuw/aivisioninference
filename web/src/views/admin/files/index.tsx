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
import { DeleteIcon, DownloadIcon } from '@chakra-ui/icons';
import { useEffect, useState, useCallback, useRef } from 'react';
import { filesApi, type FileItem } from 'services/api';

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

export default function Files() {
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  const loadFiles = useCallback(async () => {
    try {
      const data = await filesApi.list({ page: 1, page_size: 100 });
      setFiles(data.list);
    } catch {}
  }, []);

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
      toast({ title: '上传成功', status: 'success' });
      loadFiles();
    } catch (err) {
      toast({ title: '上传失败', description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setUploading(false);
      setProgress(0);
      if (inputRef.current) inputRef.current.value = '';
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除该文件？')) return;
    try {
      await filesApi.delete(id);
      toast({ title: '删除成功', status: 'success' });
      loadFiles();
    } catch (err) {
      toast({ title: '删除失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>文件管理</Text>
        <Button variant="brand" onClick={() => inputRef.current?.click()} isLoading={uploading}>
          上传文件
        </Button>
        <input ref={inputRef} type="file" hidden onChange={handleUpload} />
      </Flex>
      {uploading && (
        <Box mb={4}>
          <Text fontSize="sm" mb={1}>上传中... {progress}%</Text>
          <Progress value={progress} colorScheme="brand" borderRadius="full" />
        </Box>
      )}
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr><Th>ID</Th><Th>文件名</Th><Th>类型</Th><Th>大小</Th><Th>上传时间</Th><Th>操作</Th></Tr>
          </Thead>
          <Tbody>
            {files.map((f) => (
              <Tr key={f.id}>
                <Td>{f.id}</Td>
                <Td fontWeight="600">{f.name}</Td>
                <Td><Badge>{f.mime_type}</Badge></Td>
                <Td>{formatSize(f.size)}</Td>
                <Td>{new Date(f.created_at).toLocaleString('zh-CN')}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label="删除" icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => handleDelete(f.id)} />
                  </HStack>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </Box>
  );
}
