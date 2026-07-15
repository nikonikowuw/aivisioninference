import { ChevronLeftIcon, DeleteIcon, RepeatIcon } from "@chakra-ui/icons";
import {
    Box,
    Button,
    Center,
    Select as ChakraSelect,
    Flex,
    HStack,
    IconButton,
    SimpleGrid,
    Spinner,
    Tab,
    Table,
    TabList,
    TabPanel,
    TabPanels,
    Tabs,
    Tag,
    Tbody,
    Td,
    Text,
    Th,
    Thead,
    Tr,
    useColorModeValue,
    useDisclosure,
    useToast
} from "@chakra-ui/react";
import Card from "components/card/Card";
import MetricsTimeSeries from "components/charts/MetricsTimeSeries";
import ConfirmDialog from "components/confirm-dialog/ConfirmDialog";
import Terminal from "components/terminal/Terminal";
import { useDateFormat } from "hooks/useDateFormat";
import { useWebSocket } from "hooks/useWebSocket";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import {
    edgeNodeApi,
    type EdgeNode,
    type NodeAlgorithm,
} from "services/edgeNode";
import {
    edgeNodeMetricsApi,
    type MetricDataPoint,
    type NodeMetrics,
} from "services/edgeNodeMetrics";
import { formatUptime } from "utils/convert";
import { AlgorithmDeployModal } from "./components/AlgorithmDeployModal";
import EdgeNodeEditModal from "./components/EdgeNodeEditModal";
import ScheduledTaskList from "./scheduled-tasks/index";

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

