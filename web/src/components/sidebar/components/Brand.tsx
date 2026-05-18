import { Flex, Text, useColorModeValue, Icon, Box } from '@chakra-ui/react';
import { RiCommandFill } from 'react-icons/ri';

// Custom components
import { HSeparator } from 'components/separator/Separator';

export function SidebarBrand(props: { collapsed?: boolean }) {
	const { collapsed } = props;
	let brandColor = useColorModeValue('navy.700', 'white');

	return (
		<Flex alignItems='flex-start' flexDirection='column' w='100%'>
			<Flex
				alignItems='center'
				justifyContent='flex-start'
				w='100%'
				ps={collapsed ? '22px' : '28px'}
				py='36px'
				gap='12px'
				transition='all 0.3s cubic-bezier(0.685, 0.0473, 0.346, 1)'>
				<Box
					display='flex'
					alignItems='center'
					justifyContent='center'
					w='36px'
					h='36px'
					minW='36px'
					borderRadius='10px'
					bgGradient='linear(to-br, brand.400, brand.600)'
					boxShadow='0px 4px 10px rgba(67, 24, 255, 0.25)'>
					<Icon as={RiCommandFill} w='20px' h='20px' color='white' />
				</Box>
				{!collapsed && (
					<Text
						display='flex'
						alignItems='center'
						fontSize='20px'
						fontWeight='800'
						color={brandColor}
						letterSpacing='-0.5px'
						bgGradient='linear(to-r, navy.700, brand.500)'
						bgClip='text'>
						Niko
						<Text as='span' color='brand.500' ml='1.5px'>
							Admin
						</Text>
					</Text>
				)}
			</Flex>
			<Flex w='100%' justifyContent='center'>
				<HSeparator w='calc(100% - 56px)' />
			</Flex>
		</Flex>
	);
}

export default SidebarBrand;
