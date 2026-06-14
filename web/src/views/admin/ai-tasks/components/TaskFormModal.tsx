import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  Button,
  FormControl,
  FormLabel,
  Select,
  VStack,
  useToast,
  HStack,
  Text,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Box,
  Badge,
  Tag,
  Divider,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
} from '@chakra-ui/react';
import { useState, useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { aiVisionTasksApi, devicesApi, algoPackagesApi, aiTimeSchedulesApi, mediaApi, type AIVisionTask, type Device, type AlgorithmPackage, type AITimeSchedule, type ROIRegion, type MarkRegion, type LineRegion } from 'services/api';
import { edgeNodeApi, recommendNodeApi, type EdgeNode } from 'services/edgeNode';
import DynamicParamsForm from 'components/DynamicParamsForm';
import RegionCanvas from 'components/RegionCanvas';
import VideoPlayer from 'components/VideoPlayer';

interface TaskFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  initialData?: AIVisionTask | null;
}

interface FormData {
  name: string;
  target_node_id: string;
  algo_package_id: string;
  device_channel_id: string;
  schedule_id: string;
  ai_params: Record<string, any>;
  roi_regions: ROIRegion[];
  mark_regions: MarkRegion[];
  line_regions: LineRegion[];
}

const edgeNodeOptions: EdgeNodeOption[] = [
  { id: '00000000-0000-4000-8000-000000000001', name: 'form.defaultNode' },
  { id: '00000000-0000-4000-8000-000000000002', name: 'form.highPerformanceNode' },
];

interface EdgeNodeOption {
  id: string;
  name: string;
  loadRate?: number;
  isRecommended?: boolean;
}

const defaultForm: FormData = {
  name: '',
  target_node_id: edgeNodeOptions[0].id,
  algo_package_id: '',
  device_channel_id: '',
  schedule_id: '',
  ai_params: {},
  roi_regions: [],
  mark_regions: [],
  line_regions: [],
};

