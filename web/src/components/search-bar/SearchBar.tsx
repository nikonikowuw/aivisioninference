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
  showSearchButton?: boolean;
  onFilterChange: (key: string, value: string | undefined) => void;
  onSearch: () => void;
  onReset: () => void;
  filters: FilterState;
}

export function SearchBar({
  filters,
  onFilterChange,
  onSearch,
  onReset,
  keyword = true,
  selects,
  dateRange,
  showSearchButton = true,
}: SearchBarProps) {
  const { t } = useTranslation();
  const bgColor = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        onSearch();
      }
    },
    [onSearch],
  );

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

  const hasContent =
    filters.keyword ||
    selects?.some(s => filters[s.name]) ||
    (dateRange && (filters.start_time || filters.end_time));

  return (
    <Box
      bg={bgColor}
      p={4}
      borderRadius="lg"
      border="1px solid"
      borderColor={borderColor}
      mb={4}
    >
      <Flex gap={3} align="flex-end" wrap="wrap">
        {keyword && (
          <FormControl flex="1" minW="200px">
            <FormLabel fontSize="sm" mb={0}>{t('searchBar.keyword', '关键字')}</FormLabel>
            <Input
              variant="main"
              aria-label={t('searchBar.keyword', '关键字')}
              placeholder={t('searchBar.keywordPlaceholder', '搜索关键字...')}
              value={filters.keyword || ''}
              onChange={handleKeywordChange}
              onKeyDown={handleKeyDown}
            />
          </FormControl>
        )}

        {selects?.map(select => (
          <Box key={select.name} minW="150px">
            <FormControl>
              <FormLabel fontSize="sm" mb={0}>{select.label}</FormLabel>
              <Select
                variant="main"
                placeholder={select.placeholder || t('searchBar.all', 'All')}
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
              <FormLabel fontSize="sm" mb={0}>{t('searchBar.startDate', '开始日期')}</FormLabel>
              <DatePicker
                value={filters.start_time || ''}
                onChange={handleDateChange('start_time')}
                placeholder={t('searchBar.startDate', '开始日期')}
              />
            </FormControl>
            <FormControl maxW="180px">
              <FormLabel fontSize="sm" mb={0}>{t('searchBar.endDate', '结束日期')}</FormLabel>
              <DatePicker
                value={filters.end_time || ''}
                onChange={handleDateChange('end_time')}
                placeholder={t('searchBar.endDate', '结束日期')}
              />
            </FormControl>
          </Stack>
        )}

        {showSearchButton && (
          <Stack direction="row" spacing={2}>
            <Button colorScheme="blue" onClick={onSearch}>
              {t('searchBar.search', '搜索')}
            </Button>
            {hasContent && (
              <Button variant="outline" onClick={onReset}>
                {t('searchBar.reset', '重置')}
              </Button>
            )}
          </Stack>
        )}
        {!showSearchButton && hasContent && (
          <Button variant="outline" onClick={onReset}>
            {t('searchBar.reset', '重置')}
          </Button>
        )}
      </Flex>
    </Box>
  );
}
