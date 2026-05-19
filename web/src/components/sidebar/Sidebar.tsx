import React from 'react';

// chakra imports
import {
	Box,
	Flex,
	Drawer,
	DrawerBody,
	Icon,
	Text,
	useColorModeValue,
	DrawerOverlay,
	useDisclosure,
	DrawerContent,
	DrawerCloseButton,
} from '@chakra-ui/react';
import Content from 'components/sidebar/components/Content';
import { renderThumb, renderTrack, renderView } from 'components/scrollbar/Scrollbar';
import type { SidebarRouteType } from '../../router/types';
import { Scrollbars } from 'react-custom-scrollbars-2';
import { useSidebar } from 'contexts/SidebarContext';

// Assets
import { IoMenuOutline } from 'react-icons/io5';

const SIDEBAR_W = '260px';
const SIDEBAR_COLLAPSED_W = '80px';

function Sidebar(props: { routes: SidebarRouteType[]; [x: string]: any }) {
	const { routes } = props;
	const { collapsed } = useSidebar();

	let variantChange = '0.2s linear';
	let shadow = useColorModeValue('14px 17px 40px 4px rgba(112, 144, 176, 0.08)', 'unset');
	let sidebarBg = useColorModeValue('white', 'navy.800');

	const sidebarWidth = collapsed ? SIDEBAR_COLLAPSED_W : SIDEBAR_W;

	return (
		<Box display={{ sm: 'none', xl: 'block' }} position='fixed' minH='100%'>
			<Box
				bg={sidebarBg}
				transition={variantChange}
				w={sidebarWidth}
				h='100vh'
				m='0px'
				minH='100%'
				overflowX='hidden'
				boxShadow={shadow}>
				<Scrollbars
					autoHide
					renderTrackVertical={renderTrack}
					renderThumbVertical={renderThumb}
					renderView={renderView}>
					<Content routes={routes} collapsed={collapsed} />
				</Scrollbars>
			</Box>
		</Box>
	);
}

export function SidebarResponsive(props: { routes: SidebarRouteType[] }) {
	let sidebarBackgroundColor = useColorModeValue('white', 'navy.800');
	let menuColor = useColorModeValue('gray.400', 'white');
	const { isOpen, onOpen, onClose } = useDisclosure();
	const btnRef = React.useRef<HTMLDivElement>(null);
	const { routes } = props;

	return (
		<Flex display={{ sm: 'flex', xl: 'none' }} alignItems='center'>
			<Flex w='max-content' h='max-content' onClick={onOpen}>
				<Icon
					as={IoMenuOutline}
					color={menuColor}
					my='auto'
					w='20px'
					h='20px'
					me='10px'
					_hover={{ cursor: 'pointer' }}
				/>
			</Flex>
			<Drawer
				isOpen={isOpen}
				onClose={onClose}
				placement={document.documentElement.dir === 'rtl' ? 'right' : 'left'}
				finalFocusRef={btnRef}>
				<DrawerOverlay />
				<DrawerContent w='260px' maxW='260px' bg={sidebarBackgroundColor}>
					<DrawerCloseButton
						zIndex='3'
						onClick={onClose}
						_focus={{ boxShadow: 'none' }}
						_hover={{ boxShadow: 'none' }}
					/>
					<DrawerBody maxW='260px' px='0rem' pb='0'>
						<Scrollbars
							autoHide
							renderTrackVertical={renderTrack}
							renderThumbVertical={renderThumb}
							renderView={renderView}>
							<Content routes={routes} collapsed={false} />
						</Scrollbars>
					</DrawerBody>
				</DrawerContent>
			</Drawer>
		</Flex>
	);
}

export default Sidebar;
