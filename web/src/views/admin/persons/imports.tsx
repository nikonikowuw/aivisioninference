import {
  Badge,
  Box,
  Button,
  Flex,
  HStack,
  Progress,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
  useDisclosure,
  useToast,
} from '@chakra-ui/react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { PersonImportTask, personImportsApi } from 'services/api';
import { EmptyState } from 'components/empty/EmptyState';
import { PersonImportModal } from './components/PersonImportModal';

const statusColorMap: Record<string, string> = {
  pending: 'yellow',
  running: 'blue',
  completed: 'green',
  failed: 'red',
};

export default function PersonImportsPage() {
  const { t } = useTranslation('modules/persons');
  const { t: tCommon } = useTranslation('common');
  const toast = useToast();
  const cardBg = useColorModeValue('white', 'navy.800');
  const { isOpen, onOpen, onClose } = useDisclosure();

  const [tasks, setTasks] = useState<PersonImportTask[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await personImportsApi.list({ page: 1, page_size: 50 });
      setTasks(res.list);
    } catch {
      toast({ title: t('message.loadFailed'), status: 'error' });
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { fetchData(); }, [fetchData]);

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" bg={cardBg} p="24px" borderRadius="16px" boxShadow="xl">
        <Flex justify="space-between" align="center" mb="16px">
          <Text fontSize="22px" fontWeight="700">{t('imports.title')}</Text>
          <Button colorScheme="blue" size="sm" onClick={onOpen}>{t('imports.upload')}</Button>
        </Flex>

        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>{t('imports.columns.fileName')}</Th>
              <Th>{t('imports.columns.status')}</Th>
              <Th>{t('imports.columns.progress')}</Th>
              <Th>{t('imports.columns.result')}</Th>
              <Th>{t('imports.columns.createdAt')}</Th>
            </Tr>
          </Thead>
          <Tbody>
            {loading ? (
              <Tr><Td colSpan={5} textAlign="center">{tCommon('loading')}</Td></Tr>
            ) : tasks.length === 0 ? (
              <Tr><Td colSpan={5}><EmptyState /></Td></Tr>
            ) : (
              tasks.map((task) => (
                <Tr key={task.id}>
                  <Td fontWeight="600">{task.file_name}</Td>
                  <Td>
                    <Badge colorScheme={statusColorMap[task.status] || 'gray'}>
                      {t(`imports.status.${task.status}`)}
                    </Badge>
                  </Td>
                  <Td>
                    {task.status === 'running' ? (
                      <Progress
                        value={task.total_rows > 0 ? ((task.success_rows + task.failed_rows) / task.total_rows) * 100 : 0}
                        size="sm"
                        colorScheme="blue"
                        borderRadius="full"
                      />
                    ) : (
                      <Text fontSize="sm" color="gray.500">
                        {task.status === 'completed' ? '100%' : '-'}
                      </Text>
                    )}
                  </Td>
                  <Td>
                    <Text fontSize="sm">
                      {task.success_rows} / {task.failed_rows} / {task.total_rows}
                    </Text>
                  </Td>
                  <Td>{new Date(task.created_at).toLocaleString()}</Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      </Flex>

      <PersonImportModal
        isOpen={isOpen}
        onClose={onClose}
        onSuccess={fetchData}
      />
    </Box>
  );
}
