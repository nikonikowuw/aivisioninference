import {
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  Button,
} from '@chakra-ui/react';
import { useRef } from 'react';
import { useTranslation } from 'react-i18next';

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  isLoading?: boolean;
  colorScheme?: string;
}

export default function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText,
  cancelText,
  isLoading,
  colorScheme = 'red',
}: ConfirmDialogProps) {
  const { t: tCommon } = useTranslation('common');
  const cancelRef = useRef<HTMLButtonElement>(null);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !isLoading) {
      e.preventDefault();
      onConfirm();
    }
  };

  return (
    <AlertDialog
      isOpen={isOpen}
      onClose={isLoading ? () => undefined : onClose}
      leastDestructiveRef={cancelRef}
      closeOnEsc={!isLoading}
      closeOnOverlayClick={!isLoading}
      isCentered
    >
      <AlertDialogOverlay onKeyDown={handleKeyDown}>
        <AlertDialogContent>
          <AlertDialogHeader fontSize="lg" fontWeight="bold">
            {title}
          </AlertDialogHeader>
          <AlertDialogBody>{message}</AlertDialogBody>
          <AlertDialogFooter>
            <Button ref={cancelRef} onClick={onClose} isDisabled={isLoading}>
              {cancelText || tCommon('button.cancel')}
            </Button>
            <Button
              colorScheme={colorScheme}
              onClick={onConfirm}
              ml={3}
              isLoading={isLoading}
            >
              {confirmText || tCommon('button.confirm')}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogOverlay>
    </AlertDialog>
  );
}