export default function TaskFormModal({ isOpen, onClose, onSuccess, initialData }: TaskFormModalProps) {
  const [form, setForm] = useState<FormData>(defaultForm);
  const [devices, setDevices] = useState<Device[]>([]);
  const [algos, setAlgos] = useState<AlgorithmPackage[]>([]);
  const [schedules, setSchedules] = useState<AITimeSchedule[]>([]);
  const [edgeNodes, setEdgeNodes] = useState<EdgeNodeOption[]>(edgeNodeOptions);
  const [recommendedNodeId, setRecommendedNodeId] = useState<string | null>(null);
  const [conflictError, setConflictError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [regionVideoUrl, setRegionVideoUrl] = useState('');
  const [regionVideoError, setRegionVideoError] = useState<string | null>(null);
  const regionDeviceIdRef = useRef<string | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  const toast = useToast();
  const { t } = useTranslation(['modules/ai-tasks', 'common']);

  // 加载设备列表、算法、时间配置
  useEffect(() => {
    if (!isOpen) return;
    setConflictError(null);

    devicesApi.list({ page: 1, page_size: 1000, status: 'online' }).then(res => {
      setDevices(res.list);
    }).catch(() => {});

    algoPackagesApi.list({ page: 1, page_size: 1000 }).then(res => {
      setAlgos(res.list);
    }).catch(() => {});

    edgeNodeApi.list({ page: 1, page_size: 1000 }).then(res => {
      const dynamicNodes: EdgeNodeOption[] = res.list
        .filter(n => n.status === 'online' && n.enabled)
        .map(n => ({
          id: n.id,
          name: `${n.name} (${n.current_load}/${n.max_load})`,
          loadRate: n.max_load > 0 ? n.current_load / n.max_load : 0,
        }));
      setEdgeNodes([...edgeNodeOptions, ...dynamicNodes]);
    }).catch(() => {});

    aiTimeSchedulesApi.listAll().then(res => {
      setSchedules(res);
    }).catch(() => {});

    if (initialData) {
      setForm({
        name: initialData.name,
        target_node_id: initialData.target_node_id,
        algo_package_id: initialData.algo_package_id,
        device_channel_id: initialData.device_channel_id,
        schedule_id: initialData.schedule_id,
        ai_params: initialData.ai_params || {},
        roi_regions: initialData.roi_regions || [],
        mark_regions: initialData.mark_regions || [],
        line_regions: initialData.line_regions || [],
      });
    } else {
      setForm({ ...defaultForm });
    }
  }, [isOpen, initialData]);

  // 算法包变更时自动调用推荐节点接口
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!form.algo_package_id) {
      setRecommendedNodeId(null);
      return;
    }
    recommendNodeApi.recommend(form.algo_package_id)
      .then((res) => {
        setRecommendedNodeId(res.recommended_node_id);
        // 仅在用户未手动选择节点时自动填充推荐节点
        if (!form.target_node_id || form.target_node_id === edgeNodeOptions[0].id) {
          setForm(prev => ({ ...prev, target_node_id: res.recommended_node_id }));
        }
      })
      .catch(() => {
        setRecommendedNodeId(null);
      });
  }, [form.algo_package_id]);

  const updateField = (key: keyof FormData, value: any) => {
    setForm(prev => ({ ...prev, [key]: value }));
  };

  const stopRegionPlay = useCallback((deviceId = regionDeviceIdRef.current) => {
    if (!deviceId) return;
    mediaApi.stopPlay(deviceId).catch(() => {});
    if (regionDeviceIdRef.current === deviceId) {
      regionDeviceIdRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (!isOpen || !form.device_channel_id) {
      stopRegionPlay();
      setRegionVideoUrl('');
      setRegionVideoError(null);
      return;
    }

    let cancelled = false;
    const deviceId = form.device_channel_id;
    setRegionVideoUrl('');
    setRegionVideoError(null);

    if (regionDeviceIdRef.current && regionDeviceIdRef.current !== deviceId) {
      stopRegionPlay();
    }

    mediaApi.getPlayUrl({ device_id: deviceId, protocol: 'auto' })
      .then(data => {
        if (cancelled) {
          stopRegionPlay(deviceId);
          return;
        }
        regionDeviceIdRef.current = deviceId;
        setRegionVideoUrl(data.url);
      })
      .catch((err: any) => {
        if (!cancelled) setRegionVideoError(err?.message || t('canvas.waitingStream'));
      });

    return () => {
      cancelled = true;
    };
  }, [isOpen, form.device_channel_id, stopRegionPlay, t]);

  useEffect(() => () => {
    stopRegionPlay();
  }, [stopRegionPlay]);

  // 获取当前选中的算法包
  const selectedAlgo = algos.find(a => a.id === form.algo_package_id);
  // 获取当前选中的时间配置
  const selectedSchedule = schedules.find(s => s.id === form.schedule_id);

  const handleSubmit = async () => {
    if (!form.name.trim()) { toast({ title: t('message.nameRequired'), status: 'warning' }); return; }
    if (!form.device_channel_id) { toast({ title: t('message.deviceRequired'), status: 'warning' }); return; }
    if (!form.algo_package_id) { toast({ title: t('message.algorithmRequired'), status: 'warning' }); return; }
    if (!form.target_node_id) { toast({ title: t('message.nodeRequired'), status: 'warning' }); return; }
    if (!form.schedule_id) { toast({ title: t('message.scheduleRequired'), status: 'warning' }); return; }

    setSubmitting(true);
    setConflictError(null);

    // 预检冲突
    const schedule = schedules.find(s => s.id === form.schedule_id);
    if (schedule) {
      try {
        await aiVisionTasksApi.checkConflict({
          target_node_id: form.target_node_id,
          start_date: schedule.start_date.substring(0, 10),
          end_date: schedule.end_date.substring(0, 10),
          time_windows: schedule.time_windows,
          exclude_task_id: initialData?.id,
        });
      } catch (err: any) {
        const msg = err?.message || t('message.conflictDefault');
        if (msg.includes('冲突') || msg.includes('conflict') || msg.includes('409')) {
          setConflictError(msg);
        } else {
          toast({ title: t('common:message.operationFailed'), description: msg, status: 'error' });
        }
        setSubmitting(false);
        return;
      }
    }

    // 提交
    try {
      const payload = {
        name: form.name,
        schedule_id: form.schedule_id,
        device_channel_id: form.device_channel_id,
        algo_package_id: form.algo_package_id,
        target_node_id: form.target_node_id,
        ai_params: Object.keys(form.ai_params).length > 0 ? form.ai_params : undefined,
        roi_regions: form.roi_regions.length > 0 ? form.roi_regions : undefined,
        mark_regions: form.mark_regions.length > 0 ? form.mark_regions : undefined,
        line_regions: form.line_regions.length > 0 ? form.line_regions : undefined,
      };

      if (initialData) {
        await aiVisionTasksApi.update(initialData.id, payload);
        toast({ title: t('common:message.updateSuccess'), status: 'success' });
      } else {
        await aiVisionTasksApi.create(payload);
        toast({ title: t('common:message.createSuccess'), status: 'success' });
      }
      onSuccess();
    } catch (err: any) {
      toast({ title: t('common:message.updateFailed'), description: err?.message || '', status: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  const formatTimeWindows = (tw: { start: string; end: string }[]) => {
    if (!tw || tw.length === 0) return '';
    return tw.map(w => `${w.start}-${w.end}`).join('、');
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="4xl" scrollBehavior="inside" isCentered>
      <ModalOverlay />
      <ModalContent maxH="90vh">
        <ModalHeader>{initialData ? t('modal.titleEdit') : t('modal.titleCreate')}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack spacing={5} align="stretch">
            {conflictError && (
              <Alert status="error" borderRadius="md">
                <AlertIcon />
                <Box>
                  <AlertTitle>{t('message.conflictTitle')}</AlertTitle>
                  <AlertDescription>{conflictError}</AlertDescription>
                </Box>
              </Alert>
            )}

            {/* 基本信息 */}
            <Box>
              <Text fontSize="sm" fontWeight="600" color="gray.500" mb={3}>{t('modal.basicInfo')}</Text>
              <VStack spacing={4}>
                <HStack w="100%" spacing={4}>
                  <FormControl isRequired flex={1}>
                    <FormLabel fontSize="sm">{t('fields.name')}</FormLabel>
                    <input
                      style={{
                        width: '100%',
                        padding: '8px 12px',
                        border: '1px solid #E2E8F0',
                        borderRadius: '6px',
                        fontSize: '14px',
                      }}
                      value={form.name}
                      onChange={e => updateField('name', e.target.value)}
                      placeholder={t('form.namePlaceholder')}
                    />
                  </FormControl>
                  <FormControl isRequired flex={1}>
                    <FormLabel fontSize="sm">{t('fields.device')}</FormLabel>
                    <Select placeholder={t('form.selectDevice')} value={form.device_channel_id} onChange={e => updateField('device_channel_id', e.target.value)} size="md">
                      {devices.map(d => (
                        <option key={d.id} value={d.id}>{d.device_name || d.id}</option>
                      ))}
                    </Select>
                  </FormControl>
                </HStack>

                <HStack w="100%" spacing={4}>
                  <FormControl isRequired flex={1}>
                    <FormLabel fontSize="sm">{t('fields.algorithm')}</FormLabel>
                    <Select placeholder={t('form.selectAlgorithm')} value={form.algo_package_id} onChange={e => updateField('algo_package_id', e.target.value)} size="md">
                      {algos.map(a => (
                        <option key={a.id} value={a.id}>{a.algorithm_alias || a.algorithm_name} ({a.version})</option>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl isRequired flex={1}>
                    <FormLabel fontSize="sm">
                      {t('fields.node')}
                      {recommendedNodeId && (
                        <Tag size="sm" colorScheme="green" ml={2} variant="subtle">
                          推荐节点
                        </Tag>
                      )}
                    </FormLabel>
                    <Select value={form.target_node_id} onChange={e => updateField('target_node_id', e.target.value)} size="md">
                      {edgeNodes.map(node => (
                        <option key={node.id} value={node.id}>
                          {node.name.startsWith('form.') ? t(node.name) : node.name}
                          {node.isRecommended ? ' ★' : ''}
                        </option>
                      ))}
                    </Select>
                    {recommendedNodeId && form.target_node_id === recommendedNodeId && (
                      <Text fontSize="xs" color="green.500" mt={1}>
                        ✔ 已为您推荐负载最低的节点
                      </Text>
                    )}
                  </FormControl>
                </HStack>

                <FormControl isRequired>
                  <FormLabel fontSize="sm">{t('fields.schedule')}</FormLabel>
                  <Select placeholder={t('form.selectSchedule')} value={form.schedule_id} onChange={e => updateField('schedule_id', e.target.value)} size="md">
                    {schedules.map(s => (
                      <option key={s.id} value={s.id}>
                        {s.name} ({s.start_date?.substring(0, 10)} ~ {s.end_date?.substring(0, 10)})
                      </option>
                    ))}
                  </Select>
                  {selectedSchedule && (
                    <HStack mt={2} spacing={2}>
                      <Badge colorScheme="blue">
                        {selectedSchedule.start_date?.substring(0, 10)} ~ {selectedSchedule.end_date?.substring(0, 10)}
                      </Badge>
                      <Badge colorScheme="green">
                        {formatTimeWindows(selectedSchedule.time_windows)}
                      </Badge>
                    </HStack>
                  )}
                </FormControl>
              </VStack>
            </Box>

            <Divider />

            {/* 算法参数 */}
            <Accordion allowToggle defaultIndex={form.algo_package_id ? [0] : []}>
              <AccordionItem border="none">
                <AccordionButton px={0} _hover={{ bg: 'transparent' }}>
                  <Box flex="1" textAlign="left">
                    <Text fontSize="sm" fontWeight="600" color="gray.500">
                      {t('fields.algoParams')} {selectedAlgo && `(${selectedAlgo.algorithm_alias || selectedAlgo.algorithm_name})`}
                    </Text>
                  </Box>
                  <AccordionIcon />
                </AccordionButton>
                <AccordionPanel px={0} pb={4}>
                  {selectedAlgo?.ai_params_schema ? (
                    <DynamicParamsForm
                      schema={selectedAlgo.ai_params_schema}
                      value={form.ai_params}
                      onChange={(params) => updateField('ai_params', params)}
                    />
                  ) : (
                    <Box p={3} bg="gray.50" borderRadius="md">
                      <Text color="gray.500" fontSize="sm">
                        {form.algo_package_id ? t('form.noParams') : t('form.selectAlgoFirst')}
                      </Text>
                    </Box>
                  )}
                </AccordionPanel>
              </AccordionItem>
            </Accordion>

            <Divider />

            {/* 区域划分 */}
            <Accordion allowToggle>
              <AccordionItem border="none">
                <AccordionButton px={0} _hover={{ bg: 'transparent' }}>
                  <Box flex="1" textAlign="left">
                    <Text fontSize="sm" fontWeight="600" color="gray.500">
                      {t('fields.regions')}
                      {(form.roi_regions.length + form.mark_regions.length + form.line_regions.length) > 0 &&
                        ` (${t('form.regionsSummary', { count: form.roi_regions.length + form.mark_regions.length + form.line_regions.length })})`}
                    </Text>
                  </Box>
                  <AccordionIcon />
                </AccordionButton>
                <AccordionPanel px={0} pb={4}>
                  {regionVideoError && (
                    <Alert status="warning" borderRadius="md" mb={3}>
                      <AlertIcon />
                      <AlertDescription>{regionVideoError}</AlertDescription>
                    </Alert>
                  )}
                  {regionVideoUrl && (
                    <Box position="absolute" w="1px" h="1px" opacity={0} pointerEvents="none" overflow="hidden">
                      <VideoPlayer
                        url={regionVideoUrl}
                        videoRef={videoRef}
                        fallbackConfig={{ showProtocol: false }}
                      />
                    </Box>
                  )}
                  <RegionCanvas
                    videoRef={videoRef}
                    roiRegions={form.roi_regions}
                    markRegions={form.mark_regions}
                    lineRegions={form.line_regions}
                    onChange={(data) => {
                      updateField('roi_regions', data.roi);
                      updateField('mark_regions', data.mark);
                      updateField('line_regions', data.line);
                    }}
                  />
                </AccordionPanel>
              </AccordionItem>
            </Accordion>
          </VStack>
        </ModalBody>

        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose}>{t('common:button.cancel')}</Button>
          <Button colorScheme="brand" onClick={handleSubmit} isLoading={submitting}>{t('actions.save')}</Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
