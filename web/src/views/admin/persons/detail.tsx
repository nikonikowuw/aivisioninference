import { ChevronLeftIcon } from "@chakra-ui/icons";
import {
  Badge,
  Box,
  Button,
  Center,
  Flex,
  HStack,
  Image,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalOverlay,
  SimpleGrid,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react";
import Card from "components/card/Card";
import { useAuth } from "contexts/AuthContext";
import { useDateFormat } from "hooks/useDateFormat";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";
import {
  getFileUrl,
  personsApi,
  smartRecordsApi,
  type Person,
  type SmartRecord,
} from "services/api";
import { hasPermission } from "utils/permission";

const statusColorMap: Record<string, string> = {
  pending: "yellow",
  extracting: "blue",
  active: "green",
  failed: "red",
  disabled: "gray",
};

export default function PersonDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation("modules/persons");
  const { t: tCommon } = useTranslation("common");
  const { formatDateTime } = useDateFormat();
  const { user } = useAuth();

  const textColor = useColorModeValue("secondaryGray.900", "white");
  const textColorSecondary = "gray.400";
  const borderColor = useColorModeValue("gray.200", "whiteAlpha.100");

  const [person, setPerson] = useState<Person | null>(null);
  const [loading, setLoading] = useState(true);

  const [smartRecords, setSmartRecords] = useState<SmartRecord[]>([]);
  const [loadingTabs, setLoadingTabs] = useState(false);
  const [previewImage, setPreviewImage] = useState<string | null>(null);

  const canViewSensitive = hasPermission(
    user?.permission_codes || [],
    "person:sensitive:view",
  );

  useEffect(() => {
    if (id) {
      loadPerson(id);
    }
  }, [id]);

  // Esc 键退出详情页（仅在图片预览 Modal 未打开时生效）
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape" && !previewImage) {
        navigate("/admin/persons");
      }
    },
    [navigate, previewImage],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  const loadPerson = async (personId: string) => {
    try {
      setLoading(true);
      const data = await personsApi.get(personId);
      setPerson(data);
      loadRecordsData(data);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const loadRecordsData = async (p: Person) => {
    setLoadingTabs(true);
    try {
      const recordsRes = await smartRecordsApi.list({
        person_name: p.person_name,
        page: 1,
        page_size: 20,
      });
      setSmartRecords((recordsRes as any).data?.list || []);
    } catch (e) {
      console.error("Failed to load records data", e);
    } finally {
      setLoadingTabs(false);
    }
  };

  if (loading) {
    return (
      <Center h="50vh">
        <Spinner size="xl" color="brand.500" />
      </Center>
    );
  }

  if (!person) {
    return (
      <Center h="50vh">
        <Text>{tCommon("message.notFound")}</Text>
      </Center>
    );
  }

  return (
    <Box pt={{ base: "130px", md: "80px", xl: "80px" }}>
      <Flex direction="column" mb="20px">
        <HStack mb="20px" spacing={4}>
          <Button
            leftIcon={<ChevronLeftIcon />}
            variant="ghost"
            onClick={() => navigate("/admin/persons")}
          >
            {tCommon("button.back")}
          </Button>
          <Text color={textColor} fontSize="2xl" fontWeight="bold">
            {t("detail.title")}
          </Text>
        </HStack>

        <Card px="24px" py="24px" mb="20px">
          <Flex
            direction={{ base: "column", md: "row" }}
            align="start"
            gap="30px"
          >
            <VStack spacing={3} align="center" minW="180px">
              <Image
                src={getFileUrl(person.image_url)}
                boxSize="150px"
                objectFit="cover"
                borderRadius="lg"
                fallbackSrc="/img/placeholder-avatar.png"
                cursor="pointer"
                onClick={() => setPreviewImage(getFileUrl(person.image_url))}
              />
              <Text color={textColor} fontSize="xl" fontWeight="bold">
                {person.person_name}
              </Text>
              <HStack>
                <Badge
                  colorScheme={
                    statusColorMap[person.embedding_status] || "gray"
                  }
                  borderRadius="full"
                  px="3"
                  py="1"
                >
                  {t(`embedding.status.${person.embedding_status}`)}
                </Badge>
                {person.enabled ? (
                  <Badge colorScheme="green" borderRadius="full" px="3" py="1">
                    {t("status.enabled")}
                  </Badge>
                ) : (
                  <Badge colorScheme="red" borderRadius="full" px="3" py="1">
                    {t("status.disabled")}
                  </Badge>
                )}
              </HStack>
            </VStack>

            <Box flex={1} w="100%">
              <Text color={textColor} fontSize="lg" fontWeight="bold" mb="15px">
                {t("detail.basicInfo")}
              </Text>
              <SimpleGrid columns={{ base: 1, sm: 2, md: 3 }} spacing="20px">
                <Box
                  borderBottom="1px solid"
                  borderColor={borderColor}
                  pb="10px"
                >
                  <Text color={textColorSecondary} fontSize="xs">
                    {t("table.columns.personCode")}
                  </Text>
                  <Text
                    color={textColor}
                    fontSize="sm"
                    fontWeight="500"
                    mt="5px"
                  >
                    {person.person_code || "-"}
                  </Text>
                </Box>
                <Box
                  borderBottom="1px solid"
                  borderColor={borderColor}
                  pb="10px"
                >
                  <Text color={textColorSecondary} fontSize="xs">
                    {t("table.columns.gender")}
                  </Text>
                  <Text
                    color={textColor}
                    fontSize="sm"
                    fontWeight="500"
                    mt="5px"
                  >
                    {person.gender === "male"
                      ? t("form.gender.male")
                      : person.gender === "female"
                        ? t("form.gender.female")
                        : t("form.gender.unknown")}
                  </Text>
                </Box>
                <Box
                  borderBottom="1px solid"
                  borderColor={borderColor}
                  pb="10px"
                >
                  <Text color={textColorSecondary} fontSize="xs">
                    {t("table.columns.phone")}
                  </Text>
                  <Text
                    color={textColor}
                    fontSize="sm"
                    fontWeight="500"
                    mt="5px"
                  >
                    {person.phone
                      ? canViewSensitive
                        ? person.phone
                        : person.phone.replace(
                            /(\d{3})\d{4}(\d{4})/,
                            "$1****$2",
                          )
                      : "-"}
                  </Text>
                </Box>

                <Box
                  borderBottom="1px solid"
                  borderColor={borderColor}
                  pb="10px"
                >
                  <Text color={textColorSecondary} fontSize="xs">
                    {t("table.columns.createdAt")}
                  </Text>
                  <Text
                    color={textColor}
                    fontSize="sm"
                    fontWeight="500"
                    mt="5px"
                  >
                    {formatDateTime(person.created_at)}
                  </Text>
                </Box>
              </SimpleGrid>

              {/* 分组信息 */}
              <Box mt="20px">
                <Text color={textColor} fontSize="sm" fontWeight="600" mb="8px">
                  {t("table.columns.groups")}
                </Text>
                {person.groups && person.groups.length > 0 ? (
                  <HStack spacing={2} wrap="wrap">
                    {person.groups.map((g) => (
                      <Badge
                        key={g.id}
                        variant="subtle"
                        colorScheme="brand"
                        fontSize="sm"
                        px="3"
                        py="1"
                        borderRadius="full"
                      >
                        {g.group_name}
                      </Badge>
                    ))}
                  </HStack>
                ) : (
                  <Text color={textColorSecondary} fontSize="sm">
                    {t("detail.noGroups")}
                  </Text>
                )}
              </Box>
            </Box>
          </Flex>
        </Card>

        <Card px="24px" py="24px">
          <Text color={textColor} fontSize="lg" fontWeight="bold" mb="20px">
            {t("detail.recentIdentifications")} ({smartRecords.length})
          </Text>
          {loadingTabs ? (
            <Center py="40px">
              <Spinner />
            </Center>
          ) : smartRecords.length === 0 ? (
            <Center py="40px">
              <Text color={textColorSecondary}>{t("detail.noRecords")}</Text>
            </Center>
          ) : (
            <Box overflowX="auto">
              <Table variant="simple" size="sm">
                <Thead>
                  <Tr>
                    <Th>{t("table.columns.faceImage")}</Th>
                    <Th>{tCommon("table.columns.time")}</Th>
                    <Th>{t("table.columns.device")}</Th>
                    <Th>{t("table.columns.similarity")}</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {smartRecords.map((rec) => (
                    <Tr key={rec.record_id}>
                      <Td>
                        <Image
                          src={getFileUrl(
                            rec.target_crop_url || rec.snapshot_image_url || "",
                          )}
                          boxSize="40px"
                          objectFit="cover"
                          borderRadius="md"
                          cursor="pointer"
                          onClick={() =>
                            setPreviewImage(
                              getFileUrl(
                                rec.target_crop_url ||
                                  rec.snapshot_image_url ||
                                  "",
                              ),
                            )
                          }
                        />
                      </Td>
                      <Td>{formatDateTime(rec.created_at)}</Td>
                      <Td>{rec.device_name || "-"}</Td>
                      <Td>
                        {rec.similarity
                          ? `${(rec.similarity * 100).toFixed(2)}%`
                          : "-"}
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Box>
          )}
        </Card>
      </Flex>

      {/* Image Preview Modal */}
      <Modal
        isOpen={!!previewImage}
        onClose={() => setPreviewImage(null)}
        isCentered
        size="xl"
      >
        <ModalOverlay backdropFilter="blur(8px)" />
        <ModalContent bg="transparent" boxShadow="none">
          <ModalCloseButton color="white" />
          <ModalBody
            p={0}
            display="flex"
            justifyContent="center"
            alignItems="center"
          >
            {previewImage && (
              <Image
                src={previewImage}
                maxH="80vh"
                maxW="100%"
                objectFit="contain"
                borderRadius="md"
              />
            )}
          </ModalBody>
        </ModalContent>
      </Modal>
    </Box>
  );
}