function formatByteRate(bytesPerSecond: number): string {
    if (!bytesPerSecond) return "0 B/s";
    if (bytesPerSecond >= 1_000_000_000) return `${(bytesPerSecond / 1_000_000_000).toFixed(2)} GB/s`;
    if (bytesPerSecond >= 1_000_000) return `${(bytesPerSecond / 1_000_000).toFixed(2)} MB/s`;
    if (bytesPerSecond >= 1_000) return `${(bytesPerSecond / 1_000).toFixed(2)} KB/s`;
    return `${bytesPerSecond.toFixed(0)} B/s`;
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
    const [liveMetrics, setLiveMetrics] = useState<NodeMetrics | null>(null);
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

    // Metrics state
    const [timeRange, setTimeRange] = useState<string>("1h");
    const [cpuData, setCpuData] = useState<MetricDataPoint[]>([]);
    const [memData, setMemData] = useState<MetricDataPoint[]>([]);
    const [netRxData, setNetRxData] = useState<MetricDataPoint[]>([]);
    const [netTxData, setNetTxData] = useState<MetricDataPoint[]>([]);
    const [accData, setAccData] = useState<MetricDataPoint[]>([]);
    const [streamData, setStreamData] = useState<MetricDataPoint[]>([]);
    const [metricsLoading, setMetricsLoading] = useState(false);

    // Compute time range params
    const getTimeParams = useCallback(() => {
        const now = new Date();
        let from: Date;
        switch (timeRange) {
            case "6h": from = new Date(now.getTime() - 6 * 60 * 60 * 1000); break;
            case "24h": from = new Date(now.getTime() - 24 * 60 * 60 * 1000); break;
            case "7d": from = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000); break;
            default: from = new Date(now.getTime() - 60 * 60 * 1000); break; // 1h
        }
        return {
            from: from.toISOString(),
            to: now.toISOString(),
            page_size: 500,
        };
    }, [timeRange]);

    // Fetch metrics when time range or node loads
    useEffect(() => {
        if (!id || !hasLoaded) return;
        const timeParams = getTimeParams();
        setMetricsLoading(true);
        Promise.all([
            edgeNodeMetricsApi.queryMetrics(id, { metric: "cpu_usage", ...timeParams }),
            edgeNodeMetricsApi.queryMetrics(id, { metric: "memory_usage", ...timeParams }),
            edgeNodeMetricsApi.queryMetrics(id, { metric: "net_rx_bytes", ...timeParams }),
            edgeNodeMetricsApi.queryMetrics(id, { metric: "net_tx_bytes", ...timeParams }),
            edgeNodeMetricsApi.queryMetrics(id, { metric: "accelerator_utilization", ...timeParams }),
            edgeNodeMetricsApi.queryMetrics(id, { metric: "active_stream_count", ...timeParams }),
        ])
            .then(([cpu, mem, netRx, netTx, acc, streams]) => {
                setCpuData(cpu.list || []);
                setMemData(mem.list || []);
                setNetRxData(netRx.list || []);
                setNetTxData(netTx.list || []);
                setAccData(acc.list || []);
                setStreamData(streams.list || []);
            })
            .catch((err) => {
                console.error("Failed to load metrics:", err);
            })
            .finally(() => setMetricsLoading(false));
    }, [id, hasLoaded, getTimeParams]);

    // WebSocket: auto-refresh node & algorithms when heartbeat/deployment status updates come in
    const { send } = useWebSocket({
        onOpen: useCallback((ws: WebSocket) => {
            if (!id) return;
            ws.send(JSON.stringify({ type: "subscribe", payload: { node_id: id, topic: "metrics" } }));
            ws.send(JSON.stringify({ type: "subscribe", payload: { node_id: id, topic: "edge-node-status" } }));
            ws.send(JSON.stringify({ type: "subscribe", payload: { node_id: id, topic: "edge-node-algo-status" } }));
        }, [id]),
        onMessage: useCallback((msg: any) => {
            if (msg.payload?.node_id !== id) return;
            const statusChanged = msg.type === "edge-node-status";
            const algoChanged = msg.type === "edge-node-algo-status";
            if (statusChanged) {
                edgeNodeApi.get(id).then(setNode).catch(() => { });
            }
            if (statusChanged || algoChanged) {
                edgeNodeApi.getNodeAlgorithms(id).then(setAlgorithms).catch(() => { });
            }
            if (msg.type === "edge-node-metrics") {
                const payload = msg.payload || msg;
                setLiveMetrics((prev) => {
                    const base = prev || {
                        cpu_usage: 0,
                        memory_usage: 0,
                        cpu_load_1m: 0,
                        cpu_load_5m: 0,
                        cpu_load_15m: 0,
                        net_rx_speed: 0,
                        net_tx_speed: 0,
                        process_count: 0,
                        thread_count: 0,
                        temperature: 0,
                    };
                    return {
                        ...base,
                        cpu_usage: payload.cpu_usage ?? 0,
                        memory_usage: payload.memory_usage ?? 0,
                        cpu_load_1m: payload.cpu_load_1m ?? 0,
                        cpu_load_5m: payload.cpu_load_5m ?? 0,
                        cpu_load_15m: payload.cpu_load_15m ?? 0,
                        net_rx_speed: payload.net_rx_speed ?? 0,
                        net_tx_speed: payload.net_tx_speed ?? 0,
                        process_count: payload.process_count ?? 0,
                        thread_count: payload.thread_count ?? 0,
                        temperature: payload.temperature ?? 0,
                    };
                });
            } else if (msg.type === "edge-node-engine-metrics") {
                const payload = msg.payload || msg;
                setLiveMetrics((prev) => {
                    const base = prev || {
                        cpu_usage: 0,
                        memory_usage: 0,
                        cpu_load_1m: 0,
                        cpu_load_5m: 0,
                        cpu_load_15m: 0,
                        net_rx_speed: 0,
                        net_tx_speed: 0,
                        process_count: 0,
                        thread_count: 0,
                        temperature: 0,
                    };
                    return {
                        ...base,
                        active_stream_count: payload.active_stream_count,
                        decode_slots_used: payload.decode_slots_used,
                        encode_slots_used: payload.encode_slots_used,
                        egress_bps: payload.egress_bps,
                        accelerator_utilization: payload.accelerator_utilization,
                        accelerator_metrics_valid: payload.accelerator_metrics_valid,
                    };
                });
            }
        }, [id]),
    });

    useEffect(() => {
        if (!id) return;
        send({ type: "subscribe", payload: { node_id: id, topic: "metrics" } });
        send({ type: "subscribe", payload: { node_id: id, topic: "edge-node-status" } });
        send({ type: "subscribe", payload: { node_id: id, topic: "edge-node-algo-status" } });
        return () => {
            send({ type: "unsubscribe", payload: { node_id: id, topic: "metrics" } });
            send({ type: "unsubscribe", payload: { node_id: id, topic: "edge-node-status" } });
            send({ type: "unsubscribe", payload: { node_id: id, topic: "edge-node-algo-status" } });
        };
    }, [id, send]);

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
                                    {node.hal_platform || "-"}
                                </Text>
                            </Box>
                        </SimpleGrid>
                    </Card>
                )}

                {/* Live Metrics Card */}
                <Card px="24px" py="24px" mb="20px">
                    <Text color={textColor} fontSize="lg" fontWeight="bold" mb="15px">
                        {t('fields.metrics')}
                    </Text>
                    {liveMetrics ? (
                        <SimpleGrid columns={{ base: 1, sm: 2, md: 4, lg: 6 }} spacing="20px">
                            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                                <Text color={textColorSecondary} fontSize="xs">{t('fields.cpuUsage')}</Text>
                                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                                    {liveMetrics.cpu_usage.toFixed(1)}%
                                </Text>
                            </Box>
                            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                                <Text color={textColorSecondary} fontSize="xs">{t('fields.memUsage')}</Text>
                                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                                    {liveMetrics.memory_usage.toFixed(1)}%
                                </Text>
                            </Box>
                            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                                <Text color={textColorSecondary} fontSize="xs">{t('fields.cpuLoad')}</Text>
                                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                                    {liveMetrics.cpu_load_1m.toFixed(2)} / {liveMetrics.cpu_load_5m.toFixed(2)} / {liveMetrics.cpu_load_15m.toFixed(2)}
                                </Text>
                            </Box>
                            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                                <Text color={textColorSecondary} fontSize="xs">{t('fields.networkRxTx')}</Text>
                                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                                    {formatByteRate(liveMetrics.net_rx_speed)} / {formatByteRate(liveMetrics.net_tx_speed)}
                                </Text>
                            </Box>
                            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                                <Text color={textColorSecondary} fontSize="xs">{t('fields.acceleratorUsage')}</Text>
                                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                                    {liveMetrics.accelerator_metrics_valid && liveMetrics.accelerator_utilization !== undefined
                                        ? `${liveMetrics.accelerator_utilization.toFixed(1)}%`
                                        : 'N/A'}
                                </Text>
                            </Box>
                            <Box borderBottom="1px solid" borderColor={borderColor} pb="10px">
                                <Text color={textColorSecondary} fontSize="xs">{t('fields.activeStreams')}</Text>
                                <Text color={textColor} fontSize="sm" fontWeight="500" mt="5px">
                                    {liveMetrics.active_stream_count !== undefined ? liveMetrics.active_stream_count : 'N/A'}
                                </Text>
                            </Box>
                        </SimpleGrid>
                    ) : (
                        <Text color={textColorSecondary} fontSize="sm">
                            {t('message.noMetrics')}
                        </Text>
                    )}
                </Card>

                {/* Metrics Charts */}
                <Card px="24px" py="24px" mb="20px">
                    <Flex justify="space-between" align="center" mb="15px">
                        <Text color={textColor} fontSize="lg" fontWeight="bold">
                            {t("metrics.title")}
                        </Text>
                        <ChakraSelect
                            value={timeRange}
                            onChange={(e) => setTimeRange(e.target.value)}
                            width="120px"
                            size="sm"
                        >
                            <option value="1h">{t("metrics.last1h")}</option>
                            <option value="6h">{t("metrics.last6h")}</option>
                            <option value="24h">{t("metrics.last24h")}</option>
                            <option value="7d">{t("metrics.last7d")}</option>
                        </ChakraSelect>
                    </Flex>
                    <SimpleGrid columns={{ base: 1, md: 2 }} spacing="20px">
                        <Box>
                            <MetricsTimeSeries
                                title={t("metrics.cpuUsage")}
                                data={cpuData}
                                loading={metricsLoading}
                                unit="%"
                                colorScheme="blue"
                                height={200}
                            />
                        </Box>
                        <Box>
                            <MetricsTimeSeries
                                title={t("metrics.memoryUsage")}
                                data={memData}
                                loading={metricsLoading}
                                unit="%"
                                colorScheme="green"
                                height={200}
                            />
                        </Box>
                        <Box>
                            <MetricsTimeSeries
                                title={t("metrics.netRx")}
                                data={netRxData}
                                loading={metricsLoading}
                                unit="B"
                                colorScheme="purple"
                                height={200}
                            />
                        </Box>
                        <Box>
                            <MetricsTimeSeries
                                title={t("metrics.netTx")}
                                data={netTxData}
                                loading={metricsLoading}
                                unit="B"
                                colorScheme="orange"
                                height={200}
                            />
                        </Box>
                        <Box>
                            <MetricsTimeSeries
                                title={t('metrics.acceleratorUsage')}
                                data={accData}
                                loading={metricsLoading}
                                unit="%"
                                colorScheme="cyan"
                                height={200}
                            />
                        </Box>
                        <Box>
                            <MetricsTimeSeries
                                title={t('metrics.activeVideoStreams')}
                                data={streamData}
                                loading={metricsLoading}
                                unit=""
                                colorScheme="pink"
                                height={200}
                            />
                        </Box>
                    </SimpleGrid>
                </Card>

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

                {/* Phase 3: Remote Operations — Scheduled Tasks & Web Terminal */}
                <Card px="24px" py="24px" mb="20px">
                    <Tabs colorScheme="brand" variant="enclosed">
                        <TabList>
                            <Tab>{t("scheduledTasks.title")}</Tab>
                            <Tab>{t("terminal.title")}</Tab>
                        </TabList>

                        <TabPanels>
                            <TabPanel px="0" pt="20px">
                                <ScheduledTaskList nodeId={node.id} />
                            </TabPanel>
                            <TabPanel px="0" pt="20px">
                                <Terminal nodeId={node.id} />
                            </TabPanel>
                        </TabPanels>
                    </Tabs>
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
                        edgeNodeApi.getNodeAlgorithms(id!).then(setAlgorithms).catch(() => { });
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
