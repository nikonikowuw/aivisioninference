/* eslint-disable */
// Chakra Imports
import { Box, Breadcrumb, BreadcrumbItem, BreadcrumbLink, Flex, Icon, Link, Text, useColorModeValue } from '@chakra-ui/react';
import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import AdminNavbarLinks from 'components/navbar/NavbarLinksAdmin';
import { useSidebar } from 'contexts/SidebarContext';
import { IoMenuOutline } from 'react-icons/io5';

export default function AdminNavbar(props: {
	secondary: boolean;
	message: string | boolean;
	brandText: string;
	logoText: string;
	fixed: boolean;
	onOpen: (...args: any[]) => any;
}) {
	const [scrolled, setScrolled] = useState(false);
	const { secondary, brandText, collapsed } = props;
	const { setCollapsed } = useSidebar();
	const { t } = useTranslation('layout');

	useEffect(() => {
		const changeNavbar = () => setScrolled(window.scrollY > 1);
		window.addEventListener('scroll', changeNavbar);
		return () => window.removeEventListener('scroll', changeNavbar);
	}, []);

	const mainText = useColorModeValue('navy.700', 'white');
	const secondaryText = useColorModeValue('gray.700', 'white');
	const navbarBg = useColorModeValue('rgba(244, 247, 254, 0.2)', 'rgba(11,20,55,0.5)');

	return (
		<Box
			position="fixed"
			bg={navbarBg}
			borderColor="transparent"
			backdropFilter="blur(20px)"
			borderRadius='16px'
			borderWidth='1.5px'
			borderStyle='solid'
			transitionDelay='0s, 0s, 0s, 0s'
			transitionDuration=' 0.25s, 0.25s, 0.25s, 0s'
			transitionProperty='box-shadow, background-color, filter, border'
			transitionTimingFunction='linear, linear, linear, linear'
			alignItems={{ xl: 'center' }}
			display={secondary ? 'block' : 'flex'}
			minH='75px'
			justifyContent={{ xl: 'center' }}
			lineHeight='25.6px'
			mx='auto'
			pb='8px'
			right={{ base: '12px', md: '30px', lg: '30px', xl: '30px' }}
			px={{ sm: '15px', md: '10px' }}
			ps={{ xl: '12px' }}
			pt='8px'
			top={{ base: '12px', md: '16px', xl: '18px' }}
			w={{
				base: 'calc(100vw - 6%)',
				md: 'calc(100vw - 8%)',
				lg: 'calc(100vw - 6%)',
				xl: collapsed ? 'calc(100vw - 130px)' : 'calc(100vw - 320px)',
				'2xl': collapsed ? 'calc(100vw - 130px)' : 'calc(100vw - 335px)',
			}}>
			<Flex
				w='100%'
				flexDirection={{ sm: 'column', md: 'row' }}
				alignItems={{ xl: 'center' }}>
				<Flex alignItems='center' gap='12px'>
					<Icon
						as={IoMenuOutline}
						color={mainText}
						w='22px'
						h='22px'
						cursor='pointer'
						onClick={() => setCollapsed(!collapsed)}
						_hover={{ opacity: 0.7 }}
					/>
					<Box mb={{ sm: '8px', md: '0px' }}>
						<Breadcrumb>
							<BreadcrumbItem color={secondaryText} fontSize='sm' mb='5px'>
								<BreadcrumbLink href='#' color={secondaryText}>
									{t('navbar.pages')}
								</BreadcrumbLink>
							</BreadcrumbItem>
							<BreadcrumbItem color={secondaryText} fontSize='sm'>
								<BreadcrumbLink href='#' color={secondaryText}>
									{brandText}
								</BreadcrumbLink>
							</BreadcrumbItem>
						</Breadcrumb>
						<Link
							color={mainText}
							href='#'
							bg='inherit'
							borderRadius='inherit'
							fontWeight='bold'
							fontSize='34px'
							_hover={{ color: mainText }}
							_active={{
								bg: 'inherit',
								transform: 'none',
								borderColor: 'transparent',
							}}
							_focus={{
								boxShadow: 'none',
							}}>
							{brandText}
						</Link>
					</Box>
				</Flex>
				<Box ms='auto' w={{ sm: '100%', md: 'unset' }}>
					<AdminNavbarLinks secondary={secondary} />
				</Box>
			</Flex>
		</Box>
	);
}
