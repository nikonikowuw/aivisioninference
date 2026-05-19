// chakra imports
import { Box, Flex, Stack } from '@chakra-ui/react';
//   Custom components
import Brand from 'components/sidebar/components/Brand';
import Links from 'components/sidebar/components/Links';
import type { SidebarRouteType } from '../../../router/types';

function SidebarContent(props: { routes: SidebarRouteType[]; collapsed: boolean }) {
	const { routes, collapsed } = props;

	return (
		<Flex direction='column' height='100%' pt='25px' borderRadius='30px'>
			<Brand collapsed={collapsed} />
			<Stack direction='column' mt='8px' mb='auto'>
				<Box ps={collapsed ? '10px' : '20px'} pe={collapsed ? '6px' : '16px'}>
					<Links routes={routes} collapsed={collapsed} />
				</Box>
			</Stack>
		</Flex>
	);
}

export default SidebarContent;
