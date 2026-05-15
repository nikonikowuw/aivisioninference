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
  Select,
  useDisclosure,
  HStack,
  Badge,
  Spinner,
  Center,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon, EditIcon } from '@chakra-ui/icons';
import { useEffect, useState, useCallback } from 'react';
import { usersApi, rolesApi, type User, type Role } from 'services/api';

export default function Users() {
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [users, setUsers] = useState<User[]>([]);
  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<User | null>(null);
  const [form, setForm] = useState({ username: '', display_name: '', email: '', password: '', status: 1 });

  const loadUsers = useCallback(async () => {
    try {
      const data = await usersApi.list({ page: 1, page_size: 100 });
      setUsers(data.list);
    } catch {}
  }, []);

  useEffect(() => {
    Promise.all([loadUsers(), rolesApi.list({ page: 1, page_size: 100 }).then((d) => setAllRoles(d.list))])
      .finally(() => setLoading(false));
  }, [loadUsers]);

  const openCreate = () => {
    setEditing(null);
    setForm({ username: '', display_name: '', email: '', password: '', status: 1 });
    onOpen();
  };

  const openEdit = (user: User) => {
    setEditing(user);
    setForm({ username: user.username, display_name: user.display_name, email: user.email, password: '', status: user.status });
    onOpen();
  };

  const handleSave = async () => {
    try {
      if (editing) {
        const updateData: Record<string, unknown> = { ...form };
        if (!form.password) delete updateData.password;
        await usersApi.update(editing.id, updateData as Partial<User>);
        toast({ title: '更新成功', status: 'success' });
      } else {
        await usersApi.create(form as Partial<User>);
        toast({ title: '创建成功', status: 'success' });
      }
      onClose();
      loadUsers();
    } catch (err) {
      toast({ title: '操作失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除该用户？')) return;
    try {
      await usersApi.delete(id);
      toast({ title: '删除成功', status: 'success' });
      loadUsers();
    } catch (err) {
      toast({ title: '删除失败', description: err instanceof Error ? err.message : '', status: 'error' });
    }
  };

  if (loading) {
    return <Center h="400px"><Spinner size="xl" color="brand.500" /></Center>;
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex justify="space-between" align="center" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>用户管理</Text>
        <Button leftIcon={<AddIcon />} variant="brand" onClick={openCreate}>新增用户</Button>
      </Flex>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} overflow="hidden">
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>ID</Th><Th>用户名</Th><Th>显示名称</Th><Th>邮箱</Th><Th>状态</Th><Th>角色</Th><Th>操作</Th>
            </Tr>
          </Thead>
          <Tbody>
            {users.map((u) => (
              <Tr key={u.id}>
                <Td>{u.id}</Td>
                <Td fontWeight="600">{u.username}</Td>
                <Td>{u.display_name}</Td>
                <Td>{u.email}</Td>
                <Td>
                  <Badge colorScheme={u.status === 1 ? 'green' : 'red'}>{u.status === 1 ? '正常' : '禁用'}</Badge>
                </Td>
                <Td>{u.roles?.map((r) => r.name).join(', ') || '-'}</Td>
                <Td>
                  <HStack spacing={2}>
                    <IconButton aria-label="编辑" icon={<EditIcon />} size="sm" variant="ghost" onClick={() => openEdit(u)} />
                    <IconButton aria-label="删除" icon={<DeleteIcon />} size="sm" variant="ghost" colorScheme="red" onClick={() => handleDelete(u.id)} />
                  </HStack>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>

      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{editing ? '编辑用户' : '新增用户'}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl mb={4}>
              <FormLabel>用户名</FormLabel>
              <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>显示名称</FormLabel>
              <Input value={form.display_name} onChange={(e) => setForm({ ...form, display_name: e.target.value })} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>邮箱</FormLabel>
              <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>密码{editing && '（留空不修改）'}</FormLabel>
              <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </FormControl>
            <FormControl mb={4}>
              <FormLabel>状态</FormLabel>
              <Select value={form.status} onChange={(e) => setForm({ ...form, status: Number(e.target.value) })}>
                <option value={1}>正常</option>
                <option value={0}>禁用</option>
              </Select>
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>取消</Button>
            <Button variant="brand" onClick={handleSave}>保存</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
}
