import { Flex, Box, Table, Tbody, Td, Text, Th, Thead, Tr, useColorModeValue } from '@chakra-ui/react';
import * as React from 'react';
import { useTranslation } from 'react-i18next';
import { useDateFormat } from 'hooks/useDateFormat';
import {
	createColumnHelper,
	flexRender,
	getCoreRowModel,
	getSortedRowModel,
	SortingState,
	useReactTable
} from '@tanstack/react-table';

// Custom components
import Card from 'components/card/Card';
import type { DashboardAuditLog } from 'services/api';
 
const columnHelper = createColumnHelper<DashboardAuditLog>();

export default function AuditTable(props: { tableData?: DashboardAuditLog[] }) {
	const { tableData = [] } = props;
	const { t } = useTranslation(['modules/dashboard', 'modules/audit-logs']);
	const { formatDateTime } = useDateFormat();
	const [ sorting, setSorting ] = React.useState<SortingState>([]);
	const textColor = useColorModeValue('secondaryGray.900', 'white');
	const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
	
	const columns = [
		columnHelper.accessor('username', {
			id: 'username',
			header: () => t('auditLog.columns.username'),
			cell: (info) => (
				<Text color={textColor} fontSize='sm' fontWeight='700'>
					{info.getValue()}
				</Text>
			)
		}),
		columnHelper.accessor('action', {
			id: 'action',
			header: () => t('auditLog.columns.action'),
			cell: (info) => (
				<Text color={textColor} fontSize='sm' fontWeight='700'>
					{info.getValue()}
				</Text>
			)
		}),
		columnHelper.accessor('method', {
			id: 'method',
			header: () => t('auditLog.columns.method'),
			cell: (info) => (
				<Text color={textColor} fontSize='sm' fontWeight='700'>
					{info.getValue()}
				</Text>
			)
		}),
		columnHelper.accessor('created_at', {
			id: 'created_at',
			header: () => t('auditLog.columns.time'),
			cell: (info) => (
				<Text color={textColor} fontSize='sm' fontWeight='700' whiteSpace="nowrap">
					{formatDateTime(info.getValue())}
				</Text>
			)
		})
	];

	const table = useReactTable({
		data: tableData,
		columns,
		state: {
			sorting
		},
		onSortingChange: setSorting,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		debugTable: false
	});

	return (
		<Card flexDirection='column' w='100%' px='0px' overflowX={{ sm: 'scroll', lg: 'hidden' }}>
			<Flex px='25px' mb="8px" justifyContent='space-between' align='center'>
				<Text color={textColor} fontSize='22px' fontWeight='700' lineHeight='100%'>
					{t('auditLog.title')}
				</Text>
			</Flex>
			<Box>
				<Table variant='simple' color='gray.500' mb='24px' mt="12px">
					<Thead>
						{table.getHeaderGroups().map((headerGroup) => (
							<Tr key={headerGroup.id}>
								{headerGroup.headers.map((header) => {
									return (
										<Th
											key={header.id}
											colSpan={header.colSpan}
											pe='10px' 
											borderColor={borderColor}
											cursor='pointer'
											onClick={header.column.getToggleSortingHandler()}>
											<Flex
												justifyContent='space-between'
												align='center'
												fontSize={{ sm: '10px', lg: '12px' }}
												color='gray.400'>
												{flexRender(header.column.columnDef.header, header.getContext())}{{
													asc: ' 🔼',
													desc: ' 🔽',
												}[header.column.getIsSorted() as string] ?? null}
											</Flex>
										</Th>
									);
								})}
							</Tr>
						))}
					</Thead>
					<Tbody>
						{table.getRowModel().rows.length === 0 ? (
							<Tr>
								<Td colSpan={columns.length} borderColor='transparent'>
									<Text color='gray.400' fontSize='sm' textAlign='center' py='24px'>
										{t('auditLog.empty')}
									</Text>
								</Td>
							</Tr>
						) : (
							table.getRowModel().rows.map((row) => {
								return (
									<Tr key={row.id}>
										{row.getVisibleCells().map((cell) => {
											return (
												<Td
													key={cell.id}
													fontSize={{ sm: '14px' }}
													minW={{ sm: '150px', md: '200px', lg: 'auto' }}
													borderColor='transparent'>
													{flexRender(cell.column.columnDef.cell, cell.getContext())}
												</Td>
											);
										})}
									</Tr>
								);
							})
						)}
					</Tbody>
				</Table>
			</Box>
		</Card>
	);
}
