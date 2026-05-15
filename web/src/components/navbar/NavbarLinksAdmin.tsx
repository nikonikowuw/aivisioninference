import {
	Avatar,
	Button,
	Flex,
	Icon,
	Menu,
	MenuButton,
	MenuItem,
	MenuList,
	Text,
	useColorModeValue,
	useColorMode
} from '@chakra-ui/react';
import { SearchBar } from 'components/navbar/searchBar/SearchBar';
import { SidebarResponsive } from 'components/sidebar/Sidebar';
import { useAuth } from 'contexts/AuthContext';
import { useNavigate } from 'react-router-dom';
import { IoMdMoon, IoMdSunny } from 'react-icons/io';
import routes from 'routes';

export default function HeaderLinks(props: { secondary: boolean; [key: string]: any }) {
	const { secondary } = props;
	const { colorMode, toggleColorMode } = useColorMode();
	const { user, logout } = useAuth();
	const navigate = useNavigate();

	const navbarIcon = useColorModeValue('gray.400', 'white');
	const menuBg = useColorModeValue('white', 'navy.800');
	const textColor = useColorModeValue('secondaryGray.900', 'white');
	const shadow = useColorModeValue(
		'14px 17px 40px 4px rgba(112, 144, 176, 0.18)',
		'14px 17px 40px 4px rgba(112, 144, 176, 0.06)'
	);
	const borderColor = useColorModeValue('#E6ECFA', 'rgba(135, 140, 189, 0.3)');

	const handleLogout = async () => {
		await logout();
		navigate('/auth/sign-in');
	};

	return (
		<Flex
			w={{ sm: '100%', md: 'auto' }}
			alignItems='center'
			flexDirection='row'
			bg={menuBg}
			flexWrap={secondary ? { base: 'wrap', md: 'nowrap' } : 'unset'}
			p='10px'
			borderRadius='30px'
			boxShadow={shadow}
		>
			<SidebarResponsive routes={routes} />
			<Button
				variant='no-hover'
				bg='transparent'
				p='0px'
				minW='unset'
				minH='unset'
				h='18px'
				w='max-content'
				onClick={toggleColorMode}
			>
				<Icon
					me='10px'
					h='18px'
					w='18px'
					color={navbarIcon}
					as={colorMode === 'light' ? IoMdMoon : IoMdSunny}
				/>
			</Button>
			<Menu>
				<MenuButton p='0px'>
					<Avatar
						_hover={{ cursor: 'pointer' }}
						color='white'
						name={user?.display_name || user?.username || 'User'}
						bg='#11047A'
						size='sm'
						w='40px'
						h='40px'
						src={user?.avatar_url}
					/>
				</MenuButton>
				<MenuList boxShadow={shadow} p='0px' mt='10px' borderRadius='20px' bg={menuBg} border='none'>
					<Flex w='100%' mb='0px'>
						<Text
							ps='20px'
							pt='16px'
							pb='10px'
							w='100%'
							borderBottom='1px solid'
							borderColor={borderColor}
							fontSize='sm'
							fontWeight='700'
							color={textColor}
						>
							{user?.display_name || user?.username || '用户'}
						</Text>
					</Flex>
					<Flex flexDirection='column' p='10px'>
						<MenuItem
							_hover={{ bg: 'none' }}
							_focus={{ bg: 'none' }}
							color='red.400'
							borderRadius='8px'
							px='14px'
							onClick={handleLogout}
						>
							<Text fontSize='sm'>退出登录</Text>
						</MenuItem>
					</Flex>
				</MenuList>
			</Menu>
		</Flex>
	);
}
