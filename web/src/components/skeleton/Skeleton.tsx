import { Center, Skeleton, Stack, Table, Tbody, Td, Th, Thead, Tr } from '@chakra-ui/react';
import React from 'react';

interface TableSkeletonProps {
  columns: number;
  rows?: number;
}

export const TableSkeleton: React.FC<TableSkeletonProps> = ({ columns, rows = 5 }) => {
  return (
    <Table variant="simple">
      <Thead>
        <Tr>
          {Array.from({ length: columns }).map((_, i) => (
            <Th key={i}>
              <Skeleton h="20px" />
            </Th>
          ))}
        </Tr>
      </Thead>
      <Tbody>
        {Array.from({ length: rows }).map((_, rowIndex) => (
          <Tr key={rowIndex}>
            {Array.from({ length: columns }).map((_, colIndex) => (
              <Td key={colIndex}>
                <Skeleton h="20px" />
              </Td>
            ))}
          </Tr>
        ))}
      </Tbody>
    </Table>
  );
};

export const CardSkeleton: React.FC = () => (
  <Stack p={4} spacing={4}>
    <Skeleton h="40px" w="30%" />
    <Skeleton h="200px" />
  </Stack>
);

export const FullPageSkeleton: React.FC = () => (
  <Center h="400px">
    <Stack w="80%" spacing={6}>
      <Skeleton h="40px" />
      <Skeleton h="300px" />
    </Stack>
  </Center>
);
