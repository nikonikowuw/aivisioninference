import { CopyIcon, DownloadIcon, ViewIcon } from '@chakra-ui/icons';
import {
  Badge,
  Box,
  Button,
  Checkbox,
  Divider,
  Flex,
  Grid,
  GridItem,
  HStack,
  IconButton,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Select,
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
  VStack,
} from '@chakra-ui/react';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useDateFormat } from 'hooks/useDateFormat';
import { useFilter } from 'hooks/useFilter';
import { usePagination } from 'hooks/usePagination';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import ReactMarkdown from 'react-markdown';
import { feedbackApi, type Feedback } from 'services/api';

const feedbackStatuses = ['open', 'processing', 'resolved', 'closed'] as const;
type FeedbackStatus = (typeof feedbackStatuses)[number];

const statusColor: Record<string, string> = {
  open: 'yellow',
  processing: 'blue',
  resolved: 'green',
  closed: 'gray',
};

export default function FeedbackPage() {
  const { t } = useTranslation('modules/feedback');
  const { t: tCommon } = useTranslation('common');
  const toast = useToast();
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const modalBg = useColorModeValue('white', 'navy.800');
  const markdownBg = useColorModeValue('gray.50', 'navy.900');
  const markdownCodeBg = useColorModeValue('#edf2f7', 'rgba(255,255,255,0.1)');
  const { filters, setFilter, resetFilters, searchTrigger, refresh } = useFilter();
  const [isExporting, setIsExporting] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [batchAction, setBatchAction] = useState<FeedbackStatus | null>(null);
  const [isBatching, setIsBatching] = useState(false);

  const [selectedFeedback, setSelectedFeedback] = useState<Feedback | null>(null);
  const { isOpen: isDetailOpen, onOpen: onDetailOpen, onClose: onDetailClose } = useDisclosure();

  const fetchFeedback = useMemo(() => (p: number, ps: number) => feedbackApi.list({
    page: p,
    page_size: ps,
    keyword: filters.keyword,
    source: filters.source,
    status: filters.status,
    start_time: filters.start_time,
    end_time: filters.end_time,
  }), [filters]);

  const { list, total, page, pageSize, load, changePage, changePageSize } = usePagination<Feedback>(fetchFeedback);

  useEffect(() => { load({ page: 1 }); }, [searchTrigger, load]);

  const selectedIdSet = useMemo(() => new Set(selectedIds), [selectedIds]);
  const pageIds = useMemo(() => list.map(item => item.id), [list]);
  const selectedOnPage = pageIds.filter(id => selectedIdSet.has(id));
  const isAllPageSelected = pageIds.length > 0 && selectedOnPage.length === pageIds.length;
  const isPageSelectionIndeterminate = selectedOnPage.length > 0 && !isAllPageSelected;

  const togglePageSelection = () => {
    if (isAllPageSelected) {
      setSelectedIds(prev => prev.filter(id => !pageIds.includes(id)));
      return;
    }
    setSelectedIds(prev => Array.from(new Set([...prev, ...pageIds])));
  };

  const toggleRowSelection = (id: string) => {
    setSelectedIds(prev => (
      prev.includes(id) ? prev.filter(selectedId => selectedId !== id) : [...prev, id]
    ));
  };

  const openFeedbackDetail = (feedback: Feedback) => {
    setSelectedFeedback(feedback);
    onDetailOpen();
  };

  const copyEmail = async (email: string) => {
    try {
      await navigator.clipboard.writeText(email);
      toast({ title: t('detail.copySuccess'), status: 'success', duration: 2000 });
    } catch {
      toast({ title: t('detail.copyFailed'), status: 'error', duration: 2000 });
    }
  };

  const handleBatchConfirm = async () => {
    if (!batchAction || selectedIds.length === 0) return;
    setIsBatching(true);
    try {
      const result = await feedbackApi.batchUpdateStatus(selectedIds, batchAction);
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

  async function changeStatus(id: string, status: string) {
    try {
      await feedbackApi.updateStatus(id, status);
      toast({ title: t('message.updated'), status: 'success' });
      load();
      return true;
    } catch (err) {
      toast({ title: t('message.updateFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
      return false;
    }
  }

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await feedbackApi.exportCsv({
        keyword: filters.keyword,
        source: filters.source,
        status: filters.status,
        start_time: filters.start_time,
        end_time: filters.end_time,
        ids: selectedIds.length > 0 ? selectedIds.join(',') : undefined,
      });
    } catch (err) {
      toast({ title: tCommon('message.exportFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button
          leftIcon={<DownloadIcon />}
          variant="outline"
          onClick={handleExport}
          isLoading={isExporting}
        >
          {selectedIds.length > 0 ? tCommon('button.exportSelected') : tCommon('button.export')}
        </Button>
      </Flex>
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          { name: 'source', label: t('table.source'), options: [{ value: 'user', label: t('source.user') }, { value: 'email', label: t('source.email') }] },
          { name: 'status', label: t('table.status'), options: [{ value: 'open', label: t('status.open') }, { value: 'processing', label: t('status.processing') }, { value: 'resolved', label: t('status.resolved') }, { value: 'closed', label: t('status.closed') }] },
        ]}
        dateRange
      />
      {selectedIds.length > 0 && (
        <Flex mb={4} p={3} bg={bgCard} border="1px solid" borderColor={borderColor} borderRadius="12px" justify="space-between" align="center">
          <Text fontSize="sm" color={textColor}>{t('batch.selected', { count: selectedIds.length })}</Text>
          <HStack spacing={2}>
            {feedbackStatuses.map(status => (
              <Button key={status} size="sm" onClick={() => setBatchAction(status)}>
                {t(`status.${status}`)}
              </Button>
            ))}
          </HStack>
        </Flex>
      )}
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="auto">
        <Table variant="simple" size="md" minW="900px">
          <Thead>
            <Tr>
              <Th w="48px">
                <Checkbox
                  isChecked={isAllPageSelected}
                  isIndeterminate={isPageSelectionIndeterminate}
                  onChange={togglePageSelection}
                />
              </Th>
              <Th>{t('table.columns.id', { ns: 'common', defaultValue: 'ID' })}</Th>
              <Th>{t('table.source')}</Th>
              <Th>{t('table.category')}</Th>
              <Th>{t('table.title')}</Th>
              <Th>{t('table.content')}</Th>
              <Th>{t('table.status')}</Th>
              <Th>{t('table.createdAt')}</Th>
              <Th>{t('table.actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {list.map(item => (
              <Tr key={item.id}>
                <Td>
                  <Checkbox
                    isChecked={selectedIdSet.has(item.id)}
                    onChange={() => toggleRowSelection(item.id)}
                  />
                </Td>
                <Td>{item.id}</Td>
                <Td>{t(`source.${item.source}`)}</Td>
                <Td>{t(`filter.resourceTypes.${item.category}`, { ns: 'modules/audit-logs', defaultValue: item.category || '-' })}</Td>
                <Td>{item.title}</Td>
                <Td maxW="320px">
                  <Text
                    noOfLines={1}
                    cursor="pointer"
                    onClick={() => openFeedbackDetail(item)}
                    _hover={{ color: 'brand.500', textDecoration: 'underline' }}
                  >
                    {item.content}
                  </Text>
                </Td>
                <Td><Badge colorScheme={statusColor[item.status] || 'gray'}>{t(`status.${item.status}`)}</Badge></Td>
                <Td>{formatDateTime(item.created_at)}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton
                      aria-label={t('actions.viewDetails')}
                      icon={<ViewIcon />}
                      size="sm"
                      variant="ghost"
                      onClick={() => openFeedbackDetail(item)}
                    />
                    <Select size="sm" w="110px" value={item.status} onChange={(e) => changeStatus(item.id, e.target.value)}>
                      {feedbackStatuses.map(status => (
                        <option key={status} value={status}>{t(`status.${status}`)}</option>
                      ))}
                    </Select>
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
        />
      </Box>
      <ConfirmDialog
        isOpen={batchAction !== null}
        onClose={() => setBatchAction(null)}
        onConfirm={handleBatchConfirm}
        title={t('actions.batchUpdateStatus')}
        message={t('message.batchUpdateConfirm', { count: selectedIds.length })}
        isLoading={isBatching}
      />

      {/* Feedback Detail Modal */}
      <Modal isOpen={isDetailOpen} onClose={onDetailClose} size="lg" isCentered>
        <ModalOverlay backdropFilter="blur(8px)" />
        <ModalContent
          borderRadius="24px"
          overflow="hidden"
          border="1px solid"
          borderColor={borderColor}
          boxShadow="2xl"
          bg={modalBg}
        >
          <ModalHeader
            fontSize="20px"
            fontWeight="800"
            color={textColor}
            pt="25px"
            px="30px"
            pb="15px"
          >
            {t('detail.title')}
          </ModalHeader>
          <ModalCloseButton top="20px" right="25px" borderRadius="12px" />
          <ModalBody px="30px" py="10px">
            <VStack align="stretch" spacing={4}>
              <Grid templateColumns="repeat(2, 1fr)" gap={4}>
                <GridItem colSpan={2}>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="4px">
                    {t('table.title')}
                  </Text>
                  <Text fontSize="md" fontWeight="600" color={textColor}>
                    {selectedFeedback?.title || '-'}
                  </Text>
                </GridItem>

                <GridItem>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="4px">
                    {t('table.source')}
                  </Text>
                  <Badge colorScheme={selectedFeedback?.source === 'email' ? 'purple' : 'teal'} borderRadius="8px" px="8px" py="2px">
                    {selectedFeedback ? t(`source.${selectedFeedback.source}`) : '-'}
                  </Badge>
                </GridItem>

                <GridItem>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="4px">
                    {t('table.category')}
                  </Text>
                  <Badge variant="outline" colorScheme="brand" borderRadius="8px" px="8px" py="2px">
                    {selectedFeedback ? t(`filter.resourceTypes.${selectedFeedback.category}`, { ns: 'modules/audit-logs', defaultValue: selectedFeedback.category || '-' }) : '-'}
                  </Badge>
                </GridItem>

                <GridItem colSpan={2}>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="4px">
                    {t('detail.email')}
                  </Text>
                  {selectedFeedback?.email ? (
                    <HStack>
                      <Text fontSize="sm" color={textColor} fontWeight="500">{selectedFeedback.email}</Text>
                      <IconButton
                        aria-label={t('actions.copy')}
                        icon={<CopyIcon />}
                        size="xs"
                        variant="ghost"
                        onClick={() => copyEmail(selectedFeedback.email)}
                      />
                    </HStack>
                  ) : (
                    <Text fontSize="sm" color="gray.400" fontStyle="italic">{t('detail.noEmail')}</Text>
                  )}
                </GridItem>

                <GridItem>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="4px">
                    {t('table.createdAt')}
                  </Text>
                  <Text fontSize="sm" color={textColor}>
                    {selectedFeedback ? formatDateTime(selectedFeedback.created_at) : '-'}
                  </Text>
                </GridItem>

                <GridItem>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="4px">
                    {t('detail.updatedAt')}
                  </Text>
                  <Text fontSize="sm" color={textColor}>
                    {selectedFeedback ? formatDateTime(selectedFeedback.updated_at) : '-'}
                  </Text>
                </GridItem>

                <GridItem colSpan={2}>
                  <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="6px">
                    {t('table.status')}
                  </Text>
                  <HStack>
                    <Badge colorScheme={selectedFeedback ? (statusColor[selectedFeedback.status] || 'gray') : 'gray'} mr="2">
                      {selectedFeedback ? t(`status.${selectedFeedback.status}`) : '-'}
                    </Badge>
                    {selectedFeedback && (
                      <Select
                        size="sm"
                        w="150px"
                        borderRadius="10px"
                        value={selectedFeedback.status}
                        onChange={async (e) => {
                          const nextStatus = e.target.value;
                          const updated = await changeStatus(selectedFeedback.id, nextStatus);
                          if (updated) {
                            setSelectedFeedback({ ...selectedFeedback, status: nextStatus });
                          }
                        }}
                      >
                        {feedbackStatuses.map(status => (
                          <option key={status} value={status}>{t(`status.${status}`)}</option>
                        ))}
                      </Select>
                    )}
                  </HStack>
                </GridItem>
              </Grid>

              <Divider py="2px" />

              <Box>
                <Text fontSize="xs" fontWeight="700" color="gray.500" textTransform="uppercase" mb="8px">
                  {t('table.content')}
                </Text>
                <Box
                  p="20px"
                  bg={markdownBg}
                  border="1px solid"
                  borderColor={borderColor}
                  borderRadius="16px"
                  maxHeight="320px"
                  overflowY="auto"
                  fontSize="sm"
                  color={textColor}
                  lineHeight="tall"
                  css={{
                    'h1, h2, h3, h4, h5, h6': {
                      fontWeight: 'bold',
                      marginTop: '16px',
                      marginBottom: '8px',
                    },
                    'h1': { fontSize: '1.4em' },
                    'h2': { fontSize: '1.3em' },
                    'h3': { fontSize: '1.2em' },
                    'p': { marginBottom: '10px' },
                    'ul, ol': { paddingLeft: '20px', marginBottom: '10px' },
                    'li': { marginBottom: '4px' },
                    'code': {
                      background: markdownCodeBg,
                      padding: '2px 6px',
                      borderRadius: '4px',
                      fontFamily: 'monospace',
                    },
                    'pre': {
                      background: markdownCodeBg,
                      padding: '12px',
                      borderRadius: '8px',
                      overflowX: 'auto',
                      marginBottom: '10px',
                      code: { padding: 0, background: 'none' },
                    },
                    'blockquote': {
                      borderLeft: '4px solid',
                      borderColor: 'var(--chakra-colors-brand-500)',
                      paddingLeft: '12px',
                      color: 'gray.500',
                      fontStyle: 'italic',
                      marginBottom: '10px',
                    },
                    'a': {
                      color: 'var(--chakra-colors-brand-500)',
                      textDecoration: 'underline',
                      '&:hover': { color: 'var(--chakra-colors-brand-600)' },
                    },
                  }}
                >
                  <ReactMarkdown
                    components={{
                      a: ({ children, ...props }) => (
                        <a {...props} target="_blank" rel="noopener noreferrer">
                          {children}
                        </a>
                      ),
                    }}
                  >
                    {selectedFeedback?.content || ''}
                  </ReactMarkdown>
                </Box>
              </Box>
            </VStack>
          </ModalBody>
          <ModalFooter px="30px" pt="15px" pb="25px">
            <Button borderRadius="16px" px="24px" colorScheme="brand" onClick={onDetailClose}>
              {tCommon('button.close', { defaultValue: 'Close' })}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
