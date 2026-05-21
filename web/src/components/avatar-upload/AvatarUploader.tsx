import { useState, useCallback, useRef, useEffect } from 'react';
import {
  Avatar,
  Box,
  VStack,
  Text,
  Spinner,
  useColorModeValue,
  Icon,
  useToast,
  Flex,
} from '@chakra-ui/react';
import { useDropzone } from 'react-dropzone';
import { useTranslation } from 'react-i18next';
import { FiCamera } from 'react-icons/fi';
import { authApi, usersApi } from 'services/api';
import CropperModal from './CropperModal';

const MAX_SIZE = 2 * 1024 * 1024;
const ALLOWED_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];

interface AvatarUploaderProps {
  value?: string;
  onChange: (url: string) => void;
  size?: number;
  disabled?: boolean;
  userId?: string;
  name?: string;
}

export default function AvatarUploader({
  value,
  onChange,
  size = 110,
  disabled = false,
  userId,
  name,
}: AvatarUploaderProps) {
  const { t } = useTranslation('common');
  const toast = useToast();
  const [cropSrc, setCropSrc] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const borderColor = useColorModeValue('gray.200', 'gray.600');
  const overlayBg = useColorModeValue('blackAlpha.600', 'blackAlpha.700');

  const userIdRef = useRef(userId);
  useEffect(() => {
    userIdRef.current = userId;
  }, [userId]);

  const onDrop = useCallback(
    (acceptedFiles: File[], fileRejections: unknown[]) => {
      if (fileRejections && Array.isArray(fileRejections) && fileRejections.length > 0) {
        const file = (fileRejections[0] as { file: File; errors: { code: string }[] }).file;
        const errors = (fileRejections[0] as { errors: { code: string }[] }).errors;
        if (errors.some((e: { code: string }) => e.code === 'file-too-large')) {
          toast({ title: t('profile.avatarFileTooLarge'), status: 'error' });
          return;
        }
      }

      const file = acceptedFiles[0];
      if (!file) return;

      if (!ALLOWED_TYPES.includes(file.type)) {
        toast({ title: t('profile.avatarInvalidFormat'), status: 'error' });
        return;
      }

      const reader = new FileReader();
      reader.onload = () => {
        setCropSrc(reader.result as string);
      };
      reader.readAsDataURL(file);
    },
    [t, toast],
  );

  const { getRootProps, getInputProps } = useDropzone({
    onDrop,
    accept: {
      'image/jpeg': ['.jpg', '.jpeg'],
      'image/png': ['.png'],
      'image/gif': ['.gif'],
      'image/webp': ['.webp'],
    },
    maxSize: MAX_SIZE,
    disabled: disabled || uploading,
    multiple: false,
  });

  const handleCropComplete = useCallback(
    async (file: File) => {
      setUploading(true);
      try {
        let result: { avatar_url: string };
        const currentUserId = userIdRef.current;
        if (currentUserId) {
          result = await usersApi.uploadAvatar(currentUserId, file);
        } else {
          result = await authApi.uploadAvatar(file);
        }
        onChange(result.avatar_url);
        toast({ title: t('profile.avatarUpdateSuccess'), status: 'success' });
      } catch (err) {
        toast({
          title: err instanceof Error ? err.message : t('message.operationFailed'),
          status: 'error',
        });
      } finally {
        setUploading(false);
      }
    },
    [onChange, toast, t],
  );

  return (
    <>
      <VStack spacing={2}>
        <Box
          {...getRootProps()}
          position="relative"
          w={`${size}px`}
          h={`${size}px`}
          borderRadius="full"
          cursor={disabled ? 'not-allowed' : 'pointer'}
          transition="all 0.3s"
          _hover={{
            transform: 'scale(1.02)',
            boxShadow: 'xl',
          }}
          border="4px solid"
          borderColor={useColorModeValue('white', 'navy.700')}
          boxShadow="lg"
          overflow="hidden"
          bg={useColorModeValue('gray.100', 'navy.800')}
        >
          <input {...getInputProps()} />
          <Avatar
            src={value}
            name={name}
            w="100%"
            h="100%"
            opacity={uploading ? 0.5 : 1}
            borderRadius="full"
          />

          <Flex
            position="absolute"
            top="0"
            left="0"
            w="100%"
            h="100%"
            bg={overlayBg}
            opacity="0"
            transition="opacity 0.2s"
            _hover={{ opacity: 1 }}
            justify="center"
            align="center"
            flexDirection="column"
          >
            {uploading ? (
              <Spinner size="md" color="white" />
            ) : (
              <>
                <Icon as={FiCamera} boxSize={6} color="white" mb={1} />
                <Text fontSize="xs" color="white" fontWeight="bold">
                  {t('profile.avatarUpdate')}
                </Text>
              </>
            )}
          </Flex>
        </Box>
      </VStack>

      {cropSrc && (
        <CropperModal
          isOpen={!!cropSrc}
          onClose={() => setCropSrc(null)}
          imageSrc={cropSrc}
          onCropComplete={handleCropComplete}
        />
      )}
    </>
  );
}