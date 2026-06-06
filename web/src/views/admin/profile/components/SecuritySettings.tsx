import { useState } from 'react';
import {
  Box,
  Button,
  FormControl,
  FormLabel,
  FormErrorMessage,
  Input,
  VStack,
  Text,
  useToast,
  useColorModeValue,
  Icon,
  InputGroup,
  InputRightElement,
  IconButton,
  HStack,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { FiLock, FiEye, FiEyeOff, FiSave } from 'react-icons/fi';
import { authApi } from 'services/api';
import Card from 'components/card/Card';

export default function SecuritySettings() {
  const { t } = useTranslation();
  const toast = useToast();

  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [showOld, setShowOld] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const secondaryColor = useColorModeValue('gray.600', 'gray.400');
  const inputBg = useColorModeValue('secondaryGray.300', 'navy.900');

  const handleUpdatePassword = async () => {
    if (!oldPassword || !newPassword || !confirmPassword) {
      toast({
        title: t('common:profile.passwordRequired'),
        status: 'warning',
        duration: 3000,
        isClosable: true,
      });
      return;
    }

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

    setLoading(true);
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
    } catch (err) {
      toast({
        title: err instanceof Error ? err.message : t('common:message.operationFailed'),
        status: 'error',
        duration: 3000,
        isClosable: true,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <VStack spacing={5} align="stretch" w="100%">
      <FormControl isRequired>
        <FormLabel fontSize="sm" fontWeight="700" color={textColor} mb="8px">
          {t('common:profile.currentPassword')}
        </FormLabel>
        <InputGroup size="lg">
          <Input
            isRequired={true}
            fontSize="sm"
            variant="auth"
            type={showOld ? 'text' : 'password'}
            placeholder="********"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            bg={inputBg}
            borderRadius="16px"
          />
          <InputRightElement display="flex" alignItems="center" mt="4px">
            <IconButton
              aria-label={showOld ? t('common:profile.hidePassword') : t('common:profile.showPassword')}
              variant="ghost"
              onClick={() => setShowOld(!showOld)}
              icon={<Icon color={secondaryColor} as={showOld ? FiEyeOff : FiEye} />}
            />
          </InputRightElement>
        </InputGroup>
      </FormControl>

      <FormControl isRequired>
        <FormLabel fontSize="sm" fontWeight="700" color={textColor} mb="8px">
          {t('common:profile.newPassword')}
        </FormLabel>
        <InputGroup size="lg">
          <Input
            isRequired={true}
            fontSize="sm"
            variant="auth"
            type={showNew ? 'text' : 'password'}
            placeholder="********"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            bg={inputBg}
            borderRadius="16px"
          />
          <InputRightElement display="flex" alignItems="center" mt="4px">
            <IconButton
              aria-label={showNew ? t('common:profile.hidePassword') : t('common:profile.showPassword')}
              variant="ghost"
              onClick={() => setShowNew(!showNew)}
              icon={<Icon color={secondaryColor} as={showNew ? FiEyeOff : FiEye} />}
            />
          </InputRightElement>
        </InputGroup>
      </FormControl>

      <FormControl isRequired isInvalid={confirmPassword.length > 0 && newPassword !== confirmPassword}>
        <FormLabel fontSize="sm" fontWeight="700" color={textColor} mb="8px">
          {t('common:profile.confirmPassword')}
        </FormLabel>
        <InputGroup size="lg">
          <Input
            isRequired={true}
            fontSize="sm"
            variant="auth"
            type={showConfirm ? 'text' : 'password'}
            placeholder="********"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            bg={inputBg}
            borderRadius="16px"
          />
          <InputRightElement display="flex" alignItems="center" mt="4px">
            <IconButton
              aria-label={showConfirm ? t('common:profile.hidePassword') : t('common:profile.showPassword')}
              variant="ghost"
              onClick={() => setShowConfirm(!showConfirm)}
              icon={<Icon color={secondaryColor} as={showConfirm ? FiEyeOff : FiEye} />}
            />
          </InputRightElement>
        </InputGroup>
        {confirmPassword.length > 0 && newPassword !== confirmPassword && (
          <FormErrorMessage fontSize="sm">
            {t('common:profile.passwordMismatch')}
          </FormErrorMessage>
        )}
      </FormControl>

      <Button
        variant="brand"
        size="lg"
        fontWeight="500"
        w="100%"
        onClick={handleUpdatePassword}
        isLoading={loading}
        leftIcon={<Icon as={FiSave} />}
        borderRadius="16px"
        mt={4}
      >
        {t('common:profile.updatePassword')}
      </Button>
    </VStack>
  );
}
