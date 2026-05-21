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
  Badge,
  HStack,
  IconButton,
  Icon,
  Collapse,
  Tooltip,
  VStack,
} from '@chakra-ui/react';
import { AddIcon, EditIcon, DeleteIcon, ChevronDownIcon, ChevronRightIcon } from '@chakra-ui/icons';
import { MdMenu, MdRadioButtonChecked } from 'react-icons/md';
import { useTranslation } from 'react-i18next';
import { useEffect, useState, useMemo } from 'react';
import { permissionsApi, type Permission } from 'services/api';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import { SearchBar } from 'components/search-bar/SearchBar';
import { useFilter } from 'hooks/useFilter';
import { filterTree, type FilteredNode } from 'utils/treeFilter';
import Card from 'components/card/Card';

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

const PermissionRow = ({ 
  node, 
  depth, 
  onEdit, 
  onDelete, 
  expandedIds, 
  onToggleExpand 
}: { 
  node: FilteredNode; 
  depth: number; 
  onEdit: (p: Permission) => void; 
  onDelete: (id: string) => void;
  expandedIds: Set<string>;
  onToggleExpand: (id: string) => void;
}) => {
  const { t: tMenu } = useTranslation('menu');
  const isExpanded = expandedIds.has(node.id);
  const hasChildren = node.children && node.children.length > 0;
  
  const bgHover = useColorModeValue('gray.50', 'whiteAlpha.50');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const lineConnectorColor = useColorModeValue('gray.200', 'gray.600');
  const menuIconColor = useColorModeValue('brand.500', 'brand.300');
  const buttonIconColor = useColorModeValue('orange.500', 'orange.300');
  const textColor = useColorModeValue('navy.700', 'white');

  const getDisplayName = (p: Permission) => {
    if (p.type === 'menu') {
      return tMenu(p.code, { defaultValue: p.name });
    }
    return p.name;
  };

  return (
    <Box w="100%">
      <Flex 
        align="center" 
        py={2} 
        px={3} 
        borderRadius="lg" 
        _hover={{ bg: bgHover }}
        transition="all 0.2s"
        cursor="pointer"
        onClick={() => hasChildren && onToggleExpand(node.id)}
      >
        <HStack spacing={3} flex="1">
          {hasChildren ? (
            <Icon 
              as={isExpanded ? ChevronDownIcon : ChevronRightIcon} 
              w="18px" 
              h="18px" 
              color="gray.400" 
            />
          ) : (
            <Box w="18px" />
          )}
          
          <Icon
            as={node.type === 'menu' ? MdMenu : MdRadioButtonChecked}
            color={node.type === 'menu' ? menuIconColor : buttonIconColor}
            w="16px"
            h="16px"
          />
          
          <HStack spacing={2}>
            <Text 
              fontSize="sm" 
              fontWeight={depth === 0 ? 'bold' : 'medium'} 
              color={textColor}
              opacity={node.isAncestor ? 0.6 : 1}
            >
              {getDisplayName(node as Permission)}
            </Text>
            <Badge variant="subtle" colorScheme="blue" fontSize="2xs" px={2} borderRadius="full">
              {node.code}
            </Badge>
            {node.type === 'button' && (
              <Badge variant="subtle" colorScheme="orange" fontSize="2xs" px={2} borderRadius="full">
                {node.type}
              </Badge>
            )}
          </HStack>
        </HStack>
        
        <HStack spacing={1}>
          <IconButton 
            aria-label="Edit" 
            icon={<EditIcon />} 
            size="xs" 
            variant="ghost" 
            colorScheme="brand"
            onClick={(e) => { e.stopPropagation(); onEdit(node as Permission); }}
          />
          <IconButton 
            aria-label="Delete" 
            icon={<DeleteIcon />} 
            size="xs" 
            variant="ghost" 
            colorScheme="red"
            onClick={(e) => { e.stopPropagation(); onDelete(node.id); }}
          />
        </HStack>
      </Flex>

      {hasChildren && (
        <Collapse in={isExpanded} animateOpacity>
          <Box 
            position="relative" 
            ml={4} 
            pl={4} 
            borderLeft="1px solid"
            borderColor={lineConnectorColor}
          >
            {node.children?.map((child) => (
              <PermissionRow 
                key={child.id} 
                node={child as FilteredNode} 
                depth={depth + 1} 
                onEdit={onEdit} 
                onDelete={onDelete}
                expandedIds={expandedIds}
                onToggleExpand={onToggleExpand}
              />
            ))}
          </Box>
        </Collapse>
      )}
    </Box>
  );
};

