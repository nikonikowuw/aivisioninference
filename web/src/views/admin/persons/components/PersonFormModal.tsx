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
  Box,
  SimpleGrid,
  Select,
  Image,
  Center,
  Icon,
} from '@chakra-ui/react';
import { MdFileUpload, MdImage } from 'react-icons/md';
import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Person, PersonGroup, chunkedUpload, getFileUrl } from 'services/api';

interface PersonFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (data: FormData | Record<string, unknown>) => Promise<void>;
  editingPerson: Person | null;
  allGroups: PersonGroup[];
}

export const PersonFormModal: React.FC<PersonFormModalProps> = ({
  isOpen,
  onClose,
  onSave,
  editingPerson,
  allGroups,
}) => {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const bgHover = useColorModeValue('gray.50', 'whiteAlpha.50');

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [form, setForm] = useState({
    person_name: '',
    person_code: '',
    phone: '',
    gender: 'unknown',
    enabled: true,
    group_ids: [] as string[],
  });
  
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);

  useEffect(() => {
    if (isOpen) {
      setForm({
        person_name: editingPerson?.person_name || '',
        person_code: editingPerson?.person_code || '',
        phone: editingPerson?.phone || '',
        gender: editingPerson?.gender || 'unknown',
        enabled: editingPerson?.enabled ?? true,
        group_ids: editingPerson?.groups?.map(g => g.id) || [],
      });
      setPreviewUrl(getFileUrl(editingPerson?.image_url) || '');
      setSelectedFile(null);
    }
  }, [isOpen, editingPerson]);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setSelectedFile(file);
      setPreviewUrl(URL.createObjectURL(file));
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    setUploadProgress(0);
    try {
      // 如果有新选择的文件，先通过分片上传获取路径
      let imageUrl = '';
      if (selectedFile) {
        imageUrl = await chunkedUpload(selectedFile, 3, setUploadProgress);
      }

      // 构造 JSON body 提交
      const payload: Record<string, unknown> = {
        person_name: form.person_name,
        person_code: form.person_code,
        phone: form.phone,
        gender: form.gender,
        enabled: form.enabled,
        group_ids: form.group_ids,
      };
      if (imageUrl) {
        payload.image_url = imageUrl;
      }
      
      await onSave(payload);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="xl">
      <ModalOverlay backdropFilter="blur(4px)" />
      <ModalContent borderRadius="20px">
        <ModalHeader fontSize="22px" fontWeight="800" color={textColor} pt="25px" px="25px">
          {editingPerson ? t('modal.editTitle') : t('modal.createTitle')}
        </ModalHeader>
        <ModalCloseButton top="25px" right="25px" />
        <ModalBody px="25px" pb="25px">
          <FormControl isRequired mb={6}>
            <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
              {t('form.image.label')}
            </FormLabel>
            <Center
              border="1px dashed"
              borderColor={borderColor}
              borderRadius="12px"
              p={2}
              minH="200px"
              cursor="pointer"
              transition="0.2s"
              _hover={{ bg: bgHover, borderColor: 'brand.500' }}
              onClick={() => fileInputRef.current?.click()}
              overflow="hidden"
              position="relative"
            >
              {previewUrl ? (
                <Image
                  src={previewUrl}
                  alt="Face Preview"
                  maxH="280px"
                  objectFit="contain"
                  borderRadius="8px"
                />
              ) : (
                <Stack align="center" spacing={2}>
                  <Icon as={MdImage} w={10} h={10} color="gray.400" />
                  <Text fontSize="sm" color="gray.500">{t('message.imageUploadHint')}</Text>
                </Stack>
              )}
              {previewUrl && (
                <Box
                  position="absolute"
                  bottom={2}
                  right={2}
                  bg="blackAlpha.600"
                  color="white"
                  p={1}
                  borderRadius="md"
                  fontSize="xs"
                >
                  {t('message.imageClickToReplace')}
                </Box>
              )}
            </Center>
            <input
              type="file"
              ref={fileInputRef}
              style={{ display: 'none' }}
              accept="image/*"
              onChange={handleFileChange}
            />
            <Text mt={2} fontSize="xs" color="gray.500">{t('form.image.hint')}</Text>
          </FormControl>

          <SimpleGrid columns={{ base: 1, md: 2 }} spacing={6}>
            <FormControl isRequired>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
                {t('table.columns.personName')}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                placeholder={t('form.personName.placeholder')}
                value={form.person_name}
                onChange={(e) => setForm({ ...form, person_name: e.target.value })}
              />
            </FormControl>

            <FormControl isRequired>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
                {t('table.columns.personCode')}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                placeholder={t('form.personCode.placeholder')}
                value={form.person_code}
                onChange={(e) => setForm({ ...form, person_code: e.target.value })}
              />
            </FormControl>

            <FormControl>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
                {t('table.columns.phone')}
              </FormLabel>
              <Input
                variant="auth"
                fontSize="sm"
                placeholder={t('form.phone.placeholder')}
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
              />
            </FormControl>

            <FormControl>
              <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
                {t('form.gender.label')}
              </FormLabel>
              <Select
                variant="auth"
                fontSize="sm"
                value={form.gender}
                onChange={(e) => setForm({ ...form, gender: e.target.value })}
              >
                <option value="unknown">{t('form.gender.unknown')}</option>
                <option value="male">{t('form.gender.male')}</option>
                <option value="female">{t('form.gender.female')}</option>
              </Select>
            </FormControl>
          </SimpleGrid>

          <FormControl mt={6}>
            <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
              {t('table.columns.groups')}
            </FormLabel>
            <CheckboxGroup
              colorScheme="brand"
              value={form.group_ids}
              onChange={(values) => setForm({ ...form, group_ids: values as string[] })}
            >
              <Stack direction="row" wrap="wrap" spacing={4}>
                {allGroups.map((g) => (
                  <Checkbox key={g.id} value={g.id}>
                    {g.group_name}
                  </Checkbox>
                ))}
              </Stack>
            </CheckboxGroup>
          </FormControl>

          <FormControl mt={6}>
            <FormLabel fontSize="sm" fontWeight="700" color={textColor}>
              {t('table.columns.enabled')}
            </FormLabel>
            <RadioGroup
              value={form.enabled ? '1' : '0'}
              onChange={(v) => setForm({ ...form, enabled: v === '1' })}
            >
              <Stack direction="row">
                <Radio value="1">{tCommon('status.enabled')}</Radio>
                <Radio value="0">{tCommon('status.disabled')}</Radio>
              </Stack>
            </RadioGroup>
          </FormControl>
        </ModalBody>
        <ModalFooter px="25px" pb="25px">
          <Button variant="ghost" mr={3} onClick={onClose}>
            {tCommon('button.cancel')}
          </Button>
          <Button
            variant="brand"
            isLoading={isSaving}
            loadingText={uploadProgress > 0 && uploadProgress < 100 ? `${uploadProgress}%` : undefined}
            onClick={handleSave}
            isDisabled={!form.person_name}
          >
            {tCommon('button.save')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};
