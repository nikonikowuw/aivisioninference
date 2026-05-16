import {
  Box,
  Button,
  Flex,
  Text,
  useColorModeValue,
  useToast,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  Select,
  useDisclosure,
  Spinner,
  Center,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Badge,
  HStack,
} from '@chakra-ui/react';
import { AddIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState } from 'react';
import { permissionsApi, type Permission } from 'services/api';

export default function Permissions() {
  const { t } = useTranslation('modules/permissions');
  const { t: tCommon } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [tree, setTree] = useState<Permission[]>([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState({ name: '', code: '', type: 'menu', parent_id: '' });

  const loadTree = async () => {
    try {
      const data = await permissionsApi.tree();
      setTree(data);
    } catch {
      toast({ title: tCommon('message.loadFailed'), status: 'error' });
    }
  };

  useEffect(() => {
    loadTree().finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    try {
      await permissionsApi.create(form);
      toast({ title: t('message.createSuccess'), status: 'success' });
      onClose();
      loadTree();
    } catch (err) {
      toast({ title: t('message.createFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const leafBg = useColorModeValue('gray.50', 'whiteAlpha.50');

  const renderTree = (nodes: Permission[], depth = 0) =>
    nodes.map((p) => (
      <Box key={p.id} ml={depth * 4} mb={2}>
        {p.children && p.children.length > 0 ? (
          <Accordion allowToggle>
            <AccordionItem border="none">
              <AccordionButton px={2} py={2}>
                <Box flex="1" textAlign="left">
                  <HStack>
                    <Text fontWeight="600">{p.name}</Text>
                    <Badge colorScheme="blue">{p.code}</Badge>
                    <Badge colorScheme="gray">{p.type}</Badge>
                  </HStack>
                </Box>
                <AccordionIcon />
              </AccordionButton>
              <AccordionPanel pb={0} pl={4}>
                {renderTree(p.children, depth + 1)}
              </AccordionPanel>
            </AccordionItem>
          </Accordion>
        ) : (
          <Box py={2} px={4} borderRadius="8px" bg={leafBg}>
            <HStack>
              <Text fontWeight="500">{p.name}</Text>
              <Badge colorScheme="blue">{p.code}</Badge>
              <Badge colorScheme="gray">{p.type}</Badge>
            </HStack>
          </Box>
        )}
      </Box>
    ));

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={onOpen}>{t('button.create')}</Button>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} p={6}>
        {tree.length > 0 ? renderTree(tree) : <Text color="gray.500">{t('message.emptyData')}</Text>}
      </Box>

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('modal.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb={4}>
              <FormLabel>{t('form.name.label')}</FormLabel>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={t('form.name.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.code.label')}</FormLabel>
              <Input value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} placeholder={t('form.code.placeholder')} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.type.label')}</FormLabel>
              <Select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}>
                <option value="menu">{t('form.type.menu')}</option>
                <option value="button">{t('form.type.button')}</option>
                <option value="api">{t('form.type.api')}</option>
              </Select>
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>{t('form.parentId.label')}</FormLabel>
              <Input value={form.parent_id} onChange={(e) => setForm({ ...form, parent_id: e.target.value })} placeholder={t('form.parentId.placeholder')} />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleSave}>{tCommon('button.create')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
