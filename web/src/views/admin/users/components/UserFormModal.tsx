import {
  Button,
  Checkbox,
  CheckboxGroup,
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
  Radio,
  RadioGroup,
  Stack,
  Text,
  useColorModeValue,
  Box
} from '@chakra-ui/react';
import AvatarUploader from 'components/avatar-upload/AvatarUploader';
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Role, User } from 'services/api';

interface UserFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: any) => Promise<void>;
  editingUser: User | null;
  allRoles: Role[];
  currentUser?: User | null;
  refreshUser: () => Promise<void>;
}

export const UserFormModal: React.FC<UserFormModalProps> = ({
  isOpen,
  onClose,
  onSave,
  editingUser,
  allRoles,
  currentUser,
  refreshUser
}) => {
  const { t } = useTranslation('modules/users');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');

  const [form, setForm] = useState({
    username: '',
    display_name: '',
    email: '',
    password: '',
    status: 1,
    role_ids: [] as string[],
  });
  const [avatarUrl, setAvatarUrl] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setForm({
        username: editingUser?.username || '',
        display_name: editingUser?.display_name || '',
        email: editingUser?.email || '',
        password: '',
        status: editingUser?.status ?? 1,
        role_ids: editingUser?.roles?.map((r) => r.id) || [],
      });
      setAvatarUrl(editingUser?.avatar_url || '');
    }
  }, [isOpen, editingUser]);

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await onSave({ ...form, avatar_url: avatarUrl });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg">
      <ModalOverlay backdropFilter="blur(4px)" />
      <ModalContent borderRadius="20px">
        <ModalHeader fontSize="22px" fontWeight="800" color={textColor} pt="25px" px="25px">
          {editingUser ? t('modal.editTitle') : t('modal.createTitle')}
        </ModalHeader>
        <ModalCloseButton top="25px" right="25px" />
        <ModalBody px="25px" pb="25px">
          {editingUser && (
            <Box textAlign="center" mb={4}>
              <AvatarUploader
                value={avatarUrl}
                onChange={async (url) => {
                  setAvatarUrl(url);
                  if (editingUser.id === currentUser?.id) {
                    await refreshUser();
                  }
                }}
                userId={editingUser.id}
                name={form.display_name || form.username}
                size={80}
              />
            </Box>
          )}
          <FormControl isRequired mb="24px">
            <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
              {t('form.username.label')}
            </FormLabel>
            <Input
              variant="auth"
              fontSize="sm"
              type="text"
              placeholder={t('form.username.placeholder')}
              fontWeight="500"
              size="lg"
              h="50px"
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
            />
          </FormControl>
          <FormControl mb="24px">
            <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
              {t('form.displayName.label')}
            </FormLabel>
            <Input
              variant="auth"
              fontSize="sm"
              type="text"
              placeholder={t('form.displayName.placeholder')}
              fontWeight="500"
              size="lg"
              h="50px"
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
            />
          </FormControl>
          <FormControl isRequired mb="24px">
            <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
              {t('form.email.label')}
            </FormLabel>
            <Input
              variant="auth"
              fontSize="sm"
              type="email"
              placeholder={t('form.email.placeholder')}
              fontWeight="500"
              size="lg"
              h="50px"
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </FormControl>
          <FormControl isRequired={!editingUser} mb="24px">
            <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
              {t('form.password.label')}
              {editingUser && <Text as="span" fontWeight="400" ms="1">（{t('form.password.hint')}）</Text>}
            </FormLabel>
            <Input
              variant="auth"
              fontSize="sm"
              type="password"
              placeholder={t('form.password.placeholder')}
              fontWeight="500"
              size="lg"
              h="50px"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
          </FormControl>
          <FormControl mb="24px">
            <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
              {t('table.columns.roles')}
            </FormLabel>
            <CheckboxGroup
              colorScheme="brand"
              value={form.role_ids}
              onChange={(values) => setForm({ ...form, role_ids: values as string[] })}
            >
              <Stack spacing={[2, 4]} direction="row" wrap="wrap" p="4px">
                {allRoles.map((role) => (
                  <Checkbox key={role.id} value={role.id} fontWeight="500" fontSize="sm">
                    {role.name}
                  </Checkbox>
                ))}
              </Stack>
            </CheckboxGroup>
          </FormControl>
          {!editingUser && (
            <FormControl mb="24px">
              <FormLabel ms="4px" fontSize="sm" fontWeight="700" color={textColor}>
                {t('form.status.label')}
              </FormLabel>
              <RadioGroup
                onChange={(val) => setForm({ ...form, status: Number(val) })}
                value={String(form.status)}
              >
                <Stack direction="row" spacing={5} ms="4px">
                  <Radio value="1" colorScheme="green">
                    <Text fontSize="sm" fontWeight="500">{t('form.status.active')}</Text>
                  </Radio>
                  <Radio value="0" colorScheme="red">
                    <Text fontSize="sm" fontWeight="500">{t('form.status.inactive')}</Text>
                  </Radio>
                </Stack>
              </RadioGroup>
            </FormControl>
          )}
        </ModalBody>
        <ModalFooter pb="25px" px="25px">
          <Button
            variant="no-effects"
            mr={3}
            onClick={onClose}
            fontWeight="600"
            fontSize="sm"
            h="44px"
          >
            {tCommon('button.cancel')}
          </Button>
          <Button
            variant="brand"
            onClick={handleSave}
            isLoading={isSaving}
            fontWeight="600"
            fontSize="sm"
            h="44px"
            px="24px"
          >
            {tCommon('button.save')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};
