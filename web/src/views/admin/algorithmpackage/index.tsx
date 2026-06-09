import {
  Box,
  Button,
  Flex,
  Text,
  useColorModeValue,
  IconButton,
  useToast,
  HStack,
  Center,
  Spinner,
  Progress,
  Badge,
  Grid,
  GridItem,
  VStack,
  Input,
  Tag,
  TagLabel,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Code,
  Tooltip,
} from '@chakra-ui/react';
import { DeleteIcon, WarningIcon, CheckCircleIcon, InfoIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback, useRef } from 'react';
import { useDropzone } from 'react-dropzone';
import { MdExtension, MdCloudUpload, MdSearch, MdContentCopy } from 'react-icons/md';

import { algorithmPackagesApi, type AlgorithmPackage } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { usePagination } from 'hooks/usePagination';
import { useFilter } from 'hooks/useFilter';

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatSchema(raw: string | null | undefined): string | null {
  if (!raw) return null;
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw; // 降级：展示原始字符串
  }
}

export default function AlgorithmPackages() {
  const { t } = useTranslation('modules/algorithmpackage');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const secondaryTextColor = useColorModeValue('gray.600', 'gray.400');
  const bgCard = useColorModeValue('white', 'navy.800');
  const bgUpload = useColorModeValue('gray.50', 'navy.900');
  const bgUploadHover = useColorModeValue('gray.100', 'navy.950');
  const bgCode = useColorModeValue('gray.50', 'navy.900');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const [uploading, setUploading] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const { filters, setFilter, searchTrigger, refresh } = useFilter();

  const fetchPackages = useCallback((p: number, ps: number) => {
    return algorithmPackagesApi.list({
      page: p,
      page_size: ps,
      keyword: filters.keyword,
    });
  }, [filters]);

  const {
    list: packages,
    total,
    page,
    pageSize,
    initialLoading,
    load,
    changePage,
    changePageSize
  } = usePagination<AlgorithmPackage>(fetchPackages);

  useEffect(() => {
    load({ page: 1 });
  }, [searchTrigger, load]);

  // Real-time polling when packages are in pending/running self-check
  useEffect(() => {
    const hasRunningCheck = packages.some(
      (pkg) => pkg.self_check_status === 'pending' || pkg.self_check_status === 'running'
    );
    if (!hasRunningCheck) return;

    const interval = setInterval(() => {
      refresh();
    }, 3000);

    return () => clearInterval(interval);
  }, [packages, refresh]);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(e.target.value);
  };

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      setFilter('keyword', searchQuery);
    }
  };

  const handleSearchClick = () => {
    setFilter('keyword', searchQuery);
  };

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    const file = acceptedFiles[0];
    if (!file) return;

    if (!file.name.endsWith('.zip')) {
      toast({
        title: t('message.uploadFailed'),
        description: t('uploadZone.onlyZip'),
        status: 'error',
      });
      return;
    }

    setUploading(true);
    try {
      await algorithmPackagesApi.upload(file);
      toast({ title: t('message.uploadSuccess'), status: 'success' });
      load({ page: 1 });
    } catch (err) {
      toast({
        title: t('message.uploadFailed'),
        description: err instanceof Error ? err.message : String(err || ''),
        status: 'error',
      });
    } finally {
      setUploading(false);
    }
  }, [t, load, toast]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    maxFiles: 1,
    disabled: uploading,
  });

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await algorithmPackagesApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      load();
    } catch (err) {
      toast({
        title: t('message.deleteFailed'),
        description: err instanceof Error ? err.message : String(err || ''),
        status: 'error',
      });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast({
      title: t('card.copied'),
      status: 'success',
      duration: 2000,
    });
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'passed':
        return <Badge colorScheme="green"><HStack spacing={1}><CheckCircleIcon w={3} h={3} /> <Text>{t('card.selfCheckSuccess')}</Text></HStack></Badge>;
      case 'failed':
        return <Badge colorScheme="red"><HStack spacing={1}><WarningIcon w={3} h={3} /> <Text>{t('card.selfCheckFailed')}</Text></HStack></Badge>;
      case 'running':
        return <Badge colorScheme="blue" className="pulse-animation"><HStack spacing={1}><Spinner size="xs" /> <Text>{t('card.selfCheckRunning')}</Text></HStack></Badge>;
      default:
        return <Badge colorScheme="gray"><HStack spacing={1}><InfoIcon w={3} h={3} /> <Text>{t('card.selfCheckPending')}</Text></HStack></Badge>;
    }
  };

  const stats = {
    total: total,
    passed: packages.filter(p => p.self_check_status === 'passed').length,
    failed: packages.filter(p => p.self_check_status === 'failed').length,
    active: packages.filter(p => p.status === 'active').length
  };

  if (initialLoading) {
    return (
      <Center h="400px">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }} px="20px">
      <Flex direction="column" mb="24px">
        <Text color={textColor} fontSize="2xl" fontWeight="700" mb="8px">
          {t('title')}
        </Text>

        {/* Stats Grid */}
        <Grid templateColumns={{ base: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }} gap="20px" mb="24px">
          <GridItem bg={bgCard} p="20px" borderRadius="16px" border="1px solid" borderColor={borderColor}>
            <Text fontSize="sm" color={secondaryTextColor}>{t('stats.total')}</Text>
            <Text fontSize="2xl" fontWeight="700" color={textColor}>{stats.total}</Text>
          </GridItem>
          <GridItem bg={bgCard} p="20px" borderRadius="16px" border="1px solid" borderColor={borderColor}>
            <Text fontSize="sm" color={secondaryTextColor}>{t('stats.passed')}</Text>
            <Text fontSize="2xl" fontWeight="700" color="green.500">{stats.passed}</Text>
          </GridItem>
          <GridItem bg={bgCard} p="20px" borderRadius="16px" border="1px solid" borderColor={borderColor}>
            <Text fontSize="sm" color={secondaryTextColor}>{t('stats.failed')}</Text>
            <Text fontSize="2xl" fontWeight="700" color="red.500">{stats.failed}</Text>
          </GridItem>
          <GridItem bg={bgCard} p="20px" borderRadius="16px" border="1px solid" borderColor={borderColor}>
            <Text fontSize="sm" color={secondaryTextColor}>{t('stats.active')}</Text>
            <Text fontSize="2xl" fontWeight="700" color="blue.500">{stats.active}</Text>
          </GridItem>
        </Grid>

        {/* Upload Zone */}
        <Box
          {...getRootProps()}
          bg={bgUpload}
          border="2px dashed"
          borderColor={isDragActive ? 'brand.500' : borderColor}
          borderRadius="20px"
          p="40px"
          cursor={uploading ? 'not-allowed' : 'pointer'}
          transition="all 0.2s"
          _hover={{ borderColor: 'brand.500', bg: bgUploadHover }}
          mb="24px"
          textAlign="center"
        >
          <input {...getInputProps()} />
          <VStack spacing={3}>
            {uploading ? (
              <>
                <Spinner size="lg" color="brand.500" />
                <Text fontWeight="600" color={textColor}>{t('uploading')}</Text>
                <Progress size="xs" isIndeterminate w="200px" colorScheme="brand" borderRadius="4px" />
              </>
            ) : (
              <>
                <Box color="brand.500" fontSize="48px">
                  <MdCloudUpload />
                </Box>
                <Text fontWeight="600" fontSize="lg" color={textColor}>
                  {isDragActive ? t('uploadZone.dropHere') : t('uploadZone.title')}
                </Text>
                <Text fontSize="sm" color={secondaryTextColor}>
                  {t('uploadZone.hint')}
                </Text>
              </>
            )}
          </VStack>
        </Box>

        {/* Search Bar */}
        <Flex mb="20px" align="center" gap="10px">
          <Input
            value={searchQuery}
            onChange={handleSearchChange}
            onKeyDown={handleSearchKeyDown}
            placeholder={t('search.placeholder')}
            bg={bgCard}
            borderColor={borderColor}
            _hover={{ borderColor: 'brand.500' }}
            _focus={{ borderColor: 'brand.500' }}
            maxW="400px"
          />
          <IconButton
            onClick={handleSearchClick}
            aria-label="Search"
            icon={<MdSearch />}
            colorScheme="brand"
          />
        </Flex>

        {/* Card Grid List */}
        {packages.length === 0 ? (
          <Center bg={bgCard} borderRadius="16px" p="60px" border="1px solid" borderColor={borderColor}>
            <VStack spacing={4}>
              <Box fontSize="48px" color="gray.400"><MdExtension /></Box>
              <Text color={secondaryTextColor} fontSize="lg">{t('card.empty')}</Text>
            </VStack>
          </Center>
        ) : (
          <VStack spacing="20px" align="stretch">
            {packages.map((pkg) => (
              <Box
                key={pkg.id}
                bg={bgCard}
                p="24px"
                borderRadius="16px"
                border="1px solid"
                borderColor={borderColor}
                boxShadow="sm"
                transition="transform 0.2s, box-shadow 0.2s"
                _hover={{ transform: 'translateY(-2px)', boxShadow: 'md' }}
              >
                <Flex direction={{ base: 'column', md: 'row' }} justify="space-between" align={{ base: 'stretch', md: 'center' }} gap="16px">
                  <HStack spacing="16px" align="center">
                    <Box bg="brand.500" color="white" p="12px" borderRadius="12px" fontSize="24px">
                      <MdExtension />
                    </Box>
                    <VStack align="start" spacing="4px">
                      <HStack spacing="8px">
                        <Text fontSize="lg" fontWeight="700" color={textColor}>
                          {pkg.algorithm_alias || pkg.algorithm_name}
                        </Text>
                        <Badge variant="subtle" colorScheme="blue">v{pkg.version}</Badge>
                      </HStack>
                      <Text fontSize="sm" color={secondaryTextColor}>{pkg.description || 'No description provided'}</Text>
                      <HStack spacing="8px" flexWrap="wrap">
                        <Tag size="sm" variant="subtle" colorScheme="purple">
                          <TagLabel>{t('card.domain')}: {pkg.domain || 'general'}</TagLabel>
                        </Tag>
                        {pkg.hardware?.map(hw => (
                          <Tag key={hw} size="sm" variant="solid" colorScheme="teal">
                            <TagLabel>{hw}</TagLabel>
                          </Tag>
                        ))}
                      </HStack>
                    </VStack>
                  </HStack>

                  <Flex align="center" gap="16px" justify={{ base: 'space-between', md: 'flex-end' }}>
                    <VStack align={{ base: 'start', md: 'end' }} spacing="4px">
                      <Text fontSize="xs" color={secondaryTextColor}>{t('card.checkStatus')}</Text>
                      {getStatusBadge(pkg.self_check_status)}
                    </VStack>

                    <IconButton
                      onClick={() => setDeleteTarget(pkg.id)}
                      aria-label="Delete"
                      icon={<DeleteIcon />}
                      colorScheme="red"
                      variant="ghost"
                    />
                  </Flex>
                </Flex>

                {/* Details Accordion */}
                <Accordion allowToggle mt="20px" borderTop="1px solid" borderColor={borderColor}>
                  <AccordionItem border="none">
                    <h2>
                      <AccordionButton px="0" _hover={{ bg: 'transparent' }}>
                        <Box as="span" flex="1" textAlign="start" fontSize="sm" fontWeight="600" color="brand.500">
                          {t('card.details')}
                        </Box>
                        <AccordionIcon color="brand.500" />
                      </AccordionButton>
                    </h2>
                    <AccordionPanel pb="4px" px="0" pt="12px">
                      <Grid templateColumns={{ base: '1fr', md: '1fr 1fr' }} gap="20px" mb="16px">
                        <VStack align="stretch" spacing="10px">
                          <HStack justify="space-between">
                            <Text fontSize="xs" color={secondaryTextColor}>{t('card.packageSize')}</Text>
                            <Text fontSize="xs" fontWeight="600" color={textColor}>{formatSize(pkg.package_size)}</Text>
                          </HStack>
                          <HStack justify="space-between">
                            <Text fontSize="xs" color={secondaryTextColor}>{t('card.md5')}</Text>
                            <HStack spacing="4px">
                              <Text fontSize="xs" fontFamily="mono" color={textColor}>{pkg.package_md5?.substring(0, 16)}...</Text>
                              <Tooltip label="Copy MD5">
                                <IconButton
                                  size="xs"
                                  icon={<MdContentCopy />}
                                  aria-label="Copy MD5"
                                  variant="ghost"
                                  onClick={() => copyToClipboard(pkg.package_md5)}
                                />
                              </Tooltip>
                            </HStack>
                          </HStack>
                          {pkg.self_check_at && (
                            <HStack justify="space-between">
                              <Text fontSize="xs" color={secondaryTextColor}>自检时间</Text>
                              <Text fontSize="xs" fontWeight="600" color={textColor}>{formatDateTime(pkg.self_check_at)}</Text>
                            </HStack>
                          )}
                        </VStack>

                        <VStack align="stretch" spacing="10px">
                          <Text fontSize="xs" color={secondaryTextColor}>{t('card.capabilities')}</Text>
                          <HStack spacing="6px" flexWrap="wrap">
                            {pkg.capabilities_image?.map(cap => (
                              <Tag key={cap} size="sm" colorScheme="blue" variant="outline"><TagLabel>{cap}</TagLabel></Tag>
                            ))}
                            {pkg.capabilities_data?.map(cap => (
                              <Tag key={cap} size="sm" colorScheme="green" variant="outline"><TagLabel>{cap}</TagLabel></Tag>
                            ))}
                            {(!pkg.capabilities_image?.length && !pkg.capabilities_data?.length) && <Text fontSize="xs" color="gray.400">None</Text>}
                          </HStack>
                        </VStack>
                      </Grid>

                      {/* Self Check Failure Alert */}
                      {pkg.self_check_status === 'failed' && pkg.self_check_result && (
                        <Box bg="red.50" _dark={{ bg: 'red.950' }} p="16px" borderRadius="8px" borderLeft="4px solid" borderColor="red.500" mb="16px">
                          <Text fontSize="sm" fontWeight="700" color="red.500" mb="4px">{t('card.errorMsg')}</Text>
                          <Text fontSize="xs" color="red.600" _dark={{ color: 'red.300' }}>
                            {typeof pkg.self_check_result === 'string' 
                              ? pkg.self_check_result 
                              : JSON.stringify(pkg.self_check_result, null, 2)}
                          </Text>
                        </Box>
                      )}

                      {/* Schema Code Blocks */}
                      <Grid templateColumns={{ base: '1fr', lg: '1fr 1fr' }} gap="20px">
                        <Box>
                          <Flex justify="space-between" align="center" mb="6px">
                            <Text fontSize="xs" fontWeight="600" color={secondaryTextColor}>{t('card.schema')}</Text>
                            {pkg.result_schema && (
                              <Button size="xs" leftIcon={<MdContentCopy />} onClick={() => copyToClipboard(pkg.result_schema)}>Copy</Button>
                            )}
                          </Flex>
                          <Code p="10px" borderRadius="8px" fontSize="xs" w="100%" maxH="200px" overflowY="auto" display="block" whiteSpace="pre-wrap" bg={bgCode}>
                            {formatSchema(pkg.result_schema) || t('card.noSchema')}
                          </Code>
                        </Box>
                        <Box>
                          <Flex justify="space-between" align="center" mb="6px">
                            <Text fontSize="xs" fontWeight="600" color={secondaryTextColor}>{t('card.paramsSchema')}</Text>
                            {pkg.ai_params_schema && (
                              <Button size="xs" leftIcon={<MdContentCopy />} onClick={() => copyToClipboard(JSON.stringify(pkg.ai_params_schema))}>Copy</Button>
                            )}
                          </Flex>
                          <Code p="10px" borderRadius="8px" fontSize="xs" w="100%" maxH="200px" overflowY="auto" display="block" whiteSpace="pre-wrap" bg={bgCode}>
                            {pkg.ai_params_schema ? JSON.stringify(pkg.ai_params_schema, null, 2) : t('card.noSchema')}
                          </Code>
                        </Box>
                      </Grid>
                    </AccordionPanel>
                  </AccordionItem>
                </Accordion>
              </Box>
            ))}
          </VStack>
        )}

        {/* Pagination */}
        {total > 0 && (
          <Box mt="24px">
            <Pagination
              page={page}
              pageSize={pageSize}
              total={total}
              onChange={changePage}
              onPageSizeChange={changePageSize}
            />
          </Box>
        )}
      </Flex>

      {/* Confirm Delete Dialog */}
      <ConfirmDialog
        isOpen={!!deleteTarget}
        title={t('card.delete')}
        message={t('card.deleteConfirm')}
        isLoading={isDeleting}
        onConfirm={handleDelete}
        onClose={() => setDeleteTarget(null)}
      />
    </Box>
  );
}
