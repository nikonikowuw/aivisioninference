import { AddIcon, DeleteIcon, EditIcon, SearchIcon } from '@chakra-ui/icons';
import { EmptyState } from 'components/empty/EmptyState';
import {
  Box,
  Button,
  Center,
  HStack,
  IconButton,
  Input,
  InputGroup,
  InputLeftElement,
  Spinner,
  Switch,
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
  Badge,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import ConfirmDialog from 'components/confirm-dialog/ConfirmDialog';
import Pagination from 'components/pagination/Pagination';
import { usePagination } from 'hooks/usePagination';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { alertRuleApi, type AlertRule } from 'services/alertRule';
import AlertRuleForm from './form';

const METRIC_TYPE_COLORS: Record<string, string> = {
  cpu_usage: 'blue',
  memory_usage: 'green',
  disk_usage: 'orange',
  temperature: 'red',
  node_offline: 'gray',
  node_error: 'red',
};

export default function AlertRuleList() {
  const { t } = useTranslation('modules/edge-nodes');
  const textColor = useColorModeValue('navy.700', 'white');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const toast = useToast();

  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure();
  const { isOpen: isFormOpen, onOpen: onFormOpen, onClose: onFormClose } = useDisclosure();
  const [deleteTarget, setDeleteTarget] = useState<AlertRule | null>(null);
  const [editTarget, setEditTarget] = useState<AlertRule | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [keyword, setKeyword] = useState('');

  const fetchRules = useCallback((page: number, pageSize: number) => alertRuleApi.list({
    page,
    page_size: pageSize,
    keyword: keyword || undefined,
  }), [keyword]);

  const { list: rules, total, page, pageSize, initialLoading, pageLoading, load: loadRules, changePage, changePageSize } = usePagination<AlertRule>(fetchRules);

  // Load on mount and keyword change
  const prevKeywordRef = useRef(keyword);
  useEffect(() => {
    if (prevKeywordRef.current !== keyword) {
      prevKeywordRef.current = keyword;
      loadRules({ page: 1, pageSize });
    }
  }, [keyword, loadRules, pageSize]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await alertRuleApi.delete(deleteTarget.id);
      toast({ title: 'Rule deleted', status: 'success' });
      onDeleteClose();
      loadRules({ page, pageSize });
    } catch {
      toast({ title: 'Failed to delete rule', status: 'error' });
    } finally {
      setIsDeleting(false);
    }
  };

  const handleToggleEnabled = async (rule: AlertRule) => {
    try {
      await alertRuleApi.update(rule.id, { enabled: !rule.enabled });
      loadRules({ page, pageSize });
    } catch {
      toast({ title: 'Failed to update rule', status: 'error' });
    }
  };

  const handleEdit = (rule: AlertRule) => {
    setEditTarget(rule);
    onFormOpen();
  };

  const handleCreate = () => {
    setEditTarget(null);
    onFormOpen();
  };

  const handleFormClose = () => {
    setEditTarget(null);
    onFormClose();
    loadRules({ page, pageSize });
  };

  return (
    <Box pt={{ sm: '50px', md: '20px' }}>
      <Card>
        <Box p="6">
          <HStack justify="space-between" mb="4">
            <Text fontSize="xl" fontWeight="bold" color={textColor}>
              {t('alert.rules')}
            </Text>
            <Button leftIcon={<AddIcon />} colorScheme="blue" onClick={handleCreate}>
              {t('alert.createRule')}
            </Button>
          </HStack>

          <HStack mb="4">
            <InputGroup maxW="320px">
              <InputLeftElement pointerEvents="none">
                <SearchIcon color="gray.400" />
              </InputLeftElement>
              <Input
                placeholder="Search rules..."
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
            </InputGroup>
          </HStack>

          {initialLoading ? (
            <Text>Loading...</Text>
          ) : (
            <>
              <Table variant="simple" color={textColor} borderColor={borderColor}>
                <Thead>
                  <Tr>
                    <Th>Name</Th>
                    <Th>Metric</Th>
                    <Th>Condition</Th>
                    <Th>Duration</Th>
                    <Th>Channels</Th>
                    <Th>Enabled</Th>
                    <Th>Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {rules.map((rule) => (
                    <Tr key={rule.id}>
                      <Td>
                        <Text fontWeight="bold">{rule.name}</Text>
                        {rule.description && (
                          <Text fontSize="sm" color="gray.500">{rule.description}</Text>
                        )}
                      </Td>
                      <Td>
                        <Badge colorScheme={METRIC_TYPE_COLORS[rule.metric_type] || 'gray'}>
                          {rule.metric_type}
                        </Badge>
                      </Td>
                      <Td>
                        <Text fontFamily="mono">
                          {rule.operator} {rule.threshold}
                        </Text>
                      </Td>
                      <Td>{rule.duration_seconds > 0 ? `${rule.duration_seconds}s` : '-'}</Td>
                      <Td>
                        {(rule.notify_channels || []).join(', ') || '-'}
                      </Td>
                      <Td>
                        <Switch
                          isChecked={rule.enabled}
                          onChange={() => handleToggleEnabled(rule)}
                        />
                      </Td>
                      <Td>
                        <HStack spacing="1">
                          <IconButton
                            aria-label="Edit"
                            icon={<EditIcon />}
                            size="sm"
                            variant="ghost"
                            onClick={() => handleEdit(rule)}
                          />
                          <IconButton
                            aria-label="Delete"
                            icon={<DeleteIcon />}
                            size="sm"
                            variant="ghost"
                            colorScheme="red"
                            onClick={() => { setDeleteTarget(rule); onDeleteOpen(); }}
                          />
                        </HStack>
                      </Td>
                    </Tr>
                  ))}
                  {pageLoading ? (
                    <Tr><Td colSpan={7}><Center py="20px"><Spinner color="brand.500" /></Center></Td></Tr>
                  ) : rules.length === 0 ? (
                    <Tr><Td colSpan={7}><EmptyState /></Td></Tr>
                  ) : null}
                </Tbody>
              </Table>

              <Pagination
                page={page}
                pageSize={pageSize}
                total={total}
                onChange={changePage}
                onPageSizeChange={changePageSize}
              />
            </>
          )}
        </Box>
      </Card>

      <ConfirmDialog
        isOpen={isDeleteOpen}
        onClose={onDeleteClose}
        onConfirm={handleDelete}
        title="Delete Alert Rule"
        message={`Are you sure you want to delete "${deleteTarget?.name}"?`}
        isLoading={isDeleting}
      />

      {isFormOpen && (
        <AlertRuleForm
          isOpen={isFormOpen}
          onClose={handleFormClose}
          rule={editTarget}
        />
      )}
    </Box>
  );
}
