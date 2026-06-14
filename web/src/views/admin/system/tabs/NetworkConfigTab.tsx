import {
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Input,
  Select,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useToast,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
  useDisclosure,
} from '@chakra-ui/react';
import { useEffect, useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from 'services/api';

interface NetworkInterface {
  name: string;
  mac: string;
  state: string;
  config_mode: string;
  ip_address: string;
  cidr: number;
  gateway: string;
  dns_servers: string[];
  is_current: boolean;
}

export default function NetworkConfigTab() {
  const { t } = useTranslation(['modules/system', 'common']);
  const toast = useToast();
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const { isOpen, onOpen, onClose } = useDisclosure();
  const cancelRef = useRef<any>(null);

  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<string | null>(null);
  const [config, setConfig] = useState({
    config_mode: 'dhcp',
    ip_address: '',
    cidr: 24,
    gateway: '',
    dns_servers: '',
  });

  useEffect(() => {
    loadInterfaces();
  }, []);

  async function loadInterfaces() {
    setLoading(true);
    try {
      const data = await request<NetworkInterface[]>('/system/network');
      setInterfaces(data || []);
    } catch (err) {
      console.error('Failed to load network interfaces:', err);
    } finally {
      setLoading(false);
    }
  }

  function startEdit(iface: NetworkInterface) {
    setEditing(iface.name);
    setConfig({
      config_mode: iface.config_mode,
      ip_address: iface.ip_address,
      cidr: iface.cidr,
      gateway: iface.gateway,
      dns_servers: iface.dns_servers?.join(', ') || '',
    });
  }

  async function handleApply() {
    if (!editing) return;

    try {
      await request('/system/network/apply', {
        method: 'POST',
        body: JSON.stringify({
          interfaces: [{
            name: editing,
            config_mode: config.config_mode,
            ip_address: config.config_mode === 'static' ? config.ip_address : undefined,
            cidr: config.config_mode === 'static' ? config.cidr : undefined,
            gateway: config.config_mode === 'static' ? config.gateway : undefined,
            dns_servers: config.config_mode === 'static' ? config.dns_servers.split(',').map(s => s.trim()) : undefined,
          }],
        }),
      });

      toast({
        title: t('network.apply_success'),
        status: 'success',
        duration: 3000,
      });
      setEditing(null);
      loadInterfaces();
    } catch (err: any) {
      toast({
        title: err.message || t('network.apply_failed'),
        status: 'error',
        duration: 3000,
      });
    }
  }

  if (loading) {
    return (
      <Flex justify="center" align="center" h="200px">
        <Spinner size="xl" />
      </Flex>
    );
  }

  return (
    <Box>
      <Box p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
        <Text fontWeight="600" mb={4}>{t('network.interfaces')}</Text>
        <Table variant="simple">
          <Thead>
            <Tr>
              <Th>{t('network.name')}</Th>
              <Th>{t('network.mac')}</Th>
              <Th>{t('network.ip')}</Th>
              <Th>{t('network.mode')}</Th>
              <Th>{t('network.state')}</Th>
              <Th>{t('network.current')}</Th>
              <Th>{t('common:actions')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {interfaces?.map((iface) => (
              <Tr key={iface.name}>
                <Td>{iface.name}</Td>
                <Td>{iface.mac}</Td>
                <Td>{iface.ip_address}/{iface.cidr}</Td>
                <Td>{iface.config_mode}</Td>
                <Td>
                  <Text color={iface.state === 'up' ? 'green.500' : 'red.500'}>
                    {iface.state}
                  </Text>
                </Td>
                <Td>{iface.is_current ? '✓' : ''}</Td>
                <Td>
                  <Button size="sm" colorScheme="brand" onClick={() => startEdit(iface)}>
                    {t('common:edit')}
                  </Button>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>

      {editing && (
        <Box mt={6} p={4} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
          <Text fontWeight="600" mb={4}>{t('network.edit_config')} - {editing}</Text>
          <Flex direction="column" gap={4}>
            <FormControl>
              <FormLabel>{t('network.config_mode')}</FormLabel>
              <Select
                value={config.config_mode}
                onChange={(e) => setConfig({ ...config, config_mode: e.target.value })}
              >
                <option value="dhcp">DHCP</option>
                <option value="static">{t('network.static')}</option>
              </Select>
            </FormControl>

            {config.config_mode === 'static' && (
              <>
                <FormControl>
                  <FormLabel>{t('network.ip_address')}</FormLabel>
                  <Input
                    value={config.ip_address}
                    onChange={(e) => setConfig({ ...config, ip_address: e.target.value })}
                    placeholder="192.168.1.100"
                  />
                </FormControl>
                <FormControl>
                  <FormLabel>{t('network.cidr')}</FormLabel>
                  <Input
                    type="number"
                    value={config.cidr}
                    onChange={(e) => setConfig({ ...config, cidr: parseInt(e.target.value) })}
                    placeholder="24"
                  />
                </FormControl>
                <FormControl>
                  <FormLabel>{t('network.gateway')}</FormLabel>
                  <Input
                    value={config.gateway}
                    onChange={(e) => setConfig({ ...config, gateway: e.target.value })}
                    placeholder="192.168.1.1"
                  />
                </FormControl>
                <FormControl>
                  <FormLabel>{t('network.dns')}</FormLabel>
                  <Input
                    value={config.dns_servers}
                    onChange={(e) => setConfig({ ...config, dns_servers: e.target.value })}
                    placeholder="8.8.8.8, 8.8.4.4"
                  />
                </FormControl>
              </>
            )}

            <Flex gap={4}>
              <Button colorScheme="brand" onClick={onOpen}>
                {t('network.apply')}
              </Button>
              <Button variant="ghost" onClick={() => setEditing(null)}>
                {t('common:cancel')}
              </Button>
            </Flex>
          </Flex>
        </Box>
      )}

      <AlertDialog
        isOpen={isOpen}
        leastDestructiveRef={cancelRef}
        onClose={onClose} isCentered>
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('network.apply_confirm_title', { defaultValue: '确认应用网络配置' })}
            </AlertDialogHeader>

            <AlertDialogBody>
              {t('network.apply_confirm_message', { 
                defaultValue: '修改网络配置（特别是静态 IP）可能会导致当前连接中断。如果配置错误，系统将在 60 秒后自动回滚。确认继续吗？' 
              })}
            </AlertDialogBody>

            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onClose}>
                {t('common:cancel')}
              </Button>
              <Button colorScheme="red" onClick={() => { onClose(); handleApply(); }} ml={3}>
                {t('network.apply')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Box>
  );
}
