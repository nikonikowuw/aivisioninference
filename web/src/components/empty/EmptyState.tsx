import { Box, Button, Center, Flex, Icon, Stack, Text, useColorModeValue } from '@chakra-ui/react';
import React from 'react';
import { useTranslation } from 'react-i18next';
import { LuSearchX } from 'react-icons/lu';

interface EmptyStateProps {
  title?: string;
  description?: string;
  icon?: React.ElementType;
  onClearFilters?: () => void;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  title,
  description,
  icon = LuSearchX,
  onClearFilters,
}) => {
  const { t } = useTranslation('common');
  const textColor = useColorModeValue('navy.700', 'white');
  const descColor = useColorModeValue('gray.500', 'gray.400');

  return (
    <Center p={10}>
      <Stack align="center" spacing={4} textAlign="center">
        <Box
          p={6}
          borderRadius="full"
          bg={useColorModeValue('gray.50', 'whiteAlpha.50')}
          color="gray.400"
        >
          <Icon as={icon} boxSize={12} />
        </Box>
        <Stack spacing={1}>
          <Text fontSize="xl" fontWeight="bold" color={textColor}>
            {title || t('empty.title')}
          </Text>
          <Text fontSize="md" color={descColor}>
            {description || t('empty.description')}
          </Text>
        </Stack>
        {onClearFilters && (
          <Button
            variant="outline"
            onClick={onClearFilters}
            size="sm"
            fontWeight="500"
          >
            {t('button.clearFilters')}
          </Button>
        )}
      </Stack>
    </Center>
  );
};
