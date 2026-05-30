// Chakra imports
import { Box, Flex, Select, Text, useColorModeValue } from '@chakra-ui/react';
import Card from 'components/card/Card';
import { useTranslation } from 'react-i18next';
// Custom components
import BarChart from 'components/charts/BarChart';
import { barChartDataUserActivity, barChartOptionsUserActivity } from 'variables/charts';

export default function UserActivity(props: { [x: string]: any }) {
	const { ...rest } = props;
	const { t } = useTranslation('common');

	// Chakra Color Mode
	const textColor = useColorModeValue('secondaryGray.900', 'white');
	return (
		<Card alignItems='center' flexDirection='column' w='100%' {...rest}>
			<Flex align='center' w='100%' px='15px' py='10px'>
				<Text me='auto' color={textColor} fontSize='xl' fontWeight='700' lineHeight='100%'>
					{t('date.userActivity', { defaultValue: 'User Activity' })}
				</Text>
				<Select
					id='user_type'
					w='unset'
					variant='transparent'
					display='flex'
					alignItems='center'
					defaultValue='Weekly'>
					<option value='Weekly'>{t('date.weekly', { defaultValue: 'Weekly' })}</option>
					<option value='Daily'>{t('date.daily', { defaultValue: 'Daily' })}</option>
					<option value='Monthly'>{t('date.monthly', { defaultValue: 'Monthly' })}</option>
				</Select>
			</Flex>

			<Box h='240px' mt='auto'>
				<BarChart chartData={barChartDataUserActivity} chartOptions={barChartOptionsUserActivity} />
			</Box>
		</Card>
	);
}
