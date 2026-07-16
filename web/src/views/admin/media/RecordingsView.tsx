import {
  Box, Button,
  Center,
  Checkbox,
  Flex,
  HStack,
  Icon,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent, ModalHeader,
  ModalOverlay,
  Spinner,
  Table,
  Tbody,
  Td, Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useDisclosure,
  useToast
} from '@chakra-ui/react';
import VideoPlayer from 'components/VideoPlayer';
import Card from 'components/card/Card';
import { EmptyState } from 'components/empty/EmptyState';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useDateFormat } from 'hooks/useDateFormat';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import React, { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { MdDelete, MdPlayCircle, MdVideoLibrary } from 'react-icons/md';
import { devicesApi, request, type Device } from 'services/api';
import { startGB28181Playback } from 'services/gb28181';

interface Recording { id: string; device_id: string; file_name: string; file_size: number; start_time: string; end_time: string; record_type: string }
interface RecordingPage { list: Recording[]; total: number; page: number; page_size: number }
const formatSize = (bytes: number): string => {
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1048576).toFixed(1)} MB`;
};

export default function RecordingsView() {
  const { t } = useTranslation('modules/media');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { formatDateTime } = useDateFormat();
  const { isOpen: isPlayerOpen, onOpen: openPlayer, onClose: closePlayer } = useDisclosure();
  const [playbackUrl, setPlaybackUrl] = useState('');
  const [playbackProtocol, setPlaybackProtocol] = useState<'hls' | 'flv' | 'webrtc'>('hls');

  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter({ initialValues: { keyword: '' } });
  const fetchRecordings = useCallback((page: number, pageSize: number) => {
    if (!filters.keyword) return Promise.resolve({ list: [], total: 0 });
    const end = filters.end_time || new Date().toISOString();
    const start = filters.start_time || new Date(Date.now() - 7 * 86400000).toISOString();
    return request<RecordingPage>(`/media/recordings?device_id=${filters.keyword}&start=${start}&end=${end}&page=${page}&page_size=${pageSize}`);
  }, [filters.keyword, filters.start_time, filters.end_time]);

  const { list: recordings, total, page, pageSize, pageLoading, load, changePage, changePageSize } = usePagination<Recording>(fetchRecordings);
  React.useEffect(() => { load({ page: 1 }); }, [searchTrigger, load]);

  // ── Batch operations ──
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [isDeleting, setIsDeleting] = useState(false);
  const [batchDeleteOpen, setBatchDeleteOpen] = useState(false);

  const pageIds = recordings.map(r => r.id);
  const selectedOnPage = pageIds.filter(id => selectedIds.includes(id));
  const isAllSelected = pageIds.length > 0 && selectedOnPage.length === pageIds.length;
  const isIndeterminate = selectedOnPage.length > 0 && !isAllSelected;

  const toggleAll = () => {
    setSelectedIds(prev => isAllSelected
      ? prev.filter(id => !pageIds.includes(id))
      : [...new Set([...prev, ...pageIds])]
    );
  };
  const toggleOne = (id: string) => {
    setSelectedIds(prev => prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]);
  };

  const handleBatchDelete = async () => {
    setIsDeleting(true);
    try {
      const result = await request<{ deleted: number; failed: number }>(
        '/media/recordings/batch-delete', { method: 'POST', body: JSON.stringify({ ids: selectedIds }) }
      );
      toast({ title: `已删除 ${result.deleted} 条${result.failed > 0 ? `，失败 ${result.failed} 条` : ''}`, status: result.failed > 0 ? 'warning' : 'success' });
      setSelectedIds([]);
      refresh();
    } catch (err: any) {
      toast({ title: tCommon('message.operationFailed'), status: 'error' });
    } finally {
      setIsDeleting(false);
      setBatchDeleteOpen(false);
    }
  };

  const handlePlayback = async (recording: Recording) => {
    try {
      const device = await devicesApi.get(recording.device_id) as Device;
      if (device.access_type === 'gb28181') {
        const res = await startGB28181Playback(device.id, recording.start_time, recording.end_time);
        setPlaybackUrl(res.url);
        setPlaybackProtocol(res.protocol === 'webrtc' ? 'webrtc' : res.protocol === 'hls' ? 'hls' : 'flv');
      } else {
        const res = await request<{ url: string }>(`/media/recordings/${recording.id}/playback`, { method: 'POST' });
        setPlaybackUrl(res.url);
        setPlaybackProtocol('hls');
      }
      openPlayer();
    } catch {
      toast({ title: t('playbackFailed'), status: 'error', duration: 3000 });
    }
  };

  const recType = (k: string) => t((k + 'Recording') as any);

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('recordingsTitle')}</Text>
      </Flex>

      <SearchBar filters={filters} onFilterChange={setFilter} onReset={resetFilters} onRefresh={refresh} dateRange />

      {/* Batch action bar */}
      {selectedIds.length > 0 && (
        <Flex bg={bgCard} p="4" borderRadius="lg" border="1px solid" borderColor={borderColor} mb="4" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>已选择 {selectedIds.length} 条</Text>
          <HStack spacing={2}>
            <Button size="sm" leftIcon={<MdDelete />} colorScheme="red" variant="outline" onClick={() => setBatchDeleteOpen(true)}>
              {tCommon('button.delete')}
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setSelectedIds([])}>取消选择</Button>
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
                <Th>{t('fileName')}</Th>
                <Th>{t('startTime')}</Th>
                <Th>{t('endTime')}</Th>
                <Th>{t('fileSize')}</Th>
                <Th>{t('recordType')}</Th>
                <Th textAlign="right">{t('action')}</Th>
              </Tr>
            </Thead>
            <Tbody>
              {pageLoading ? (
                <Tr><Td colSpan={7}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
              ) : recordings.length === 0 ? (
                <Tr><Td colSpan={7}><EmptyState title={filters.keyword ? t('noRecordings') : t('enterDeviceIdHint')} /></Td></Tr>
              ) : (
                recordings.map(r => (
                  <Tr key={r.id}>
                    <Td pe="10px">
                      <Checkbox isChecked={selectedIds.includes(r.id)} onChange={() => toggleOne(r.id)} />
                    </Td>
                    <Td>
                      <HStack>
                        <Icon as={MdVideoLibrary} color="brand.500" />
                        <Text color={textColor} fontSize="sm" fontWeight="700">{r.file_name}</Text>
                      </HStack>
                    </Td>
                    <Td><Text fontSize="sm">{formatDateTime(r.start_time)}</Text></Td>
                    <Td><Text fontSize="sm">{formatDateTime(r.end_time)}</Text></Td>
                    <Td><Text fontSize="sm">{formatSize(r.file_size)}</Text></Td>
                    <Td><Text fontSize="sm">{recType(r.record_type)}</Text></Td>
                    <Td textAlign="right">
                      <Button size="xs" leftIcon={<MdPlayCircle />} colorScheme="green" onClick={() => handlePlayback(r)}>
                        {t('playback')}
                      </Button>
                    </Td>
                  </Tr>
                ))
              )}
            </Tbody>
          </Table>
        </Box>
        <Box px="25px">
          <Pagination page={page} pageSize={pageSize} total={total} onChange={changePage} onPageSizeChange={changePageSize} />
        </Box>
      </Card>

      {/* Playback modal */}
      <Modal isOpen={isPlayerOpen} onClose={closePlayer} size="xl" isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('playbackModalTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody pb={4}><Box h="400px">{playbackUrl && <VideoPlayer url={playbackUrl} protocol={playbackProtocol} />}</Box></ModalBody>
        </ModalContent>
      </Modal>

      {/* Batch delete confirm */}
      <ConfirmDialog
        isOpen={batchDeleteOpen}
        onClose={() => setBatchDeleteOpen(false)}
        onConfirm={handleBatchDelete}
        title={tCommon('button.delete')}
        message={`确定删除 ${selectedIds.length} 条录像记录？`}
        isLoading={isDeleting}
      />
    </Box>
  );
}
