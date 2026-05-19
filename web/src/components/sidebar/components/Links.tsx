import { useState, useCallback, useEffect } from 'react';
import { useLocation, NavLink, useNavigate } from 'react-router-dom';
import { Box, Flex, HStack, Text, Tooltip, useColorModeValue, Collapse, Icon } from '@chakra-ui/react';
import { MdChevronRight } from 'react-icons/md';
import type { SidebarRouteType } from '../../../router/types';

export function SidebarLinks(props: {
	routes: SidebarRouteType[];
	collapsed?: boolean;
}) {
	const location = useLocation();
	const navigate = useNavigate();
	const [openMenus, setOpenMenus] = useState<Record<string, boolean>>({});

	const activeColor = useColorModeValue('gray.700', 'white');
	const activeIcon = useColorModeValue('brand.500', 'white');
	const textColor = useColorModeValue('secondaryGray.500', 'white');
	const brandColor = useColorModeValue('brand.500', 'brand.400');
	const hoverBg = useColorModeValue('gray.50', 'whiteAlpha.100');

	const { routes, collapsed } = props;

	// `+ '/'` 后缀确保路径边界精确匹配，防止 /admin/user 误匹配 /admin/users
	const activeRoute = useCallback((routeName: string) => {
		return location.pathname === routeName || location.pathname.startsWith(routeName + '/');
	}, [location.pathname]);

	const toggleMenu = (menuKey: string) => {
		setOpenMenus((prev) => ({
			...prev,
			[menuKey]: !prev[menuKey],
		}));
	};

	// 自动展开包含当前激活子项的菜单
	useEffect(() => {
		setOpenMenus((prev) => {
			const next = { ...prev };
			let changed = false;
			const checkAndExpand = (items: SidebarRouteType[]): boolean => {
				let anyActive = false;
				for (const route of items) {
					const fullPath = route.layout + route.path;
					const isDirectActive = activeRoute(fullPath);
					const isChildrenActive = route.items ? checkAndExpand(route.items) : false;

					if (isDirectActive || isChildrenActive) {
						anyActive = true;
						if (route.items && route.items.length > 0) {
							if (!next[route.key]) {
								next[route.key] = true;
								changed = true;
							}
						}
					}
				}
				return anyActive;
			};
			checkAndExpand(routes);
			return changed ? next : prev;
		});
	}, [location.pathname, routes, activeRoute]);

	const createLinks = (routes: SidebarRouteType[], isSubMenu = false) => {
		return routes.map((route: SidebarRouteType, index: number) => {
			if (route.layout === '/admin' || route.layout === '/auth' || route.layout === '/rtl') {
				const fullPath = route.layout + route.path;
				const isActive = activeRoute(fullPath);
				const hasItems = route.items && route.items.length > 0;
				const isOpen = openMenus[route.key];

				const linkContent = hasItems ? (
					<Box key={index} mb='4px'>
						<Flex
							alignItems='center'
							justifyContent='space-between'
							py='8px'
							ps={isSubMenu ? '34px' : '10px'}
							pe='10px'
							cursor='pointer'
							onClick={() => {
								if (collapsed && hasItems) {
									// 折叠状态下点击有子菜单的父项，导航到第一个子项
									const firstChild = route.items![0];
									navigate(firstChild.layout + firstChild.path);
								} else {
									toggleMenu(route.key);
								}
							}}
							transition='0.2s linear'
							borderRadius='8px'
							_hover={{ bg: hoverBg }}>
							<HStack spacing={collapsed ? '0' : '15px'} w='full'>
								<Box color={isActive ? activeIcon : textColor} display="flex" alignItems="center">
									{route.icon}
								</Box>
								{!collapsed && (
									<Text
										me='auto'
										color={isActive ? activeColor : textColor}
										fontWeight={isActive ? 'bold' : '500'}
										fontSize='sm'>
										{route.name}
									</Text>
								)}
								{!collapsed && (
									<Icon
										as={MdChevronRight}
										w='16px'
										h='16px'
										transition='transform 0.2s'
										transform={isOpen ? 'rotate(90deg)' : 'none'}
										color={textColor}
									/>
								)}
							</HStack>
						</Flex>
						<Collapse in={isOpen && !collapsed}>
							<Box mt='2px'>
								{/* hasItems 在外层已保证 route.items 非空且非空数组 */}
								{route.items && createLinks(route.items, true)}
							</Box>
						</Collapse>
					</Box>
				) : (
					<NavLink key={index} to={fullPath}>
						<Box mb='4px'>
							<HStack
								spacing={isActive ? '22px' : '26px'}
								py='8px'
								ps={isSubMenu ? '34px' : '10px'}
								justifyContent={collapsed ? 'center' : 'flex-start'}
								transition='0.2s linear'
								borderRadius='8px'
								_hover={{ bg: hoverBg }}>
								<Flex
									w='100%'
									alignItems='center'
									justifyContent={collapsed ? 'center' : 'flex-start'}>
									<Box
										color={isActive ? activeIcon : textColor}
										me={collapsed ? '0px' : '15px'}
										display="flex"
										alignItems="center">
										{route.icon}
									</Box>
									{!collapsed && (
										<Text
											me='auto'
											color={isActive ? activeColor : textColor}
											fontWeight={isActive ? 'bold' : '500'}
											fontSize='sm'>
											{route.name}
										</Text>
									)}
								</Flex>
								{!collapsed && !isSubMenu && (
									<Box
										h='30px'
										w='4px'
										bg={isActive ? brandColor : 'transparent'}
										borderRadius='5px'
									/>
								)}
							</HStack>
						</Box>
					</NavLink>
				);

				if (collapsed && !isSubMenu) {
					return (
						<Tooltip key={index} label={route.name} placement='right' hasArrow>
							{linkContent}
						</Tooltip>
					);
				}

				return linkContent;
			}
		});
	};

	return <>{createLinks(routes)}</>;
}

export default SidebarLinks;
