// Chakra imports
import { Flex, Text, useColorModeValue } from '@chakra-ui/react';

// Custom components
import { HSeparator } from 'components/separator/Separator';

export function SidebarBrand(props: { collapsed?: boolean }) {
	const { collapsed } = props;
	let brandColor = useColorModeValue('navy.700', 'white');

	return (
		<Flex alignItems='center' flexDirection='column'>
			{collapsed ? (
				<Text fontSize='20px' fontWeight='bold' color={brandColor} my='32px'>
					N
				</Text>
			) : (
				<Text fontSize='22px' fontWeight='bold' color={brandColor} my='32px' letterSpacing='tight'>
					Niko Admin
				</Text>
			)}
			<HSeparator mb='20px' />
		</Flex>
	);
}

export default SidebarBrand;
