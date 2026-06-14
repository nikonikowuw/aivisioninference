import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
	Alert,
	AlertIcon,
	Box,
	Button,
	Flex,
	FormControl,
	FormLabel,
	Input,
	Spinner,
	Text,
	useColorModeValue,
	useToast,
	Select,
	Switch,
	Grid,
	GridItem,
	VStack,
} from '@chakra-ui/react';
import { getGB28181Config, updateGB28181Config, type GB28181Config } from '../../../../services/gb28181';

export default function GB28181ConfigTab() {
	const { t } = useTranslation('modules/gb28181');
	const toast = useToast();
	const bgCard = useColorModeValue('white', 'navy.800');
	const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

	const [config, setConfig] = useState<GB28181Config>({});
	const [loading, setLoading] = useState(true);
	const [saving, setSaving] = useState(false);
	const [error, setError] = useState('');

	useEffect(() => {
		getGB28181Config()
			.then(data => setConfig(data || {}))
			.catch(err => setError(err.message))
			.finally(() => setLoading(false));
	}, []);

	function update(key: string, value: any) {
		setConfig(prev => ({ ...prev, [key]: value }));
	}

	async function handleSave() {
		setSaving(true);
		try {
			await updateGB28181Config(config);
			toast({ title: t('config.saved'), status: 'success', duration: 3000 });
		} catch (err: any) {
			toast({ title: err.message || t('config.saveFailed'), status: 'error', duration: 3000 });
		} finally {
			setSaving(false);
		}
	}

	if (loading) {
		return (
			<Flex justify="center" align="center" h="200px">
				<Spinner size="xl" />
			</Flex>
		);
	}

	if (error) return <Alert status="error"><AlertIcon />{error}</Alert>;

	const isGoSip = Boolean(config['sip.enabled'] ?? true);

	return (
		<Box>
			<Text fontSize="lg" fontWeight="bold" mb={4}>{t('config.title')}</Text>
			
			<Alert status="warning" mb={4} borderRadius="md">
				<AlertIcon />
				<VStack align="start" spacing={1}>
					<Text fontSize="sm" fontWeight="bold">
						{t('config.restartHint')} (端口、监听地址或传输协议变更后，服务将在后台异步自动重载)
					</Text>
					{isGoSip && (
						<Text fontSize="sm">
							<strong>注册提示:</strong> 请将摄像头/NVR 的注册服务器地址修改为: <code>{config['sip.listen_ip'] === '0.0.0.0' ? '当前服务器IP' : String(config['sip.listen_ip'])}</code> 端口: <code>{String(config['sip.port'])}</code>
						</Text>
					)}
				</VStack>
			</Alert>

			<Box p={6} bg={bgCard} borderRadius="lg" border="1px solid" borderColor={borderColor}>
				<Flex direction="column" gap={6}>
					{/* Group 1: Status */}
					<Box borderBottom="1px solid" borderColor={borderColor} pb={4}>
						<Text fontSize="md" fontWeight="semibold" mb={3}>运行状态 (Status)</Text>
						<FormControl display="flex" alignItems="center" h="100%">
							<FormLabel mb="0" mr={4}>
								{t('config.fields.sip.enabled')}
							</FormLabel>
							<Switch
								isChecked={Boolean(config['sip.enabled'] ?? true)}
								onChange={(e) => update('sip.enabled', e.target.checked)}
							/>
						</FormControl>
					</Box>

					{/* Group 2: SIP Server Config */}
					<Box borderBottom="1px solid" borderColor={borderColor} pb={4}>
						<Text fontSize="md" fontWeight="semibold" mb={3}>SIP 核心配置 (SIP Server Core)</Text>
						<Grid templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)' }} gap={4}>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.id')}</FormLabel>
									<Input
										value={String(config['sip.id'] ?? '')}
										onChange={(e) => update('sip.id', e.target.value)}
										placeholder="34020000002000000001"
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.domain')}</FormLabel>
									<Input
										value={String(config['sip.domain'] ?? '')}
										onChange={(e) => update('sip.domain', e.target.value)}
										placeholder="3402000000"
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.realm')}</FormLabel>
									<Input
										value={String(config['sip.realm'] ?? '')}
										onChange={(e) => update('sip.realm', e.target.value)}
										placeholder="3402000000"
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.password')}</FormLabel>
									<Input
										type="password"
										value={String(config['sip.password'] ?? '')}
										onChange={(e) => update('sip.password', e.target.value)}
										placeholder="******"
									/>
								</FormControl>
							</GridItem>
						</Grid>
					</Box>

					{/* Group 3: Network Config */}
					<Box borderBottom="1px solid" borderColor={borderColor} pb={4}>
						<Text fontSize="md" fontWeight="semibold" mb={3}>网络与传输 (Network & Transport)</Text>
						<Grid templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)' }} gap={4}>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.listen_ip')}</FormLabel>
									<Input
										value={String(config['sip.listen_ip'] ?? '0.0.0.0')}
										onChange={(e) => update('sip.listen_ip', e.target.value)}
										placeholder="0.0.0.0"
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.port')}</FormLabel>
									<Input
										type="number"
										value={String(config['sip.port'] ?? '5060')}
										onChange={(e) => update('sip.port', parseInt(e.target.value) || 5060)}
										placeholder="5060"
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.transport')}</FormLabel>
									<Select
										value={String(config['sip.transport'] || 'udp')}
										onChange={(e) => update('sip.transport', e.target.value)}
									>
										<option value="udp">UDP</option>
										<option value="tcp">TCP</option>
									</Select>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.advertised_ip')}</FormLabel>
									<Input
										value={String(config['sip.advertised_ip'] ?? '')}
										onChange={(e) => update('sip.advertised_ip', e.target.value)}
										placeholder="公网公告 IP"
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.rtp_ip')}</FormLabel>
									<Input
										value={String(config['sip.rtp_ip'] ?? '')}
										onChange={(e) => update('sip.rtp_ip', e.target.value)}
										placeholder="RTP 接收 IP"
									/>
								</FormControl>
							</GridItem>
						</Grid>
					</Box>

					{/* Group 4: Performance & Interval Config */}
					<Box pb={4}>
						<Text fontSize="md" fontWeight="semibold" mb={3}>运行策略 (Intervals & Timeouts)</Text>
						<Grid templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)' }} gap={4}>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.heartbeat_timeout')}</FormLabel>
									<Input
										type="number"
										value={String(config['sip.heartbeat_timeout'] ?? '180')}
										onChange={(e) => update('sip.heartbeat_timeout', parseInt(e.target.value) || 180)}
									/>
								</FormControl>
							</GridItem>
							<GridItem>
								<FormControl>
									<FormLabel>{t('config.fields.sip.catalog_interval')}</FormLabel>
									<Input
										type="number"
										value={String(config['sip.catalog_interval'] ?? '3600')}
										onChange={(e) => update('sip.catalog_interval', parseInt(e.target.value) || 3600)}
									/>
								</FormControl>
							</GridItem>
						</Grid>
					</Box>

					<Button colorScheme="blue" alignSelf="flex-start" isLoading={saving} onClick={handleSave}>
						{t('common.save')}
					</Button>
				</Flex>
			</Box>
		</Box>
	);
}
