import { ChevronLeftIcon, RepeatIcon, DeleteIcon } from "@chakra-ui/icons";
import {
  Box,
  Button,
  Center,
  Flex,
  HStack,
  SimpleGrid,
  Spinner,
  Table,
  Tag,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  IconButton,
  useColorModeValue,
  useDisclosure,
  useToast,
} from "@chakra-ui/react";
import Card from "components/card/Card";
import ConfirmDialog from "components/confirm-dialog/ConfirmDialog";
import { useDateFormat } from "hooks/useDateFormat";
import { useWebSocket } from "hooks/useWebSocket";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams, useLocation } from "react-router-dom";
import {
  edgeNodeApi,
  type EdgeNode,
  type NodeAlgorithm,
} from "services/edgeNode";
import { AlgorithmDeployModal } from "./components/AlgorithmDeployModal";
import EdgeNodeEditModal from "./components/EdgeNodeEditModal";

const STATUS_COLORS: Record<string, string> = {
  online: "green",
  offline: "gray",
  error: "red",
  disabled: "orange",
};

const ALGO_STATUS_COLORS: Record<string, string> = {
  pending: "orange",
  downloading: "blue",
  installed: "green",
  failed: "red",
};

function formatUptime(seconds: number): string {
  if (!seconds) return "-";
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${d}d`);
  if (h > 0) parts.push(`${h}h`);
  parts.push(`${m}m`);
  return parts.join(" ");
}

export default function EdgeNodeDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useTranslation("modules/edge-nodes");
  const { t: tCommon } = useTranslation("common");
  const { formatDateTime } = useDateFormat();

  const textColor = useColorModeValue("secondaryGray.900", "white");
  const textColorSecondary = "gray.400";
  const borderColor = useColorModeValue("gray.200", "whiteAlpha.100");
  const toast = useToast();

  const [node, setNode] = useState<EdgeNode | null>((location.state as any)?.updatedNode || null);
  const [algorithms, setAlgorithms] = useState<NodeAlgorithm[]>([]);
  const [hasLoaded, setHasLoaded] = useState(!!node);
  const [hasError, setHasError] = useState(false);
  const [isAlgosLoading, setIsAlgosLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [retryingAlgo, setRetryingAlgo] = useState<string | null>(null);
  const [deletingAlgoId, setDeletingAlgoId] = useState<string | null>(null);
  const {
    isOpen: isDeleteOpen,
    onOpen: onDeleteOpen,
    onClose: onDeleteClose,
  } = useDisclosure();
  const {
    isOpen: isDeployOpen,
    onOpen: onDeployOpen,
    onClose: onDeployClose,
  } = useDisclosure();
  const {
    isOpen: isEditOpen,
    onOpen: onEditOpen,
    onClose: onEditClose,
  } = useDisclosure();

  // WebSocket: auto-refresh node & algorithms when heartbeat/deployment status updates come in
  useWebSocket({
    onMessage: useCallback((msg: any) => {
      if (msg.payload?.node_id !== id) return;
      const statusChanged = msg.type === "edge-node-status";
      const algoChanged = msg.type === "edge-node-algo-status";
      if (statusChanged) {
        edgeNodeApi.get(id).then(setNode).catch(() => {});
      }
      if (statusChanged || algoChanged) {
        edgeNodeApi.getNodeAlgorithms(id).then(setAlgorithms).catch(() => {});
      }
    }, [id]),
  });

  // 只执行初始加载（如果 node 还没有数据），后续不再主动刷新以避免闪烁
  useEffect(() => {
    if (!id) return;
    if (hasLoaded) return;
    edgeNodeApi
      .get(id)
      .then((data) => {
        setNode(data);
        setHasLoaded(true);
      })
      .catch((err) => {
        console.error("Load node failed:", err);
        setHasError(true);
        toast({ title: t("message.loadFailed"), status: "error" });
      });
  }, [id]);

  // 只加载一次算法列表
  useEffect(() => {
    if (!id || !hasLoaded) return;
    setIsAlgosLoading(true);
    edgeNodeApi
      .getNodeAlgorithms(id)
      .then(setAlgorithms)
      .catch(() =>
        toast({ title: t("message.loadAlgosFailed"), status: "error" }),
      )
      .finally(() => setIsAlgosLoading(false));
  }, [id, hasLoaded, toast, t]);

  const handleDelete = useCallback(async () => {
    if (!node) return;
    setIsDeleting(true);
    try {
      await edgeNodeApi.delete(node.id);
      toast({ title: t("message.deleteSuccess"), status: "success" });
      navigate("/admin/devices/edge-nodes");
    } catch {
      toast({ title: t("message.deleteFailed"), status: "error" });
    } finally {
      setIsDeleting(false);
    }
  }, [node, toast, t, navigate]);

  const handleDeploy = useCallback(() => {
    if (!node) return;
    onDeployOpen();
  }, [node, onDeployOpen]);

  const handleRetryAlgo = useCallback(async (algoPackageId: string) => {
    if (!node) return;
    setRetryingAlgo(algoPackageId);
    try {
      await edgeNodeApi.deployAlgorithm(node.id, algoPackageId);
      toast({ title: t("message.retrySuccess"), status: "success" });
      const algos = await edgeNodeApi.getNodeAlgorithms(node.id);
      setAlgorithms(algos);
    } catch {
      toast({ title: t("message.retryFailed"), status: "error" });
    } finally {
      setRetryingAlgo(null);
    }
  }, [node, toast, t]);

  const handleDeleteAlgo = useCallback(async (algoPackageId: string) => {
    if (!node) return;
    setDeletingAlgoId(algoPackageId);
    try {
      await edgeNodeApi.deleteAlgorithm(node.id, algoPackageId);
      toast({ title: t("message.deleteAlgoSuccess"), status: "success" });
      const algos = await edgeNodeApi.getNodeAlgorithms(node.id);
      setAlgorithms(algos);
    } catch {
      toast({ title: t("message.deleteAlgoFailed"), status: "error" });
    } finally {
      setDeletingAlgoId(null);
    }
  }, [node, toast, t]);



  const handleEditSuccess = useCallback((updatedNode: EdgeNode) => {
    setNode(updatedNode);
  }, []);

  const handleEscBack = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        navigate("/admin/devices/edge-nodes");
      }
    },
    [navigate],
  );

  useEffect(() => {
    document.addEventListener("keydown", handleEscBack);
    return () => document.removeEventListener("keydown", handleEscBack);
  }, [handleEscBack]);

  // 在数据还没加载完成时，渲染占位框架而非 spinner
  if (!hasLoaded && !hasError && !node) {
    return (
      <Box pt={{ base: "130px", md: "80px", xl: "80px" }}>
        <Center h="50vh">
          <Spinner size="xl" color="brand.500" />
        </Center>
      </Box>
    );
  }

  if ((!hasLoaded || !node) && hasError) {
    return (
      <Box pt={{ base: "130px", md: "80px", xl: "80px" }}>
        <Center h="50vh">
          <Text>{tCommon("message.notFound")}</Text>
        </Center>
      </Box>
    );
  }

  // 如果 node 已有数据（初始数据或弹窗更新后的数据），立即渲染内容
  if (!node) return null;

  return (
    <Box pt={{ base: "130px", md: "80px", xl: "80px" }}>
      <Flex direction="column" mb="20px">
        {/* Header */}
        <HStack mb="20px" spacing={4}>
          <Button
            leftIcon={<ChevronLeftIcon />}
            variant="ghost"
            onClick={() => navigate("/admin/devices/edge-nodes")}
          >
            {tCommon("button.back")}
          </Button>
          <Text color={textColor} fontSize="2xl" fontWeight="bold">
            {node.name}
          </Text>
          <Tag colorScheme={STATUS_COLORS[node.status] || "gray"} size="md">
            {t(`status.${node.status}`)}
          </Tag>
          <Box flex={1} />
          <Button variant="outline" size="sm" onClick={onEditOpen}>
            {t("actions.editNode")}
          </Button>
          <Button
            colorScheme="red"
            variant="outline"
            size="sm"
            onClick={onDeleteOpen}
          >
            {t("actions.delete")}
          </Button>
        </HStack>

        {/* Basic Info Card */}
        <Card px="24px" py="24px" mb="20px">
          <Text color={textColor} fontSize="lg" fontWeight="bold" mb="15px">
            {t("basicInfo")}
          </Text>
          <SimpleGrid columns={{ base: 1, sm: 2, md: 3 }} spacing="20px">
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.name")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {node.name}
              </Text>
            </Box>
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.endpoint")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {node.endpoint || "-"}
              </Text>
            </Box>
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.currentLoad")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {node.current_load ?? 0}/{node.max_load ?? "-"}
              </Text>
            </Box>
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.engineVersion")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {node.engine_version || "-"}
              </Text>
            </Box>
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.uptime")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {formatUptime(node.uptime)}
              </Text>
            </Box>
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.lastHeartbeat")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {node.last_heartbeat
                  ? formatDateTime(node.last_heartbeat)
                  : "-"}
              </Text>
            </Box>
            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
              <Text color={textColorSecondary} fontSize="xs">
                {t("fields.description")}
              </Text>
              <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                {node.description || "-"}
              </Text>
            </Box>
          </SimpleGrid>
        </Card>

        {/* Hardware Info Card */}
        {node.hardware_info && (
          <Card px="24px" py="24px" mb="20px">
            <Text color={textColor} fontSize="lg" fontWeight="bold" mb="15px">
              {t("fields.hardwareInfo")}
            </Text>
            <SimpleGrid columns={{ base: 1, sm: 2, md: 3 }} spacing="20px">
              <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                <Text color={textColorSecondary} fontSize="xs">
                  {t("fields.cpuModel")}
                </Text>
                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                  {node.hardware_info.cpu_model || "-"}
                </Text>
              </Box>
              <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                <Text color={textColorSecondary} fontSize="xs">
                  {t("fields.gpuModel")}
                </Text>
                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                  {node.hardware_info.gpu_model || "-"}
                </Text>
              </Box>
              <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                <Text color={textColorSecondary} fontSize="xs">
                  {t("fields.memory")}
                </Text>
                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                  {node.hardware_info.memory
                    ? `${(Number(node.hardware_info.memory) / (1024 * 1024 * 1024)).toFixed(1)} GB`
                    : "-"}
                </Text>
              </Box>
              <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                <Text color={textColorSecondary} fontSize="xs">
                  {t("fields.platform")}
                </Text>
                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                  {node.hardware_info.platform || "-"}
                </Text>
              </Box>
            </SimpleGrid>
          </Card>
        )}

        {/* Deploy Algorithm Button */}
        <Flex justify="flex-end" mb="20px">
          <Button colorScheme="brand" onClick={handleDeploy}>
            {t("actions.deployAlgo")}
          </Button>
        </Flex>

        {/* Installed Algorithms Card */}
        <Card px="0px" pb="20px">
          <Text
            color={textColor}
            fontSize="lg"
            fontWeight="bold"
            px="24px"
            pt="24px"
            pb="15px"
          >
            {t("actions.deployAlgo")}
          </Text>
          <Box overflowX="auto">
            <Table variant="simple" color="gray.500">
              <Thead>
                <Tr>
                  <Th>{t("fields.algoName")}</Th>
                  <Th>{t("fields.algoVersion")}</Th>
                  <Th>{t("fields.status")}</Th>
                  <Th>{t("fields.installPath")}</Th>
                  <Th>{t("fields.retryCount")}</Th>
                  <Th>{t("fields.lastRetryAt")}</Th>
                  <Th>{t("fields.errorMessage")}</Th>
                  <Th>{t("fields.actions")}</Th>
                </Tr>
              </Thead>
              <Tbody>
                {isAlgosLoading ? (
                  <Tr>
                    <Td colSpan={8}>
                      <Center py={4}>
                        <Spinner size="sm" />
                      </Center>
                    </Td>
                  </Tr>
                ) : algorithms.length === 0 ? (
                  <Tr>
                    <Td colSpan={8}>
                      <Center py={4}>{tCommon("noData")}</Center>
                    </Td>
                  </Tr>
                ) : (
                  algorithms.map((algo) => (
                    <Tr key={algo.id}>
                      <Td>
                        <Text
                          fontWeight="medium"
                          color={textColor}
                          fontSize="sm"
                        >
                          {algo.algo_name || algo.algo_package?.algorithm_name || "-"}
                        </Text>
                      </Td>
                      <Td>
                        <Text fontSize="sm">{algo.algo_version || algo.algo_package?.version || "-"}</Text>
                      </Td>
                      <Td>
                        <Tag
                          colorScheme={
                            ALGO_STATUS_COLORS[algo.status] || "gray"
                          }
                          size="sm"
                        >
                          {t(`algorithmStatus.${algo.status}`, algo.status)}
                        </Tag>
                      </Td>
                      <Td>
                        <Text fontSize="sm">{algo.install_path || "-"}</Text>
                      </Td>
                      <Td>
                        <Text fontSize="sm">
                          {algo.retry_count > 0
                            ? `${algo.retry_count}/${algo.max_retry_count}`
                            : "-"}
                        </Text>
                      </Td>
                      <Td>
                        <Text fontSize="sm">
                          {algo.last_retry_at
                            ? formatDateTime(algo.last_retry_at)
                            : "-"}
                        </Text>
                      </Td>
                      <Td>
                        <Text
                          fontSize="sm"
                          color="red.500"
                          maxW="200px"
                          isTruncated
                        >
                          {algo.error_message || "-"}
                        </Text>
                      </Td>
                      <Td>
                        <HStack spacing={1}>
                          {algo.status === "failed" && (
                            <IconButton
                              aria-label={t("actions.retryAlgo")}
                              icon={<RepeatIcon />}
                              size="sm"
                              variant="ghost"
                              colorScheme="blue"
                              isLoading={retryingAlgo === algo.algo_package_id}
                              onClick={() =>
                                handleRetryAlgo(algo.algo_package_id)
                              }
                            />
                          )}
                          <IconButton
                            aria-label={t("actions.deleteAlgo")}
                            icon={<DeleteIcon />}
                            size="sm"
                            variant="ghost"
                            colorScheme="red"
                            isLoading={deletingAlgoId === algo.algo_package_id}
                            onClick={() =>
                              handleDeleteAlgo(algo.algo_package_id)
                            }
                          />
                        </HStack>
                      </Td>
                    </Tr>
                  ))
                )}
              </Tbody>
            </Table>
          </Box>
        </Card>
      </Flex>

      <ConfirmDialog
        isOpen={isDeleteOpen}
        onClose={onDeleteClose}
        onConfirm={handleDelete}
        title={tCommon("button.confirm")}
        message={t("message.deleteConfirm")}
        isLoading={isDeleting}
      />

      {node && (
        <AlgorithmDeployModal
          isOpen={isDeployOpen}
          onClose={onDeployClose}
          onSuccess={() => {
            edgeNodeApi.getNodeAlgorithms(id!).then(setAlgorithms).catch(() => {});
          }}
          node={node}
        />
      )}

      {node && (
        <EdgeNodeEditModal
          isOpen={isEditOpen}
          onClose={onEditClose}
          node={node}
          onSuccess={handleEditSuccess}
        />
      )}
    </Box>
  );
}
