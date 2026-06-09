import {
  Box,
  Button,
  Flex,
  Text,
  useColorModeValue,
  useToast,
  Spinner,
  Center,
  Badge,
  HStack,
  VStack,
  IconButton,
  Card,
  CardBody,
  CardHeader,
  Heading,
  Divider,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  SimpleGrid,
  Code,
  Tooltip,
  Input,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
} from '@chakra-ui/react';
import { CopyIcon, CheckIcon, AttachmentIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useCallback, useRef } from 'react';
import { licenseApi, type FingerprintResponse, type LicenseInfo } from 'services/api';
import { useDateFormat } from 'hooks/useDateFormat';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { usePagination } from 'hooks/usePagination';

const LICENSE_STATUS_COLOR: Record<string, string> = {
  active: 'green',
  expired: 'red',
  revoked: 'gray',
  not_yet_valid: 'yellow',
};

const LICENSE_TYPE_COLOR: Record<string, string> = {
  trial: 'yellow',
  formal: 'blue',
  permanent: 'green',
};

const MAX_LICENSE_FILE_SIZE = 1024 * 1024; // 1MB

export default function License() {
  const { t } = useTranslation('modules/license');
  const { t: tCommon } = useTranslation('common');
  const { formatDateTime } = useDateFormat();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const bgStat = useColorModeValue('gray.50', 'whiteAlpha.100');
  const toast = useToast();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [fingerprint, setFingerprint] = useState<FingerprintResponse | null>(null);
  const [activeLicense, setActiveLicense] = useState<LicenseInfo | null>(null);
  const [fingerprintLoading, setFingerprintLoading] = useState(true);
  const [licenseLoading, setLicenseLoading] = useState(true);
  const [uploading, setUploading] = useState(false);
  const [copiedField, setCopiedField] = useState<string | null>(null);

  // License list
  const fetchLicenses = useCallback((p: number, ps: number) =>
    licenseApi.list({ page: p, page_size: ps }),
  []);

  const {
    list: licenses,
    total,
    page,
    pageSize,
    initialLoading: listLoading,
    pageLoading,
    load: loadList,
    changePage,
    changePageSize,
  } = usePagination<LicenseInfo>(fetchLicenses);

  // Load fingerprint
  useEffect(() => {
    setFingerprintLoading(true);
    licenseApi.getFingerprint()
      .then(setFingerprint)
      .catch(() => {
        toast({ title: t('fingerprint.fetchError'), status: 'error' });
      })
      .finally(() => setFingerprintLoading(false));
  }, [t, toast]);

  // Load active license
  useEffect(() => {
    setLicenseLoading(true);
    licenseApi.getActive()
      .then(setActiveLicense)
      .catch(() => setActiveLicense(null))
      .finally(() => setLicenseLoading(false));
  }, []);

  // Load license list
  useEffect(() => {
    loadList({ page: 1 });
  }, [loadList]);

  const copyToClipboard = async (text: string, field: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedField(field);
      toast({ title: t('fingerprint.copied'), status: 'success' });
      setTimeout(() => setCopiedField(null), 2000);
    } catch {
      toast({ title: t('fingerprint.copyFailed'), status: 'error' });
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (file.size > MAX_LICENSE_FILE_SIZE) {
      toast({ title: t('upload.fileTooLarge'), status: 'warning' });
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
      return;
    }

    setUploading(true);
    try {
      const result = await licenseApi.upload(file);
      setActiveLicense(result);
      toast({ title: t('upload.success'), status: 'success' });
      loadList({ page: 1 });
    } catch (err) {
      toast({
        title: t('upload.failed'),
        description: err instanceof Error ? err.message : String(err || ''),
        status: 'error',
      });
    } finally {
      setUploading(false);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const formatAlgorithms = (algos: string[] | undefined) => {
    if (!algos || algos.length === 0) return t('info.none');
    if (algos.includes('*')) return '*';
    return algos.join(', ');
  };

  const formatFeatures = (features: string[] | undefined) => {
    if (!features || features.length === 0) return t('info.none');
    return features.join(', ');
  };

  const getLicenseTypeLabel = (type: string): string => {
    const map: Record<string, string> = {
      trial: t('type.trial'),
      formal: t('type.formal'),
      permanent: t('type.permanent'),
    };
    return map[type] || type;
  };

  const getLicenseStatusLabel = (status: string): string => {
    const map: Record<string, string> = {
      active: t('status.active'),
      expired: t('status.expired'),
      revoked: t('status.revoked'),
      not_yet_valid: t('status.not_yet_valid'),
      none: t('status.none'),
    };
    return map[status] || status;
  };

  const getExpireActionLabel = (action: string): string => {
    const map: Record<string, string> = {
      terminate: t('expireAction.terminate'),
      keep_running: t('expireAction.keep_running'),
    };
    return map[action] || action;
  };

  if (fingerprintLoading || licenseLoading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Box>
          <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
          <Text fontSize="sm" color="gray.500">{t('description')}</Text>
        </Box>
      </Flex>

      {/* 设备指纹卡片 */}
      <Card bg={bgCard} border="1px solid" borderColor={borderColor} mb={6}>
        <CardHeader>
          <Heading size="md" color={textColor}>{t('fingerprint.title')}</Heading>
        </CardHeader>
        <CardBody>
          {fingerprint ? (
            <SimpleGrid columns={{ base: 1, md: 3 }} spacing={4}>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('fingerprint.deviceSN')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>
                  {fingerprint.device_sn || '-'}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('fingerprint.fingerprint')}</StatLabel>
                <HStack>
                  <Code fontSize="sm" maxW="250px" isTruncated>{fingerprint.fingerprint}</Code>
                  <Tooltip label={copiedField === 'fp' ? t('fingerprint.copied') : t('fingerprint.copyFingerprint')}>
                    <IconButton
                      aria-label="Copy fingerprint"
                      icon={copiedField === 'fp' ? <CheckIcon /> : <CopyIcon />}
                      size="sm"
                      variant="ghost"
                      colorScheme={copiedField === 'fp' ? 'green' : 'gray'}
                      onClick={() => copyToClipboard(fingerprint.fingerprint, 'fp')}
                    />
                  </Tooltip>
                </HStack>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('fingerprint.hashAlgorithm')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>{fingerprint.hash_algorithm}</StatNumber>
              </Stat>
            </SimpleGrid>
          ) : (
            <Text color="gray.500">{t('fingerprint.noFingerprint')}</Text>
          )}
        </CardBody>
      </Card>

      {/* 当前授权信息 */}
      <Card bg={bgCard} border="1px solid" borderColor={borderColor} mb={6}>
        <CardHeader>
          <Flex justify="space-between" align="center">
            <Heading size="md" color={textColor}>{t('info.title')}</Heading>
            <Button
              leftIcon={<AttachmentIcon />}
              colorScheme="brand"
              variant="outline"
              onClick={() => fileInputRef.current?.click()}
              isLoading={uploading}
            >
              {t('upload.title')}
            </Button>
            <Input
              ref={fileInputRef}
              type="file"
              accept=".lic"
              display="none"
              onChange={handleFileUpload}
            />
          </Flex>
        </CardHeader>
        <CardBody>
          {licenseLoading ? (
            <Center h="100px"><Spinner /></Center>
          ) : activeLicense ? (
            <SimpleGrid columns={{ base: 1, md: 2, lg: 4 }} spacing={4}>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.licenseID')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>{activeLicense.license_id}</StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.licenseType')}</StatLabel>
                <StatNumber fontSize="md">
                  <Badge colorScheme={LICENSE_TYPE_COLOR[activeLicense.license_type] || 'gray'}>
                    {getLicenseTypeLabel(activeLicense.license_type)}
                  </Badge>
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.status')}</StatLabel>
                <StatNumber fontSize="md">
                  <Badge colorScheme={LICENSE_STATUS_COLOR[activeLicense.status] || 'gray'}>
                    {getLicenseStatusLabel(activeLicense.status)}
                  </Badge>
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.remainingDays')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>
                  {activeLicense.remaining_days !== null
                    ? `${activeLicense.remaining_days} ${t('info.days')}`
                    : t('info.permanent')}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.algorithms')}</StatLabel>
                <StatNumber fontSize="sm" color={textColor}>
                  {formatAlgorithms(activeLicense.algorithms)}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.maxStreams')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>
                  {activeLicense.max_streams > 0 ? activeLicense.max_streams : t('info.unlimited')}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.notBefore')}</StatLabel>
                <StatNumber fontSize="sm" color={textColor}>
                  {formatDateTime(activeLicense.not_before)}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.notAfter')}</StatLabel>
                <StatNumber fontSize="sm" color={textColor}>
                  {activeLicense.not_after ? formatDateTime(activeLicense.not_after) : t('info.permanent')}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.expireAction')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>
                  {getExpireActionLabel(activeLicense.expire_action)}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.features')}</StatLabel>
                <StatNumber fontSize="sm" color={textColor}>
                  {formatFeatures(activeLicense.features)}
                </StatNumber>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.deviceFingerprint')}</StatLabel>
                <HStack>
                  <Code fontSize="xs" maxW="200px" isTruncated>{activeLicense.device_fingerprint || '-'}</Code>
                  {activeLicense.device_fingerprint && (
                    <Tooltip label={copiedField === 'licFp' ? t('fingerprint.copied') : t('fingerprint.copyFingerprint')}>
                      <IconButton
                        aria-label="Copy"
                        icon={copiedField === 'licFp' ? <CheckIcon /> : <CopyIcon />}
                        size="xs"
                        variant="ghost"
                        onClick={() => copyToClipboard(activeLicense.device_fingerprint, 'licFp')}
                      />
                    </Tooltip>
                  )}
                </HStack>
              </Stat>
              <Stat bg={bgStat} p={4} borderRadius="12px">
                <StatLabel color="gray.500">{t('info.deviceSN')}</StatLabel>
                <StatNumber fontSize="md" color={textColor}>{activeLicense.device_sn || '-'}</StatNumber>
              </Stat>
            </SimpleGrid>
          ) : (
            <Center py={8}>
              <VStack spacing={4}>
                <Text color="gray.500" fontSize="lg">{t('message.noActiveLicense')}</Text>
                <Button
                  leftIcon={<AttachmentIcon />}
                  colorScheme="brand"
                  onClick={() => fileInputRef.current?.click()}
                >
                  {t('upload.title')}
                </Button>
              </VStack>
            </Center>
          )}
        </CardBody>
      </Card>

      {/* 授权历史列表 */}
      <Card bg={bgCard} border="1px solid" borderColor={borderColor}>
        <CardHeader>
          <Heading size="md" color={textColor}>{t('title')}</Heading>
        </CardHeader>
        <CardBody p={0}>
          {listLoading ? (
            <Center h="200px"><Spinner /></Center>
          ) : (
            <Box overflow="auto">
              <Table variant="simple" size="md" minW="900px">
                <Thead>
                  <Tr>
                    <Th>{t('table.columns.licenseID')}</Th>
                    <Th>{t('table.columns.licenseType')}</Th>
                    <Th>{t('table.columns.deviceSN')}</Th>
                    <Th>{t('table.columns.algorithms')}</Th>
                    <Th>{t('table.columns.maxStreams')}</Th>
                    <Th>{t('table.columns.status')}</Th>
                    <Th>{t('table.columns.notBefore')}</Th>
                    <Th>{t('table.columns.notAfter')}</Th>
                    <Th>{t('table.columns.remainingDays')}</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {(licenses || []).map((lic) => (
                    <Tr key={lic.id}>
                      <Td fontWeight="600" fontSize="sm">{lic.license_id}</Td>
                      <Td>
                        <Badge colorScheme={LICENSE_TYPE_COLOR[lic.license_type] || 'gray'}>
                          {getLicenseTypeLabel(lic.license_type)}
                        </Badge>
                      </Td>
                      <Td fontSize="sm">{lic.device_sn || '-'}</Td>
                      <Td fontSize="sm" maxW="200px" isTruncated>{formatAlgorithms(lic.algorithms)}</Td>
                      <Td>{lic.max_streams > 0 ? lic.max_streams : t('info.unlimited')}</Td>
                      <Td>
                        <Badge colorScheme={LICENSE_STATUS_COLOR[lic.status] || 'gray'}>
                          {getLicenseStatusLabel(lic.status)}
                        </Badge>
                      </Td>
                      <Td whiteSpace="nowrap" fontSize="sm">{formatDateTime(lic.not_before)}</Td>
                      <Td whiteSpace="nowrap" fontSize="sm">{lic.not_after ? formatDateTime(lic.not_after) : t('info.permanent')}</Td>
                      <Td>
                        {lic.remaining_days !== null
                          ? `${lic.remaining_days} ${t('info.days')}`
                          : t('info.permanent')}
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
          )}
        </CardBody>
      </Card>
    </Box>
  );
}
