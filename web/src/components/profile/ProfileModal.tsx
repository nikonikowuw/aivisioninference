import { useState, useEffect } from 'react';
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalCloseButton,
  Button,
  VStack,
  HStack,
  Input,
  Text,
  FormControl,
  FormLabel,
  FormErrorMessage,
  useToast,
  Divider,
  Box,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { authApi, type User } from 'services/api';
import { useAuth } from 'contexts/AuthContext';

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  user: User;
}

export default function ProfileModal({ isOpen, onClose, user }: ProfileModalProps) {
  const { t } = useTranslation();
  const toast = useToast();
  const { refreshUser } = useAuth();

  const [displayName, setDisplayName] = useState(user.display_name || '');
  const [email, setEmail] = useState(user.email || '');
  const [avatarUrl, setAvatarUrl] = useState(user.avatar_url || '');
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [profileLoading, setProfileLoading] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setDisplayName(user.display_name || '');
      setEmail(user.email || '');
      setAvatarUrl(user.avatar_url || '');
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setShowPassword(false);
    }
  }, [isOpen, user]);

  const handleSaveProfile = async () => {
    setProfileLoading(true);
    try {
      const data: Record<string, string> = {};
      if (displayName !== (user.display_name || '')) data.display_name = displayName;
      if (email !== (user.email || '')) data.email = email;
      if (avatarUrl !== (user.avatar_url || '')) data.avatar_url = avatarUrl;
      if (Object.keys(data).length === 0) {
        onClose();
        return;
      }
      await authApi.updateProfile(data);
      await refreshUser();
      toast({
        title: t('common:profile.updateSuccess'),
        status: 'success',
        duration: 3000,
        isClosable: true,
      });
      onClose();
    } catch (err) {
      toast({
        title: err instanceof Error ? err.message : t('common:message.operationFailed'),
        status: 'error',
        duration: 3000,
        isClosable: true,
      });
    } finally {
      setProfileLoading(false);
    }
  };

  const handleChangePassword = async () => {
    if (newPassword !== confirmPassword) {
      toast({
        title: t('common:profile.passwordMismatch'),
        status: 'warning',
        duration: 3000,
        isClosable: true,
      });
      return;
    }
    if (newPassword.length < 6) {
      toast({
        title: t('common:profile.passwordTooShort'),
        status: 'warning',
        duration: 3000,
        isClosable: true,
      });
      return;
    }

    setPasswordLoading(true);
    try {
      await authApi.changePassword(oldPassword, newPassword);
      toast({
        title: t('common:profile.passwordUpdateSuccess'),
        status: 'success',
        duration: 3000,
        isClosable: true,
      });
      setOldPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setShowPassword(false);
    } catch (err) {
      toast({
        title: err instanceof Error ? err.message : t('common:message.operationFailed'),
        status: 'error',
        duration: 3000,
        isClosable: true,
      });
    } finally {
      setPasswordLoading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size={{ base: 'full', md: 'md' }}>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{t('common:profile.title')}</ModalHeader>
        <ModalCloseButton />
        <ModalBody pb={6}>
          <VStack spacing={4} align="stretch">
            <FormControl>
              <FormLabel>{t('common:profile.displayName')}</FormLabel>
              <Input
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
              />
            </FormControl>

            <FormControl>
              <FormLabel>{t('common:profile.email')}</FormLabel>
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </FormControl>

            <FormControl>
              <FormLabel>{t('common:profile.avatarUrl')}</FormLabel>
              <Input
                value={avatarUrl}
                onChange={(e) => setAvatarUrl(e.target.value)}
                placeholder="https://..."
              />
            </FormControl>

            <HStack justify="flex-end">
              <Button
                colorScheme="blue"
                onClick={handleSaveProfile}
                isLoading={profileLoading}
              >
                {t('common:button.save')}
              </Button>
            </HStack>

            <Divider />

            <Box>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowPassword(!showPassword)}
                aria-label={t('common:profile.changePassword')}
              >
                {t('common:profile.changePassword')}
              </Button>

              {showPassword && (
                <VStack spacing={3} mt={3} align="stretch">
                  <FormControl>
                    <FormLabel>{t('common:profile.currentPassword')}</FormLabel>
                    <Input
                      type="password"
                      value={oldPassword}
                      onChange={(e) => setOldPassword(e.target.value)}
                      isDisabled={passwordLoading}
                    />
                  </FormControl>

                  <FormControl>
                    <FormLabel>{t('common:profile.newPassword')}</FormLabel>
                    <Input
                      type="password"
                      value={newPassword}
                      onChange={(e) => setNewPassword(e.target.value)}
                      isDisabled={passwordLoading}
                    />
                  </FormControl>

                  <FormControl isInvalid={confirmPassword.length > 0 && newPassword !== confirmPassword}>
                    <FormLabel>{t('common:profile.confirmPassword')}</FormLabel>
                    <Input
                      type="password"
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      isDisabled={passwordLoading}
                    />
                    {confirmPassword.length > 0 && newPassword !== confirmPassword && (
                      <FormErrorMessage>
                        {t('common:profile.passwordMismatch')}
                      </FormErrorMessage>
                    )}
                  </FormControl>

                  <HStack justify="flex-end">
                    <Button
                      colorScheme="blue"
                      variant="outline"
                      onClick={handleChangePassword}
                      isLoading={passwordLoading}
                    >
                      {t('common:profile.changePassword')}
                    </Button>
                  </HStack>
                </VStack>
              )}
            </Box>
          </VStack>
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}
