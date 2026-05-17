// Chakra Imports
import { Button, Icon, useColorMode } from '@chakra-ui/react';
// Custom Icons
import { IoMdMoon, IoMdSunny } from 'react-icons/io';
import React from 'react';

export default function FixedPlugin(props: { [x: string]: any }) {
	const { ...rest } = props;
	const { colorMode, toggleColorMode } = useColorMode();
	let bgButton = 'linear-gradient(135deg, #868CFF 0%, #4318FF 100%)';

	return (
		<Button
			{...rest}
			h='40px'
			w='40px'
			bg={bgButton}
			variant='no-effects'
			border='1px solid'
			borderColor='#6A53FF'
			borderRadius='50%'
			onClick={toggleColorMode}
			display='flex'
			p='0px'
			alignItems='center'
			justifyContent='center'>
			<Icon h='20px' w='20px' color='white' as={colorMode === 'light' ? IoMdMoon : IoMdSunny} />
		</Button>
	);
}
