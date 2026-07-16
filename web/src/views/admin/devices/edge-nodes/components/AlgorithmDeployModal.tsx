import {
  Box,
  Button,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Select,
  Text,
  useToast,
  VStack,
} from '@chakra-ui/react';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { algorithmPackagesApi, ApiError, type AlgorithmPackage } from 'services/api';
import { edgeNodeApi, type EdgeNode } from 'services/edgeNode';

interface AlgorithmDeployModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
  node: EdgeNode;
}

export function AlgorithmDeployModal({ isOpen, onClose, onSuccess, node }: AlgorithmDeployModalProps) {
  const { t } = useTranslation('modules/edge-nodes');
  const { t: tCommon } = useTranslation('common');
  const toast = useToast();
  const [algoPackages, setAlgoPackages] = useState<AlgorithmPackage[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [isDeploying, setIsDeploying] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setIsLoading(true);
    algorithmPackagesApi.list({ page: 1, page_size: 100, status: 'released' })
      .then((res) => setAlgoPackages(res.list))
      .catch(() => toast({ title: tCommon('message.loadFailed'), status: 'error' }))
      .finally(() => setIsLoading(false));
  }, [isOpen, toast, tCommon]);

  const handleDeploy = useCallback(async () => {
    if (!selectedId) return;
    setIsDeploying(true);
    try {
      await edgeNodeApi.deployAlgorithm(node.id, selectedId);
      toast({ title: t('message.deploySuccess'), status: 'success' });
      onSuccess?.();
      onClose();
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : t('message.deployFailed');
      toast({ title: msg, status: 'error' });
    } finally {
      setIsDeploying(false);
    }
  }, [selectedId, node.id, toast, t, onClose]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg" isCentered>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{t('actions.deployAlgo')}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack spacing={4} align="stretch">
            <Box>
              <Text fontWeight="bold" mb={2}>
                {t('fields.name')}:
              </Text>
              <Text>{node.name}</Text>
            </Box>
            <Box>
              <Text fontWeight="bold" mb={2}>
                {t('actions.deployAlgoDesc')}:
              </Text>
              <Select
                placeholder={tCommon('filter.select')}
                value={selectedId}
                onChange={(e) => setSelectedId(e.target.value)}
                isDisabled={isLoading || algoPackages.length === 0}
              >
                {algoPackages.map((pkg) => (
                  <option key={pkg.id} value={pkg.id}>
                    {pkg.algorithm_alias || pkg.algorithm_name} v{pkg.version}
                  </option>
                ))}
              </Select>
              {algoPackages.length === 0 && !isLoading && (
                <Text color="gray.400" fontSize="sm" mt={1}>
                  {tCommon('empty.title')} - {tCommon('empty.description')}
                </Text>
              )}
            </Box>
          </VStack>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose}>
            {tCommon('cancel')}
          </Button>
          <Button
            colorScheme="blue"
            onClick={handleDeploy}
            isLoading={isDeploying}
            isDisabled={!selectedId}
          >
            {t('actions.deployConfirm')}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
