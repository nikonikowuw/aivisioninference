import type { ReactNode } from 'react';
// Chakra imports
import { Flex, Stat, StatLabel, StatNumber, useColorModeValue, Text } from '@chakra-ui/react';
// Custom components
import Card from 'components/card/Card';

export default function Default(props: {
	startContent?: ReactNode;
	endContent?: ReactNode;
	name: string;
	growth?: string | number;
	value: string | number;
}) {
	const { startContent, endContent, name, growth, value } = props;
	const textColor = useColorModeValue('secondaryGray.900', 'white');
	const textColorSecondary = 'secondaryGray.600';

	return (
		<Card py='15px'>
			<Flex my='auto' h='100%' align={{ base: 'center', xl: 'start' }} justify={{ base: 'center', xl: 'center' }}>
				{startContent}

				<Stat my='auto' ms={startContent ? '18px' : '0px'}>
					<StatLabel lineHeight='100%' color={textColorSecondary} fontSize={{ base: 'sm' }}>
						{name}
					</StatLabel>
					<StatNumber color={textColor} fontSize={{ base: '2xl' }}>
						{value}
					</StatNumber>
					{growth && (
						<Flex align='center'>
							<Text color='green.500' fontSize='xs' fontWeight='700' me='5px'>
								{growth}
							</Text>
						</Flex>
					)}
				</Stat>
				<Flex ms='auto' w='max-content'>
					{endContent}
				</Flex>
			</Flex>
		</Card>
	);
}
