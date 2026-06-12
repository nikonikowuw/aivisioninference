import { useState, useRef, useCallback } from 'react';
import { MdSearch, MdImage, MdFace } from 'react-icons/md';
import {
  Box,
  Button,
  Container,
  Flex,
  Heading,
  HStack,
  Icon,
  Stack,
  Text,
  VStack,
  Divider,
  Image,
  Badge,
  SimpleGrid,
  Spinner,
  Center,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { personsApi, getFileUrl, type FaceSearchResultItem } from 'services/api';

export default function FaceSearchPage() {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation();
  const toast = useToast();

  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const textMuted = useColorModeValue('gray.500', 'gray.400');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const dashedBorderColor = useColorModeValue('gray.300', 'whiteAlpha.300');
  const uploadBg = useColorModeValue('gray.50', 'navy.700');
  const uploadHoverBg = useColorModeValue('blue.50', 'navy.600');
  const resultsBg = useColorModeValue('gray.50', 'navy.700');
  const cardBg = useColorModeValue('white', 'navy.800');
  const cardBorder = useColorModeValue('gray.200', 'whiteAlpha.200');

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [results, setResults] = useState<FaceSearchResultItem[] | null>(null);
  const [isSearching, setIsSearching] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;

      const ext = file.name.split('.').pop()?.toLowerCase();
      if (!ext || !['jpg', 'jpeg', 'png', 'webp'].includes(ext)) {
        toast({
          title: tCommon('message.operationFailed'),
          description: t('faceSearch.searchHint'),
          status: 'error',
        });
        return;
      }

      setSelectedFile(file);
      setResults(null);
      const reader = new FileReader();
      reader.onload = () => setPreviewUrl(reader.result as string);
      reader.readAsDataURL(file);
    },
    [toast, t, tCommon],
  );

  const triggerFileInput = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleSearch = useCallback(async () => {
    if (!selectedFile || isSearching) return;
    setIsSearching(true);
    try {
      const formData = new FormData();
      formData.append('image', selectedFile);
      formData.append('top_k', '5');
      formData.append('threshold', '0.7');
      const data = await personsApi.searchByFace(formData);
      setResults(data);
    } catch (err: unknown) {
      toast({
        title: tCommon('message.operationFailed'),
        description: err instanceof Error ? err.message : '',
        status: 'error',
      });
    } finally {
      setIsSearching(false);
    }
  }, [selectedFile, isSearching, toast, tCommon]);

  return (
    <Container maxW="container.xl" py={6}>
      <VStack align="stretch" spacing={2} mb={6}>
        <Heading as="h1" size="lg" color={textColor}>
          {t('faceSearch.title')}
        </Heading>
        <Text color={textMuted} fontSize="md">
          {t('faceSearch.description')}
        </Text>
      </VStack>

      <Stack direction={{ base: 'column', lg: 'row' }} spacing={6} align="stretch">
        {/* Left: Upload area */}
        <Box
          flex={1}
          bg={bgCard}
          borderWidth="1px"
          borderColor={borderColor}
          borderRadius="xl"
          p={6}
        >
          <VStack spacing={4} align="stretch">
            <Heading as="h3" size="sm" color={textColor}>
              {t('faceSearch.uploadTitle')}
            </Heading>

            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              style={{ display: 'none' }}
              onChange={handleFileSelect}
            />

            <Flex
              direction="column"
              align="center"
              justify="center"
              h="240px"
              borderWidth="2px"
              borderStyle="dashed"
              borderColor={previewUrl ? 'blue.400' : dashedBorderColor}
              borderRadius="lg"
              bg={previewUrl ? 'transparent' : uploadBg}
              _hover={{
                borderColor: 'blue.400',
                bg: previewUrl ? 'transparent' : uploadHoverBg,
              }}
              transition="all 0.2s"
              cursor="pointer"
              onClick={triggerFileInput}
              overflow="hidden"
              position="relative"
            >
              {previewUrl ? (
                <>
                  <Image
                    src={previewUrl}
                    alt="preview"
                    objectFit="contain"
                    maxH="100%"
                    maxW="100%"
                  />
                  <Box
                    position="absolute"
                    bottom={0}
                    left={0}
                    right={0}
                    bg="blackAlpha.600"
                    py={1}
                  >
                    <Text fontSize="xs" textAlign="center" color="white">
                      {t('faceSearch.replaceImage')}
                    </Text>
                  </Box>
                </>
              ) : (
                <>
                  <Icon as={MdImage} boxSize={12} color={textMuted} mb={3} />
                  <Text
                    color={textMuted}
                    fontSize="sm"
                    textAlign="center"
                    px={4}
                  >
                    {t('faceSearch.uploadPlaceholder')}
                  </Text>
                </>
              )}
            </Flex>

            <Button
              leftIcon={
                isSearching ? (
                  <Spinner size="sm" />
                ) : (
                  <Icon as={MdSearch} />
                )
              }
              colorScheme="blue"
              size="lg"
              w="full"
              onClick={handleSearch}
              isLoading={isSearching}
              loadingText={t('faceSearch.searching')}
              isDisabled={!selectedFile || isSearching}
            >
              {t('faceSearch.searchButton')}
            </Button>
            <Text color={textMuted} fontSize="xs" textAlign="center">
              {t('faceSearch.searchHint')}
            </Text>
          </VStack>
        </Box>

        {/* Right: Results */}
        <Box
          flex={2}
          bg={bgCard}
          borderWidth="1px"
          borderColor={borderColor}
          borderRadius="xl"
          p={6}
        >
          <VStack spacing={4} align="stretch">
            <Heading as="h3" size="sm" color={textColor}>
              {t('faceSearch.resultsTitle')}
              {results !== null && results.length > 0 && (
                <Text
                  as="span"
                  fontSize="sm"
                  color={textMuted}
                  fontWeight="normal"
                  ml={2}
                >
                  ({t('faceSearch.resultsCount', { count: results.length })})
                </Text>
              )}
            </Heading>
            <Divider />

            {results === null ? (
              <Flex
                direction="column"
                align="center"
                justify="center"
                h="300px"
                bg={resultsBg}
                borderRadius="lg"
              >
                <Icon as={MdFace} boxSize={16} color={textMuted} mb={3} />
                <Text color={textMuted} fontSize="md" textAlign="center" px={4}>
                  {t('faceSearch.resultsPlaceholder')}
                </Text>
              </Flex>
            ) : results.length === 0 ? (
              <Flex
                direction="column"
                align="center"
                justify="center"
                h="300px"
                bg={resultsBg}
                borderRadius="lg"
              >
                <Icon as={MdFace} boxSize={16} color={textMuted} mb={3} />
                <Text color={textMuted} fontSize="md" textAlign="center" px={4}>
                  {t('faceSearch.noResults')}
                </Text>
              </Flex>
            ) : (
              <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
                {results.map((item) => (
                  <Box
                    key={item.person.id}
                    bg={cardBg}
                    borderWidth="1px"
                    borderColor={cardBorder}
                    borderRadius="lg"
                    overflow="hidden"
                  >
                    <Flex p={3} gap={3}>
                      <Box flexShrink={0}>
                        <Image
                          src={getFileUrl(item.person.image_url)}
                          alt={item.person.person_name}
                          boxSize="80px"
                          objectFit="cover"
                          borderRadius="md"
                          fallback={
                            <Center
                              boxSize="80px"
                              bg={resultsBg}
                              borderRadius="md"
                            >
                              <Icon as={MdFace} boxSize={8} color={textMuted} />
                            </Center>
                          }
                        />
                      </Box>
                      <Box flex={1} minW={0}>
                        <Text
                          fontWeight="bold"
                          color={textColor}
                          fontSize="sm"
                          noOfLines={1}
                        >
                          {item.person.person_name}
                        </Text>
                        <Text color={textMuted} fontSize="xs" mt={0.5}>
                          {t('table.columns.personCode')}:{' '}
                          {item.person.person_code || '-'}
                        </Text>
                        {item.person.groups &&
                          item.person.groups.length > 0 && (
                            <HStack mt={1} flexWrap="wrap" spacing={1}>
                              {item.person.groups
                                .slice(0, 2)
                                .map((g) => (
                                  <Badge
                                    key={g.id}
                                    variant="subtle"
                                    colorScheme="blue"
                                    fontSize="xs"
                                  >
                                    {g.group_name}
                                  </Badge>
                                ))}
                              {item.person.groups.length > 2 && (
                                <Badge variant="subtle" fontSize="xs">
                                  +{item.person.groups.length - 2}
                                </Badge>
                              )}
                            </HStack>
                          )}
                        <Text
                          color="green.500"
                          fontWeight="semibold"
                          fontSize="sm"
                          mt={1}
                        >
                          {t('faceSearch.similarity')}:{' '}
                          {(item.similarity * 100).toFixed(1)}%
                        </Text>
                      </Box>
                    </Flex>
                  </Box>
                ))}
              </SimpleGrid>
            )}
          </VStack>
        </Box>
      </Stack>
    </Container>
  );
}
