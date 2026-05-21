import { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Button,
  FormControl,
  FormLabel,
  Input,
  VStack,
  Text,
  useToast,
  useColorModeValue,
  Icon,
  InputGroup,
  InputLeftElement,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { FiUser, FiMail, FiCheck } from 'react-icons/fi';
import { authApi } from 'services/api';
import { useAuth } from 'contexts/AuthContext';
import AvatarUploader from 'components/avatar-upload/AvatarUploader';
import Card from 'components/card/Card';

export default function PersonalInfo() {
  const { t } = useTranslation();
  const toast = useToast();
  const { user, refreshUser } = useAuth();

  const [displayName, setDisplayName] = useState(user?.display_name || '');
  const [email, setEmail] = useState(user?.email || '');
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url || '');
  const [loading, setLoading] = useState(false);

  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const secondaryColor = useColorModeValue('gray.600', 'gray.400');
  const inputBg = useColorModeValue('secondaryGray.300', 'navy.900');

  useEffect(() => {
    if (user) {
      setDisplayName(user.display_name || '');
      setEmail(user.email || '');
      setAvatarUrl(user.avatar_url || '');
    }
  }, [user]);

  const hasChanges = useMemo(() => {
    return displayName !== (user?.display_name || '') || email !== (user?.email || '');
  }, [displayName, email, user]);

  const handleSave = async () => {
    if (!hasChanges) return;

    setLoading(true);
    try {
      const data: Record<string, string> = {};
      if (displayName !== (user?.display_name || '')) data.display_name = displayName;
      if (email !== (user?.email || '')) data.email = email;
      
      await authApi.updateProfile(data);
      await refreshUser();
      toast({
        title: t('common:profile.updateSuccess'),
        status: 'success',
        duration: 3000,
        isClosable: true,
      });
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
    <VStack spacing={8} align="center" w="100%">
      <AvatarUploader
        value={avatarUrl}
        onChange={async (url) => {
          setAvatarUrl(url);
          await refreshUser();
        }}
        name={user?.display_name || user?.username}
        size={120}
      />
      
      <VStack spacing={1} mb={4}>
        <Text fontSize="xl" fontWeight="bold" color={textColor}>
          {user?.display_name || user?.username}
        </Text>
        <Text fontSize="sm" color={secondaryColor}>
          @{user?.username}
        </Text>
      </VStack>

      <VStack spacing={5} align="stretch" w="100%">
        <FormControl>
          <FormLabel fontSize="sm" fontWeight="bold" color={textColor}>
            {t('common:profile.displayName')}
          </FormLabel>
          <InputGroup size="lg">
            <InputLeftElement pointerEvents="none">
              <Icon as={FiUser} color="gray.400" />
            </InputLeftElement>
            <Input
              variant="auth"
              bg={inputBg}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t('common:profile.displayNamePlaceholder')}
              borderRadius="16px"
            />
          </InputGroup>
        </FormControl>

        <FormControl>
          <FormLabel fontSize="sm" fontWeight="bold" color={textColor}>
            {t('common:profile.email')}
          </FormLabel>
          <InputGroup size="lg">
            <InputLeftElement pointerEvents="none">
              <Icon as={FiMail} color="gray.400" />
            </InputLeftElement>
            <Input
              type="email"
              variant="auth"
              bg={inputBg}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('common:profile.emailPlaceholder')}
              borderRadius="16px"
            />
          </InputGroup>
        </FormControl>

        <Button
          variant="brand"
          size="lg"
          leftIcon={<Icon as={FiCheck} />}
          onClick={handleSave}
          isLoading={loading}
          isDisabled={!hasChanges}
          borderRadius="16px"
          mt={4}
          w="full"
        >
          {t('common:button.save')}
        </Button>
      </VStack>
    </VStack>
  );
}
