import { Flex, Image, Text, useColorModeValue } from '@chakra-ui/react';
import { useBrand } from 'contexts/BrandContext';

// Custom components
import { HSeparator } from 'components/separator/Separator';

export function SidebarBrand(props: { collapsed?: boolean }) {
	const { collapsed } = props;
	const { brand } = useBrand();
	const brandColor = useColorModeValue('navy.700', 'white');

	return (
		<Flex alignItems='flex-start' flexDirection='column' w='100%'>
			<Flex
				alignItems='center'
				justifyContent='flex-start'
				w='100%'
				ps={collapsed ? '24px' : '28px'}
				py='36px'
				gap='12px'
				transition='all 0.3s cubic-bezier(0.685, 0.0473, 0.346, 1)'>
				<Image src={brand.logo_url || '/favicon.ico'} fallbackSrc='/favicon.ico' w='32px' h='32px' minW='32px' objectFit='contain' />
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
						{brand.system_name || 'Niko Admin'}
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
