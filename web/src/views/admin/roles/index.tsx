import {
  Box,
  Button,
  Flex,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  useColorModeValue,
  IconButton,
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
  Textarea,
  useDisclosure,
  HStack,
  Spinner,
  Center,
  Checkbox,
  VStack,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon, SettingsIcon } from '@chakra-ui/icons';
import { useEffect, useState, useCallback } from 'react';
import { rolesApi, permissionsApi, type Role, type Permission } from 'services/api';

export default function Roles() {
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const { isOpen: isPermOpen, onOpen: onPermOpen, onClose: onPermClose } = useDisclosure();
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Role | null>(null);
  const [form, setForm] = useState({ name: '', description: '' });
  const [permTree, setPermTree] = useState<Permission[]>([]);
  const [selectedPerms, setSelectedPerms] = useState<string[]>([]);
  const [permRoleId, setPermRoleId] = useState<string>('');

  const loadRoles = useCallback(async () => {
    try {
      const data = await rolesApi.list({ page: 1, page_size: 100 });
      setRoles(data.list);
    } catch {}
  }, []);

  useEffect(() => {
    loadRoles().finally(() => setLoading(false));
  }, [loadRoles]);

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', description: '' });
    onOpen();
  };

  const openEdit = (role: Role) => {
    setEditing(role);
    setForm({ name: role.name, description: role.description });
    onOpen();
  };

  const handleSave = async () => {
    try {
      if (editing) {
        await rolesApi.update(editing.id, form);
        toast({ title: '更新成功', status: 'success' });
      } else {
        await rolesApi.create(form);
        toast({ title: '创建成功', status: 'success' });
      }
      onClose();
      loadRoles();
    } catch (err) {
      toast({ title: '操作失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除该角色？')) return;
    try {
      await rolesApi.delete(id);
      toast({ title: '删除成功', status: 'success' });
      loadRoles();
    } catch (err) {
      toast({ title: '删除失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const openPermissions = async (roleId: string) => {
    setPermRoleId(roleId);
    try {
      const [tree, rolePerms] = await Promise.all([
        permissionsApi.tree(),
        rolesApi.getPermissions(roleId),
      ]);
      setPermTree(tree);
      setSelectedPerms(rolePerms.map((p) => p.id));
      onPermOpen();
    } catch (err) {
      toast({ title: '获取权限失败', status: 'error' });
    }
  };

  const handleAssignPerms = async () => {
    try {
      await rolesApi.assignPermissions(permRoleId, selectedPerms);
      toast({ title: '权限分配成功', status: 'success' });
      onPermClose();
      loadRoles();
    } catch (err) {
      toast({ title: '分配失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const togglePerm = (id: string) => {
    setSelectedPerms((prev) =>
      prev.includes(id) ? prev.filter((p) => p !== id) : [...prev, id],
    );
  };

  const renderPermTree = (nodes: Permission[], depth = 0) =>
    nodes.map((p) => (
      <Box key={p.id} ml={depth * 6}>
        <Checkbox
          isChecked={selectedPerms.includes(p.id)}
          onChange={() => togglePerm(p.id)}
          mb={1}
        >
          {p.name} ({p.code})
        </Checkbox>
        {p.children && renderPermTree(p.children, depth + 1)}
      </Box>
    ));

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>角色管理</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>新增角色</Button>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr><Th>ID</Th><Th>角色名</Th><Th>描述</Th><Th>权限数</Th><Th>操作</Th></Tr>
          </Thead>
          <Tbody>
            {roles.map((r) => (
              <Tr key={r.id}>
                <Td>{r.id}</Td>
                <Td fontWeight="600">{r.name}</Td>
                <Td>{r.description || '-'}</Td>
                <Td>{r.permissions?.length ?? 0}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label="分配权限" icon={<SettingsIcon />} size="sm" variant="ghost" colorScheme="blue" onClick={() => openPermissions(r.id)} />
                    <IconButton aria-label="编辑" icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(r)} />
                    <IconButton aria-label="删除" icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => handleDelete(r.id)} />
                  </HStack>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>

      {/* Create/Edit Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editing ? '编辑角色' : '新增角色'}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb={4}>
              <FormLabel>角色名</FormLabel>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>描述</FormLabel>
              <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>取消</Button>
            <Button variant="brand" onClick={handleSave}>保存</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Permissions Modal */}
      <Modal isOpen={isPermOpen} onClose={onPermClose} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>分配权限</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack align="start" spacing={1} maxH="400px" overflowY="auto">
              {permTree.length > 0 ? renderPermTree(permTree) : <Text>暂无权限数据</Text>}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onPermClose}>取消</Button>
            <Button variant="brand" onClick={handleAssignPerms}>保存</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
