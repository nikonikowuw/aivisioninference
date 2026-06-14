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
  Input,
  VStack,
  useToast,
  HStack,
  IconButton,
  Text,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon } from '@chakra-ui/icons';
import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { aiTimeSchedulesApi, type AITimeSchedule } from 'services/api';

interface ScheduleFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  initialData?: AITimeSchedule | null;
}

interface TimeWindow {
  start: string;
  end: string;
}

interface FormData {
  name: string;
  description: string;
  start_date: string;
  end_date: string;
  time_windows: TimeWindow[];
}

const defaultForm: FormData = {
  name: '',
  description: '',
  start_date: '',
  end_date: '',
  time_windows: [{ start: '00:00', end: '23:59' }],
};

export default function ScheduleFormModal({ isOpen, onClose, onSuccess, initialData }: ScheduleFormModalProps) {
  const [form, setForm] = useState<FormData>(defaultForm);
  const [submitting, setSubmitting] = useState(false);
  const toast = useToast();
  const { t } = useTranslation(['modules/ai-time-schedules', 'common']);

  useEffect(() => {
    if (!isOpen) return;
    if (initialData) {
      setForm({
        name: initialData.name,
        description: initialData.description || '',
        start_date: initialData.start_date.substring(0, 10),
        end_date: initialData.end_date.substring(0, 10),
        time_windows: initialData.time_windows.length > 0 ? initialData.time_windows : [{ start: '00:00', end: '23:59' }],
      });
    } else {
      setForm({ ...defaultForm });
    }
  }, [isOpen, initialData]);

  const updateField = (key: keyof FormData, value: string) => {
    setForm(prev => ({ ...prev, [key]: value }));
  };

  const updateTimeWindow = (index: number, key: keyof TimeWindow, value: string) => {
    setForm(prev => {
      const tw = [...prev.time_windows];
      tw[index] = { ...tw[index], [key]: value };
      return { ...prev, time_windows: tw };
    });
  };

  const addTimeWindow = () => {
    setForm(prev => ({ ...prev, time_windows: [...prev.time_windows, { start: '00:00', end: '23:59' }] }));
  };

  const removeTimeWindow = (index: number) => {
    setForm(prev => {
      if (prev.time_windows.length <= 1) return prev;
      const tw = prev.time_windows.filter((_, i) => i !== index);
      return { ...prev, time_windows: tw };
    });
  };

  const handleSubmit = async () => {
    if (!form.name.trim()) { toast({ title: t('message.nameRequired'), status: 'warning' }); return; }
    if (!form.start_date) { toast({ title: t('message.startDateRequired'), status: 'warning' }); return; }
    if (!form.end_date) { toast({ title: t('message.endDateRequired'), status: 'warning' }); return; }

    setSubmitting(true);
    try {
      const payload = {
        name: form.name,
        description: form.description,
        start_date: form.start_date,
        end_date: form.end_date,
        time_windows: form.time_windows,
      };

      if (initialData) {
        await aiTimeSchedulesApi.update(initialData.id, payload);
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await aiTimeSchedulesApi.create(payload);
        toast({ title: t('message.createSuccess'), status: 'success' });
      }
      onSuccess();
    } catch (err: any) {
      toast({ title: t('common:message.operationFailed'), description: err?.message || '', status: 'error' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="xl" isCentered>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{initialData ? t('actions.edit') : t('actions.create')}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack spacing={4} align="stretch">
            <FormControl isRequired>
              <FormLabel>{t('fields.name')}</FormLabel>
              <Input value={form.name} onChange={e => updateField('name', e.target.value)} placeholder={t('form.namePlaceholder')} />
            </FormControl>

            <FormControl>
              <FormLabel>{t('fields.description')}</FormLabel>
              <Input value={form.description} onChange={e => updateField('description', e.target.value)} placeholder={t('form.descriptionPlaceholder')} />
            </FormControl>

            <HStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('form.startDate')}</FormLabel>
                <Input type="date" value={form.start_date} onChange={e => updateField('start_date', e.target.value)} />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('form.endDate')}</FormLabel>
                <Input type="date" value={form.end_date} onChange={e => updateField('end_date', e.target.value)} />
              </FormControl>
            </HStack>

            <FormControl isRequired>
              <FormLabel>{t('form.timeWindows')}</FormLabel>
              <VStack align="stretch" spacing={2}>
                {form.time_windows.map((tw, index) => (
                  <HStack key={index}>
                    <Input type="time" value={tw.start} onChange={e => updateTimeWindow(index, 'start', e.target.value)} />
                    <Text>-</Text>
                    <Input type="time" value={tw.end} onChange={e => updateTimeWindow(index, 'end', e.target.value)} />
                    <IconButton
                      aria-label={t('form.addTimeWindow')}
                      icon={<DeleteIcon />}
                      colorScheme="red"
                      variant="ghost"
                      size="sm"
                      isDisabled={form.time_windows.length <= 1}
                      onClick={() => removeTimeWindow(index)}
                    />
                  </HStack>
                ))}
                <Button leftIcon={<AddIcon />} size="sm" variant="outline" onClick={addTimeWindow}>
                  {t('form.addTimeWindow')}
                </Button>
              </VStack>
            </FormControl>
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
