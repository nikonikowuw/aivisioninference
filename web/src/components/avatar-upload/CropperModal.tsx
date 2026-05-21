import { useCallback, useRef, useState } from 'react';
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
  Text,
  VStack,
} from '@chakra-ui/react';
import Cropper, { type Area, type Point } from 'react-easy-crop';
import { useTranslation } from 'react-i18next';

interface CropperModalProps {
  isOpen: boolean;
  onClose: () => void;
  imageSrc: string;
  onCropComplete: (file: File) => void;
}

async function getCroppedImg(imageSrc: string, pixelCrop: Area): Promise<File | null> {
  const image = new Image();
  image.src = imageSrc;
  await new Promise<void>((resolve) => { image.onload = () => resolve(); });

  const canvas = document.createElement('canvas');
  canvas.width = pixelCrop.width;
  canvas.height = pixelCrop.height;

  const ctx = canvas.getContext('2d');
  if (!ctx) return null;

  ctx.drawImage(
    image,
    pixelCrop.x,
    pixelCrop.y,
    pixelCrop.width,
    pixelCrop.height,
    0,
    0,
    pixelCrop.width,
    pixelCrop.height,
  );

  return new Promise((resolve) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          resolve(null);
          return;
        }
        resolve(new File([blob], 'avatar.jpg', { type: 'image/jpeg' }));
      },
      'image/jpeg',
      0.92,
    );
  });
}

export default function CropperModal({ isOpen, onClose, imageSrc, onCropComplete }: CropperModalProps) {
  const { t } = useTranslation('common');
  const toast = useToast();
  const [crop, setCrop] = useState<Point>({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const croppedAreaPixelsRef = useRef<Area>({ x: 0, y: 0, width: 0, height: 0 });

  const onCropAreaChange = useCallback((_croppedArea: Area, croppedAreaPixels: Area) => {
    croppedAreaPixelsRef.current = croppedAreaPixels;
  }, []);

  const handleConfirm = useCallback(async () => {
    const pixelCrop = croppedAreaPixelsRef.current;
    if (!pixelCrop || pixelCrop.width === 0 || pixelCrop.height === 0) {
      toast({ title: t('message.operationFailed'), status: 'error' });
      return;
    }

    const file = await getCroppedImg(imageSrc, pixelCrop);
    if (!file) {
      toast({ title: t('message.operationFailed'), status: 'error' });
      return;
    }

    onCropComplete(file);
    onClose();
  }, [imageSrc, onCropComplete, onClose, toast, t]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg" isCentered motionPreset="slideInBottom">
      <ModalOverlay backdropFilter="blur(8px)" />
      <ModalContent borderRadius="2xl">
        <ModalHeader borderBottomWidth="1px" fontSize="lg" fontWeight="bold">
          {t('profile.avatarCrop')}
        </ModalHeader>
        <ModalCloseButton />
        <ModalBody py={6}>
          <VStack spacing={4}>
            <Box
              position="relative"
              w="100%"
              h="350px"
              borderRadius="xl"
              overflow="hidden"
              borderWidth="1px"
            >
              <Cropper
                image={imageSrc}
                crop={crop}
                zoom={zoom}
                aspect={1}
                cropShape="round"
                onCropChange={setCrop}
                onZoomChange={setZoom}
                onCropAreaChange={onCropAreaChange}
              />
            </Box>
            <Text fontSize="sm" color="gray.500">
              {t('profile.avatarCropHint')}
            </Text>
          </VStack>
        </ModalBody>
        <ModalFooter borderTopWidth="1px" gap={3}>
          <Button variant="ghost" onClick={onClose} borderRadius="lg">
            {t('button.cancel')}
          </Button>
          <Button colorScheme="brand" onClick={handleConfirm} borderRadius="lg" px={8}>
            {t('button.confirm')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
