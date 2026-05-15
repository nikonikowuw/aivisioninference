/* eslint-disable */

import { NavLink, useLocation } from 'react-router-dom';
// chakra imports
import { Box, Flex, HStack, Text, Tooltip, useColorModeValue } from '@chakra-ui/react';

export function SidebarLinks(props: {
	routes: RoutesType[];
	collapsed?: boolean;
}) {
	let location = useLocation();
	let activeColor = useColorModeValue('gray.700', 'white');
	let inactiveColor = useColorModeValue('secondaryGray.600', 'secondaryGray.600');
	let activeIcon = useColorModeValue('brand.500', 'white');
	let textColor = useColorModeValue('secondaryGray.500', 'white');
	let brandColor = useColorModeValue('brand.500', 'brand.400');

	const { routes, collapsed } = props;

	const activeRoute = (routeName: string) => {
		return location.pathname.includes(routeName);
	};

	const createLinks = (routes: RoutesType[]) => {
		return routes.map((route: RoutesType, index: number) => {
			if (route.layout === '/admin' || route.layout === '/auth' || route.layout === '/rtl') {
				const isActive = activeRoute(route.path.toLowerCase());
				const linkContent = (
					<NavLink key={index} to={route.layout + route.path}>
						{route.icon ? (
							<Box>
								<HStack
									spacing={isActive ? '22px' : '26px'}
									py='5px'
									ps='10px'
									justifyContent={collapsed ? 'center' : 'flex-start'}>
									<Flex
										w='100%'
										alignItems='center'
										justifyContent={collapsed ? 'center' : 'center'}>
										<Box
											color={isActive ? activeIcon : textColor}
											me={collapsed ? '0px' : '18px'}>
											{route.icon}
										</Box>
										{!collapsed && (
											<Text
												me='auto'
												color={isActive ? activeColor : textColor}
												fontWeight={isActive ? 'bold' : 'normal'}>
												{route.name}
											</Text>
										)}
									</Flex>
									{!collapsed && (
										<Box
											h='36px'
											w='4px'
											bg={isActive ? brandColor : 'transparent'}
											borderRadius='5px'
										/>
									)}
								</HStack>
							</Box>
						) : (
							<Box>
								<HStack
									spacing={isActive ? '22px' : '26px'}
									py='5px'
									ps='10px'
									justifyContent={collapsed ? 'center' : 'flex-start'}>
									{!collapsed && (
										<Text
											me='auto'
											color={isActive ? activeColor : inactiveColor}
											fontWeight={isActive ? 'bold' : 'normal'}>
											{route.name}
										</Text>
									)}
									{!collapsed && (
										<Box h='36px' w='4px' bg='brand.400' borderRadius='5px' />
									)}
								</HStack>
							</Box>
						)}
					</NavLink>
				);

				if (collapsed) {
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
