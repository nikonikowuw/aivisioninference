import React, { useCallback } from 'react';
import {
  Box,
  Button,
  Flex,
  FormControl,
  FormLabel,
  Input,
  Select,
  Stack,
  useColorModeValue,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { FilterState } from 'hooks/useFilter';
import { DatePicker } from './DatePicker';

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectConfig {
  name: string;
  label: string;
  options: SelectOption[];
  placeholder?: string;
}

export interface SearchBarProps {
  keyword?: boolean;
  selects?: SelectConfig[];
  dateRange?: boolean;
  onFilterChange: (key: string, value: string | undefined) => void;
  onReset: () => void;
  filters: FilterState;
  onRefresh?: () => void;
}

export function SearchBar({
  filters,
  onFilterChange,
  onReset,
  keyword = true,
  selects,
  dateRange,
  onRefresh,
}: SearchBarProps) {
  const { t } = useTranslation('common');
  const bgColor = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const handleKeywordChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onFilterChange('keyword', e.target.value || undefined);
    },
    [onFilterChange],
  );

  const handleSelectChange = useCallback(
    (name: string) => (e: React.ChangeEvent<HTMLSelectElement>) => {
      onFilterChange(name, e.target.value || undefined);
    },
    [onFilterChange],
  );

  const handleDateChange = useCallback(
    (name: string) => (value: string) => {
      onFilterChange(name, value || undefined);
    },
    [onFilterChange],
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (onRefresh) {
      onRefresh();
    }
  };

  return (
    <Box
      as="form"
      onSubmit={handleSubmit}
      bg={bgColor}
      p={4}
      borderRadius="lg"
      border="1px solid"
      borderColor={borderColor}
      mb={4}
    >
      <Flex justify="space-between" align="flex-end" wrap="wrap" gap={4}>
        <Box width={{ base: '100%', md: '30%' }}>
          {keyword && (
            <FormControl>
              <FormLabel fontSize="sm" mb={0}>{t('searchBar.keyword')}</FormLabel>
              <Input
                variant="main"
                aria-label={t('searchBar.keyword')}
                placeholder={t('searchBar.keywordPlaceholder')}
                value={filters.keyword || ''}
                onChange={handleKeywordChange}
              />
            </FormControl>
          )}
        </Box>

        <Flex gap={3} align="flex-end" wrap="wrap" flex="1" justify="flex-end">
          {onRefresh && (
            <Button variant="lightBrand" type="submit">
              {t('searchBar.search')}
            </Button>
          )}

          {selects?.map(select => (
            <Box key={select.name} minW="150px">
              <FormControl>
                <FormLabel fontSize="sm" mb={0}>{select.label}</FormLabel>
                <Select
                  variant="main"
                  placeholder={select.placeholder || t('searchBar.all')}
                  value={filters[select.name] || ''}
                  onChange={handleSelectChange(select.name)}
                >
                  {select.options.map(opt => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </Select>
              </FormControl>
            </Box>
          ))}

          {dateRange && (
            <Stack direction="row" spacing={2}>
              <FormControl maxW="180px">
                <FormLabel fontSize="sm" mb={0}>{t('searchBar.startDate')}</FormLabel>
                <DatePicker
                  value={filters.start_time || ''}
                  onChange={handleDateChange('start_time')}
                  placeholder={t('searchBar.startDate')}
                />
              </FormControl>
              <FormControl maxW="180px">
                <FormLabel fontSize="sm" mb={0}>{t('searchBar.endDate')}</FormLabel>
                <DatePicker
                  value={filters.end_time || ''}
                  onChange={handleDateChange('end_time')}
                  placeholder={t('searchBar.endDate')}
                />
              </FormControl>
            </Stack>
          )}

          <Button variant="lightBrand" onClick={onReset}>
            {t('searchBar.reset')}
          </Button>
        </Flex>
      </Flex>
    </Box>
  );
}