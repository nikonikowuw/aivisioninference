import {
  Button,
  FormControl,
  FormLabel,
  Input,
  Select,
  Switch,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Textarea,
  HStack,
  Checkbox,
  CheckboxGroup,
  Stack,
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  NumberIncrementStepper,
  NumberDecrementStepper,
  useToast,
  VStack,
} from '@chakra-ui/react';
import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { alertRuleApi, type AlertRule, type CreateAlertRuleRequest } from 'services/alertRule';

const METRIC_TYPES = [
  { value: 'cpu_usage', label: 'CPU Usage' },
  { value: 'memory_usage', label: 'Memory Usage' },
  { value: 'disk_usage', label: 'Disk Usage' },
  { value: 'temperature', label: 'Temperature' },
  { value: 'node_offline', label: 'Node Offline' },
  { value: 'node_error', label: 'Node Error' },
];

const OPERATORS = [
  { value: '>', label: '>' },
  { value: '>=', label: '>=' },
  { value: '<', label: '<' },
  { value: '<=', label: '<=' },
  { value: '==', label: '==' },
];

const NOTIFY_CHANNELS = [
  { value: 'webhook', label: 'Webhook' },
  { value: 'email', label: 'Email' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'dingtalk', label: 'DingTalk' },
  { value: 'feishu', label: 'Feishu' },
  { value: 'wecom', label: 'WeCom' },
];

interface AlertRuleFormProps {
  isOpen: boolean;
  onClose: () => void;
  rule?: AlertRule | null;
}

export default function AlertRuleForm({ isOpen, onClose, rule }: AlertRuleFormProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const toast = useToast();
  const isEdit = !!rule;

  const [name, setName] = useState('');
  const [metricType, setMetricType] = useState('cpu_usage');
  const [operator, setOperator] = useState('>');
  const [threshold, setThreshold] = useState(90);
  const [durationSeconds, setDurationSeconds] = useState(0);
  const [silenceMinutes, setSilenceMinutes] = useState(60);
  const [enabled, setEnabled] = useState(true);
  const [description, setDescription] = useState('');
  const [notifyChannels, setNotifyChannels] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (rule) {
      setName(rule.name);
      setMetricType(rule.metric_type);
      setOperator(rule.operator);
      setThreshold(rule.threshold);
      setDurationSeconds(rule.duration_seconds);
      setSilenceMinutes(rule.silence_minutes);
      setEnabled(rule.enabled);
      setDescription(rule.description || '');
      setNotifyChannels(rule.notify_channels || []);
    } else {
      setName('');
      setMetricType('cpu_usage');
      setOperator('>');
      setThreshold(90);
      setDurationSeconds(0);
      setSilenceMinutes(60);
      setEnabled(true);
      setDescription('');
      setNotifyChannels([]);
    }
  }, [rule, isOpen]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const data: CreateAlertRuleRequest = {
        name,
        metric_type: metricType,
        operator,
        threshold,
        duration_seconds: durationSeconds,
        silence_minutes: silenceMinutes,
        enabled,
        description,
        notify_channels: notifyChannels,
      };

      if (isEdit && rule) {
        await alertRuleApi.update(rule.id, data);
        toast({ title: 'Rule updated', status: 'success' });
      } else {
        await alertRuleApi.create(data);
        toast({ title: 'Rule created', status: 'success' });
      }
      onClose();
    } catch {
      toast({ title: isEdit ? 'Failed to update rule' : 'Failed to create rule', status: 'error' });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="xl">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{isEdit ? 'Edit Alert Rule' : 'Create Alert Rule'}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack spacing="4">
            <FormControl isRequired>
              <FormLabel>Rule Name</FormLabel>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., High CPU Alert"
              />
            </FormControl>

            <FormControl isRequired>
              <FormLabel>Metric Type</FormLabel>
              <Select value={metricType} onChange={(e) => setMetricType(e.target.value)}>
                {METRIC_TYPES.map((mt) => (
                  <option key={mt.value} value={mt.value}>
                    {mt.label}
                  </option>
                ))}
              </Select>
            </FormControl>

            <HStack spacing="4" w="full">
              <FormControl isRequired>
                <FormLabel>Operator</FormLabel>
                <Select value={operator} onChange={(e) => setOperator(e.target.value)}>
                  {OPERATORS.map((op) => (
                    <option key={op.value} value={op.value}>
                      {op.label}
                    </option>
                  ))}
                </Select>
              </FormControl>

              <FormControl isRequired>
                <FormLabel>Threshold</FormLabel>
                <NumberInput value={threshold} onChange={(_, v) => setThreshold(v)} min={0} max={999999}>
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>
            </HStack>

            <HStack spacing="4" w="full">
              <FormControl>
                <FormLabel>Duration (seconds)</FormLabel>
                <NumberInput value={durationSeconds} onChange={(_, v) => setDurationSeconds(v)} min={0}>
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel>Silence (minutes)</FormLabel>
                <NumberInput value={silenceMinutes} onChange={(_, v) => setSilenceMinutes(v)} min={0}>
                  <NumberInputField />
                  <NumberInputStepper>
                    <NumberIncrementStepper />
                    <NumberDecrementStepper />
                  </NumberInputStepper>
                </NumberInput>
              </FormControl>
            </HStack>

            <FormControl>
              <FormLabel>Notification Channels</FormLabel>
              <CheckboxGroup value={notifyChannels} onChange={(v) => setNotifyChannels(v as string[])}>
                <Stack direction="row" flexWrap="wrap">
                  {NOTIFY_CHANNELS.map((ch) => (
                    <Checkbox key={ch.value} value={ch.value}>
                      {ch.label}
                    </Checkbox>
                  ))}
                </Stack>
              </CheckboxGroup>
            </FormControl>

            <FormControl>
              <FormLabel>Description</FormLabel>
              <Textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Optional description"
              />
            </FormControl>

            <FormControl display="flex" alignItems="center">
              <FormLabel mb="0">Enabled</FormLabel>
              <Switch isChecked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
            </FormControl>
          </VStack>
        </ModalBody>

        <ModalFooter>
          <Button variant="ghost" mr="3" onClick={onClose}>
            Cancel
          </Button>
          <Button colorScheme="blue" onClick={handleSave} isLoading={saving}>
            {isEdit ? 'Update' : 'Create'}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
