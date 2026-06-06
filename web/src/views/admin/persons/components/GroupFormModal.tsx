import {
  Button,
  FormControl,
  FormLabel,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  NumberInput,
  NumberInputField,
  Textarea,
  useColorModeValue,
} from '@chakra-ui/react';
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { PersonGroup } from 'services/api';

interface GroupFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: any) => Promise<void>;
  editingGroup: PersonGroup | null;
}

export const GroupFormModal: React.FC<GroupFormModalProps> = ({
  isOpen,
  onClose,
  onSave,
  editingGroup,
}) => {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');

  const [form, setForm] = useState({
    group_name: '',
    description: '',
    sort_order: 0,
  });
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setForm({
        group_name: editingGroup?.group_name || '',
        description: editingGroup?.description || '',
        sort_order: editingGroup?.sort_order || 0,
      });
    }
  }, [isOpen, editingGroup]);

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await onSave(form);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <ModalOverlay backdropFilter="blur(4px)" />
      <ModalContent borderRadius="20px">
        <ModalHeader fontSize="22px" fontWeight="800" color={textColor} pt="25px" px="25px">
          {editingGroup ? t('groups.edit') : t('groups.create')}
        </ModalHeader>
        <ModalCloseButton top="25px" right="25px" />
        <ModalBody px="25px" pb="25px">
          <FormControl isRequired mb="24px">
            <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
              {t('groups.form.groupName.label')}
            </FormLabel>
            <Input
              variant="auth"
              fontSize="sm"
              placeholder={t('groups.form.groupName.placeholder')}
              value={form.group_name}
              onChange={(e) => setForm({ ...form, group_name: e.target.value })}
            />
          </FormControl>

          <FormControl mb="24px">
            <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
              {t('groups.form.description.label')}
            </FormLabel>
            <Textarea
              fontSize="sm"
              placeholder={t('groups.form.description.placeholder')}
              borderRadius="16px"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
          </FormControl>

          <FormControl mb="24px">
            <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
              {t('groups.form.sortOrder.label')}
            </FormLabel>
            <NumberInput
              variant="auth"
              value={form.sort_order}
              onChange={(_, val) => setForm({ ...form, sort_order: val })}
            >
              <NumberInputField fontSize="sm" placeholder={t('groups.form.sortOrder.placeholder')} />
            </NumberInput>
          </FormControl>
        </ModalBody>
        <ModalFooter px="25px" pb="25px">
          <Button variant="ghost" mr={3} onClick={onClose}>
            {tCommon('button.cancel')}
          </Button>
          <Button
            variant="brand"
            isLoading={isSaving}
            onClick={handleSave}
            isDisabled={!form.group_name}
          >
            {tCommon('button.save')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};