export default function Permissions() {
  const { t } = useTranslation('modules/permissions');
  const { t: tCommon } = useTranslation('common');
  const { t: tMenu } = useTranslation('menu');
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
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  const { filters, setFilter, resetFilters, refresh } = useFilter();

  const filteredTree = useMemo(() => {
    return filterTree(tree, filters.keyword, filters.type);
  }, [tree, filters.keyword, filters.type]);

  // 默认展开所有（或者根据需要逻辑控制）
  useEffect(() => {
    if (tree.length > 0 && expandedIds.size === 0) {
      const allIds = new Set<string>();
      const walk = (nodes: Permission[]) => {
        nodes.forEach(n => {
          if (n.children && n.children.length > 0) {
            allIds.add(n.id);
            walk(n.children);
          }
        });
      };
      walk(tree);
      setExpandedIds(allIds);
    }
  }, [tree]);

  const handleToggleExpand = (id: string) => {
    const newExpanded = new Set(expandedIds);
    if (newExpanded.has(id)) {
      newExpanded.delete(id);
    } else {
      newExpanded.add(id);
    }
    setExpandedIds(newExpanded);
  };

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

  const getPermissionName = (p: Permission) => {
    if (p.type === 'menu') {
      return tMenu(p.code, { defaultValue: p.name });
    }
    return p.name;
  };

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button 
          leftIcon={<AddIcon />} 
          variant="brand" 
          onClick={openCreate}
          borderRadius="xl"
          boxShadow="0px 4px 12px rgba(0, 0, 0, 0.1)"
        >
          {t('button.create')}
        </Button>
      </Flex>
      
      <SearchBar
        filters={filters}
        onFilterChange={setFilter}
        onReset={resetFilters}
        onRefresh={refresh}
        selects={[
          {
            name: 'type',
            label: t('form.type.label'),
            options: [
              { value: 'menu', label: t('form.type.menu') },
              { value: 'button', label: t('form.type.button') },
            ],
          },
        ]}
      />

      <Card variant="outline" p={0} overflow="hidden">
        <Box px={6} py={4} bg={useColorModeValue('gray.50', 'whiteAlpha.50')} borderBottom="1px solid" borderColor={borderColor}>
          <HStack justify="space-between">
            <Text fontWeight="bold" color={textColor}>{t('title')}</Text>
            <Badge colorScheme="brand" borderRadius="full" px={3}>{filteredTree.length} Items</Badge>
          </HStack>
        </Box>
        <Box p={4}>
          {filteredTree.length > 0 ? (
            <VStack align="stretch" spacing={0}>
              {filteredTree.map((p) => (
                <PermissionRow 
                  key={p.id} 
                  node={p} 
                  depth={0} 
                  onEdit={openEdit} 
                  onDelete={setDeleteTarget} 
                  expandedIds={expandedIds}
                  onToggleExpand={handleToggleExpand}
                />
              ))}
            </VStack>
          ) : (
            <Center py={10}>
              <VStack spacing={2}>
                <Text color="gray.500">{t('message.emptyData')}</Text>
                <Button variant="ghost" size="sm" onClick={resetFilters}>{tCommon('button.reset')}</Button>
              </VStack>
            </Center>
          )}
        </Box>
      </Card>

      <Modal isOpen={isOpen} onClose={handleClose} isCentered size="lg">
        <ModalOverlay backdropFilter="blur(4px)" />
        <ModalContent borderRadius="2xl">
          <ModalHeader>{editing ? t('modal.editTitle') : t('modal.createTitle')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody pb={6}>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>{t('form.name.label')}</FormLabel>
                <Input 
                  borderRadius="xl"
                  value={form.name} 
                  onChange={(e) => setForm({ ...form, name: e.target.value })} 
                  placeholder={t('form.name.placeholder')} 
                />
              </FormControl>
              <FormControl isRequired>
                <FormLabel>{t('form.code.label')}</FormLabel>
                <Input 
                  borderRadius="xl"
                  value={form.code} 
                  onChange={(e) => setForm({ ...form, code: e.target.value })} 
                  placeholder={t('form.code.placeholder')} 
                />
              </FormControl>
              <FormControl>
                <FormLabel>{t('form.type.label')}</FormLabel>
                <Select 
                  borderRadius="xl"
                  value={form.type} 
                  onChange={(e) => setForm({ ...form, type: e.target.value })}
                >
                  <option value="menu">{t('form.type.menu')}</option>
                  <option value="button">{t('form.type.button')}</option>
                </Select>
              </FormControl>
              <FormControl>
                <FormLabel>{t('form.parentId.label')}</FormLabel>
                <Select 
                  borderRadius="xl"
                  value={form.parent_id} 
                  onChange={(e) => setForm({ ...form, parent_id: e.target.value })} 
                  placeholder={t('form.parentId.placeholder')}
                >
                  <option value="">{t('form.parentId.none')}</option>
                  {flatMenuOptions.map((p) => (
                    <option key={p.id} value={p.id}>
                      {'　'.repeat(p._depth)}{getPermissionName(p)} ({p.code})
                    </option>
                  ))}
                </Select>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={handleClose} borderRadius="xl">{tCommon('button.cancel')}</Button>
            <Button variant="brand" onClick={handleSave} borderRadius="xl">{editing ? tCommon('button.save') : tCommon('button.create')}</Button>
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

