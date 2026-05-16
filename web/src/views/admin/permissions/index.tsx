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
  IconButton,
} from '@chakra-ui/react';
import { AddIcon, EditIcon, DeleteIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useMemo } from 'react';
import { permissionsApi, type Permission } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';

function flattenTree(nodes: Permission[], depth = 0): (Permission & { _depth: number })[] {
  const result: (Permission & { _depth: number })[] = [];
  for (const node of nodes) {
    result.push({ ...node, _depth: depth });
    if (node.children) {
      result.push(...flattenTree(node.children, depth + 1));
    }
  }
  return result;
}

function getDescendantIds(nodes: Permission[], id: string): Set<string> {
  const ids = new Set<string>();
  function walk(list: Permission[]) {
    for (const n of list) {
      ids.add(n.id);
      if (n.children) walk(n.children);
    }
  }
  function find(list: Permission[]): boolean {
    for (const n of list) {
      if (n.id === id) {
        ids.add(n.id);
        walk(n.children ?? []);
        return true;
      }
      if (n.children && find(n.children)) return true;
    }
    return false;
  }
  find(nodes);
  return ids;
}

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
  const [editing, setEditing] = useState<Permission | null>(null);
  const [form, setForm] = useState({ name: '', code: '', type: 'menu', parent_id: '' });
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleClose = () => {
    setEditing(null);
    setForm({ name: '', code: '', type: 'menu', parent_id: '' });
    onClose();
  };

  const flatMenuOptions = useMemo(() => {
    const flat = flattenTree(tree).filter(p => p.type === 'menu');
    if (!editing) return flat;
    const excludeIds = getDescendantIds(tree, editing.id);
    return flat.filter(p => !excludeIds.has(p.id));
  }, [tree, editing]);

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

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', code: '', type: 'menu', parent_id: '' });
    onOpen();
  };

  const openEdit = (p: Permission) => {
    setEditing(p);
    setForm({ name: p.name, code: p.code, type: p.type, parent_id: p.parent_id ?? '' });
    onOpen();
  };

  const handleSave = async () => {
    if (!form.name.trim() || !form.code.trim()) {
      toast({ title: tCommon('message.requiredFields'), status: 'warning' });
      return;
    }
    try {
      const payload = { ...form, parent_id: form.parent_id || null };
      if (editing) {
        await permissionsApi.update(editing.id, payload);
        toast({ title: t('message.updateSuccess'), status: 'success' });
      } else {
        await permissionsApi.create(payload);
        toast({ title: t('message.createSuccess'), status: 'success' });
      }
      handleClose();
      loadTree();
    } catch (err) {
      const key = editing ? 'message.updateFailed' : 'message.createFailed';
      toast({ title: t(key), description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await permissionsApi.delete(deleteTarget);
      toast({ title: t('message.deleteSuccess'), status: 'success' });
      loadTree();
    } catch (err) {
      toast({ title: t('message.deleteFailed'), description: err instanceof Error ? err.message : '', status: 'error' });
    } finally {
      setIsDeleting(false);
      setDeleteTarget(null);
    }
  };

  const leafBg = useColorModeValue('gray.50', 'whiteAlpha.50');

  const renderTree = (nodes: Permission[], depth = 0) =>
    nodes.map((p) => (
      <Box key={p.id} ml={depth * 4} mb={2}>
        <Flex align="center" justify="space-between">
          <Box flex="1">
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
          <HStack spacing={1}>
            <IconButton aria-label={tCommon('button.edit')} icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(p)} />
            <IconButton aria-label={tCommon('button.delete')} icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => setDeleteTarget(p.id)} />
          </HStack>
        </Flex>
      </Box>
    ));

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>{t('button.create')}</Button>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} p={6}>
        {tree.length > 0 ? renderTree(tree) : <Text color="gray.500">{t('message.emptyData')}</Text>}
      </Box>

      <Modal isOpen={isOpen} onClose={handleClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editing ? t('modal.editTitle') : t('modal.createTitle')}</ModalHeader>
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
              <Select value={form.parent_id} onChange={(e) => setForm({ ...form, parent_id: e.target.value })} placeholder={t('form.parentId.placeholder')}>
                <option value="">{t('form.parentId.none')}</option>
                {flatMenuOptions.map((p) => (
                  <option key={p.id} value={p.id}>
                    {'　'.repeat(p._depth)}{p.name} ({p.code})
                  </option>
                ))}
              </Select>
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={handleClose}>{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleSave}>{editing ? tCommon('button.save') : tCommon('button.create')}</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      <ConfirmDialog
        isOpen={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title={t('modal.deleteTitle')}
        message={t('message.deleteConfirm')}
        isLoading={isDeleting}
      />
    </Box>
  );
}
