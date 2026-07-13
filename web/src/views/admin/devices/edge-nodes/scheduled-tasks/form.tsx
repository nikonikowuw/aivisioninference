import {
  Box,
  Button,
  Flex,
  FormControl,
  FormErrorMessage,
  FormLabel,
  HStack,
  Input,
  NumberInput,
  NumberInputField,
  Select,
  Text,
  Textarea,
  useColorModeValue,
  useToast,
  VStack,
} from '@chakra-ui/react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  edgeNodeScheduledTaskApi,
  type EdgeNodeScheduledTask,
  type ScheduledTaskRequest,
} from 'services/edgeNodeScheduledTask';

interface ScheduledTaskFormProps {
  nodeId: string;
  task: EdgeNodeScheduledTask | null; // null = create, non-null = edit
  onClose: () => void;
}

export default function ScheduledTaskForm({
  nodeId,
  task,
  onClose,
}: ScheduledTaskFormProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const toast = useToast();
  const textColor = useColorModeValue('secondaryGray.900', 'white');

  const [name, setName] = useState(task?.name || '');
  const [command, setCommand] = useState(task?.command || '');
  const [cronExpr, setCronExpr] = useState(task?.cron_expr || '');
  const [timeoutSeconds, setTimeoutSeconds] = useState(
    task?.timeout_seconds || 30,
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = useCallback((): boolean => {
    const newErrors: Record<string, string> = {};
    if (!name.trim()) {
      newErrors.name = t('validation.required');
    }
    if (!command.trim()) {
      newErrors.command = t('validation.required');
    }
    if (timeoutSeconds < 1 || timeoutSeconds > 3600) {
      newErrors.timeout = t('validation.invalidTimeout');
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [name, command, timeoutSeconds, t]);

  const handleSubmit = useCallback(async () => {
    if (!validate()) return;

    setIsSubmitting(true);
    try {
      const data: ScheduledTaskRequest = {
        name: name.trim(),
        command: command.trim(),
        timeout_seconds: timeoutSeconds,
      };
      if (cronExpr.trim()) {
        data.cron_expr = cronExpr.trim();
      }

      if (task) {
        // Update existing
        await edgeNodeScheduledTaskApi.update(nodeId, task.id, data);
        toast({
          title: t('message.updateSuccess'),
          status: 'success',
        });
      } else {
        // Create new
        await edgeNodeScheduledTaskApi.create(nodeId, data);
        toast({
          title: t('message.createSuccess'),
          status: 'success',
        });
      }
      onClose();
    } catch {
      toast({
        title: t('message.submitFailed'),
        status: 'error',
      });
    } finally {
      setIsSubmitting(false);
    }
  }, [
    validate,
    name,
    command,
    timeoutSeconds,
    cronExpr,
    task,
    nodeId,
    toast,
    t,
    onClose,
  ]);

  return (
    <Box>
      <Text color={textColor} fontSize="lg" fontWeight="bold" mb="20px">
        {task ? t('scheduledTasks.editTask') : t('scheduledTasks.createTask')}
      </Text>

      <VStack spacing="20px" align="stretch" maxW="600px">
        <FormControl isRequired isInvalid={!!errors.name}>
          <FormLabel>{t('scheduledTasks.taskName')}</FormLabel>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('scheduledTasks.taskNamePlaceholder')}
          />
          <FormErrorMessage>{errors.name}</FormErrorMessage>
        </FormControl>

        <FormControl isRequired isInvalid={!!errors.command}>
          <FormLabel>{t('scheduledTasks.command')}</FormLabel>
          <Textarea
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            placeholder={t('scheduledTasks.commandPlaceholder')}
            rows={4}
            fontFamily="mono"
          />
          <FormErrorMessage>{errors.command}</FormErrorMessage>
        </FormControl>

        <FormControl>
          <FormLabel>{t('scheduledTasks.cronExpr')}</FormLabel>
          <Input
            value={cronExpr}
            onChange={(e) => setCronExpr(e.target.value)}
            placeholder={t('scheduledTasks.cronPlaceholder')}
            fontFamily="mono"
          />
          <Text fontSize="xs" color="gray.500" mt="5px">
            {t('scheduledTasks.cronHint')}
          </Text>
        </FormControl>

        <FormControl isInvalid={!!errors.timeout}>
          <FormLabel>{t('scheduledTasks.timeout')}</FormLabel>
          <NumberInput
            value={timeoutSeconds}
            onChange={(_, val) => setTimeoutSeconds(val)}
            min={1}
            max={3600}
          >
            <NumberInputField />
          </NumberInput>
          <FormErrorMessage>{errors.timeout}</FormErrorMessage>
        </FormControl>

        <HStack spacing="10px" pt="10px">
          <Button
            colorScheme="brand"
            onClick={handleSubmit}
            isLoading={isSubmitting}
          >
            {task ? tCommon('button.save') : tCommon('button.create')}
          </Button>
          <Button variant="outline" onClick={onClose}>
            {tCommon('button.cancel')}
          </Button>
        </HStack>
      </VStack>
    </Box>
  );
}
