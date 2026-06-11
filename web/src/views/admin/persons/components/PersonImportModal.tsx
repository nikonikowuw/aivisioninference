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
  useColorModeValue,
  Text,
  VStack,
  Icon,
  useToast,
  Checkbox,
  Box,
  UnorderedList,
  ListItem,
  Alert,
  AlertIcon,
} from '@chakra-ui/react';
import React, { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { MdCloudUpload } from 'react-icons/md';
import { personImportsApi, chunkedUpload } from 'services/api';

interface PersonImportModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const PersonImportModal: React.FC<PersonImportModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
}) => {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const toast = useToast();

  const [file, setFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [overwrite, setOverwrite] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      setFile(e.target.files[0]);
    }
  };

  const handleUpload = async () => {
    if (!file) return;
    setIsUploading(true);
    setUploadProgress(0);
    try {
      // 1. 通过分片上传将文件传到服务器
      const filePath = await chunkedUpload(file, 3, setUploadProgress);
      // 2. 使用文件路径创建导入任务
      await personImportsApi.createByUrl(filePath, overwrite);
      toast({
        title: t('message.importSuccess'),
        status: 'success',
        duration: 5000,
        isClosable: true,
      });
      setFile(null);
      onSuccess();
      onClose();
    } catch (error: any) {
      toast({
        title: tCommon('message.operationFailed'),
        description: error.message,
        status: 'error',
        duration: 5000,
      });
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg">
      <ModalOverlay backdropFilter="blur(4px)" />
      <ModalContent borderRadius="20px">
        <ModalHeader fontSize="22px" fontWeight="800" color={textColor} pt="25px" px="25px">
          {t('actions.import')}
        </ModalHeader>
        <ModalCloseButton top="25px" right="25px" />
        <ModalBody px="25px" pb="25px">
          <VStack spacing={5} align="stretch">
            <Alert status="info" borderRadius="12px" fontSize="sm">
              <AlertIcon />
              <Box>
                <Text fontWeight="bold" mb={1}>{t('imports.uploadHint.title')}</Text>
                <UnorderedList spacing={1} pl={2}>
                  <ListItem>{t('imports.uploadHint.format')}</ListItem>
                  <ListItem>{t('imports.uploadHint.nameRule')}</ListItem>
                  <ListItem>{t('imports.uploadHint.codeRule')}</ListItem>
                  <ListItem>{t('imports.uploadHint.imageFormats')}</ListItem>
                </UnorderedList>
              </Box>
            </Alert>

            <Box
              w="100%"
              h="160px"
              border="2px dashed"
              borderColor={file ? 'brand.500' : 'gray.200'}
              bg={file ? 'brand.50' : 'transparent'}
              borderRadius="16px"
              display="flex"
              flexDirection="column"
              alignItems="center"
              justifyContent="center"
              cursor="pointer"
              onClick={() => fileInputRef.current?.click()}
              _hover={{ borderColor: 'brand.500', bg: 'gray.50' }}
              transition="all 0.2s"
            >
              <Input
                type="file"
                accept=".zip,.tar.gz,.tgz,.tar.bz2,.tbz2"
                display="none"
                ref={fileInputRef}
                onChange={handleFileChange}
              />
              <Icon as={MdCloudUpload} w="40px" h="40px" color="brand.500" mb={2} />
              <Text fontWeight="bold" color={textColor} textAlign="center" px={4} noOfLines={1}>
                {file ? file.name : t('imports.uploadHint.dropzone')}
              </Text>
              <Text fontSize="xs" color="gray.400" mt={1}>
                {t('imports.uploadHint.maxSize')}
              </Text>
            </Box>

            <FormControl display="flex" alignItems="start" flexDirection="column">
              <Checkbox
                id="overwrite"
                isChecked={overwrite}
                onChange={(e) => setOverwrite(e.target.checked)}
                colorScheme="brand"
                fontWeight="700"
                size="md"
              >
                {t('imports.overwrite')}
              </Checkbox>
              <Text fontSize="xs" color="gray.500" ml={6} mt={1}>
                {t('imports.overwriteHint')}
              </Text>
            </FormControl>
          </VStack>
        </ModalBody>
        <ModalFooter px="25px" pb="25px">
          <Button variant="ghost" mr={3} onClick={onClose} fontWeight="500">
            {tCommon('button.cancel')}
          </Button>
          <Button
            variant="brand"
            isLoading={isUploading}
            loadingText={uploadProgress > 0 && uploadProgress < 100 ? `${uploadProgress}%` : undefined}
            onClick={handleUpload}
            isDisabled={!file}
            fontWeight="500"
            px="8"
          >
            {tCommon('button.confirm')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};
