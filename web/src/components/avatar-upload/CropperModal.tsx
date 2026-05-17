import { useCallback, useRef } from 'react';
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  Button,
  Box,
  useToast,
} from '@chakra-ui/react';
import Cropper, { type Area } from 'react-easy-crop';
import { useTranslation } from 'react-i18next';

interface CropperModalProps {
  isOpen: boolean;
  onClose: () => void;
  imageSrc: string;
  onCropComplete: (file: File) => void;
}

export default function CropperModal({ isOpen, onClose, imageSrc, onCropComplete }: CropperModalProps) {
  const { t } = useTranslation('common');
  const toast = useToast();
  const cropAreaRef = useRef<Area | null>(null);

  const onCropChange = useCallback((_: unknown, croppedArea: Area) => {
    cropAreaRef.current = croppedArea;
  }, []);

  const handleConfirm = useCallback(async () => {
    if (!cropAreaRef.current) return;

    const area = cropAreaRef.current;
    const img = new Image();
    img.src = imageSrc;

    await new Promise<void>((resolve) => {
      img.onload = () => resolve();
    });

    const canvas = document.createElement('canvas');
    const size = Math.min(area.width, area.height, 256);
    canvas.width = size;
    canvas.height = size;

    const ctx = canvas.getContext('2d');
    if (!ctx) {
      toast({ title: t('message.operationFailed'), status: 'error' });
      return;
    }

    const scaleX = img.naturalWidth / img.width;
    const scaleY = img.naturalHeight / img.height;

    ctx.drawImage(
      img,
      area.x * scaleX,
      area.y * scaleY,
      area.width * scaleX,
      area.height * scaleY,
      0,
      0,
      size,
      size,
    );

    canvas.toBlob(
      (blob) => {
        if (!blob) {
          toast({ title: t('message.operationFailed'), status: 'error' });
          return;
        }
        const file = new File([blob], 'avatar.jpg', { type: 'image/jpeg' });
        onCropComplete(file);
        onClose();
      },
      'image/jpeg',
      0.9,
    );
  }, [imageSrc, onCropComplete, onClose, toast, t]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{t('profile.avatarCrop')}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <Box position="relative" w="100%" h="300px">
            <Cropper
              image={imageSrc}
              cropShape="round"
              aspect={1}
              onCropComplete={onCropChange}
            />
          </Box>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose}>
            {t('button.cancel')}
          </Button>
          <Button colorScheme="blue" onClick={handleConfirm}>
            {t('button.confirm')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}