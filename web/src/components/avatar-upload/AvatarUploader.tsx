import { useState, useCallback } from 'react';
import {
  Avatar,
  Box,
  VStack,
  Text,
  Spinner,
  useColorModeValue,
  Icon,
  useToast,
} from '@chakra-ui/react';
import { useDropzone } from 'react-dropzone';
import { useTranslation } from 'react-i18next';
import { FiUpload } from 'react-icons/fi';
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
  size = 96,
  disabled = false,
  userId,
  name,
}: AvatarUploaderProps) {
  const { t } = useTranslation('common');
  const toast = useToast();
  const [cropSrc, setCropSrc] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const borderColor = useColorModeValue('gray.200', 'gray.600');
  const hoverBg = useColorModeValue('gray.50', 'gray.700');

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
        if (userId) {
          result = await usersApi.uploadAvatar(userId, file);
        } else {
          result = await authApi.uploadAvatar(file);
        }
        onChange(result.avatar_url);
      } catch (err) {
        toast({
          title: err instanceof Error ? err.message : t('message.operationFailed'),
          status: 'error',
        });
      } finally {
        setUploading(false);
      }
    },
    [userId, onChange, toast, t],
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
          border="2px dashed"
          borderColor={borderColor}
          cursor={disabled ? 'not-allowed' : 'pointer'}
          _hover={disabled ? {} : { bg: hoverBg }}
          overflow="hidden"
          transition="all 0.2s"
        >
          <input {...getInputProps()} />
          {value ? (
            <Avatar
              src={value}
              name={name}
              size="full"
              w={size}
              h={size}
              opacity={uploading ? 0.5 : 1}
            />
          ) : (
            <VStack
              justify="center"
              align="center"
              h="100%"
              spacing={0}
            >
              {uploading ? (
                <Spinner size="sm" color="blue.500" />
              ) : (
                <Icon as={FiUpload} boxSize={4} color="gray.400" />
              )}
              <Text fontSize="xs" color="gray.500" textAlign="center" px={1}>
                {t('profile.avatarDragHint')}
              </Text>
            </VStack>
          )}
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