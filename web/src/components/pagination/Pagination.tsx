import {
  Button,
  Flex,
  Select,
  Spinner,
  Text,
  useColorModeValue,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  pageSizeOptions?: number[];
  isLoading?: boolean;
}

/** @precondition 1 <= current <= totalPages && totalPages >= 1 */
function getPageNumbers(current: number, totalPages: number): (number | '...')[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const pages: (number | '...')[] = [1];
  const left = Math.max(2, current - 1);
  const right = Math.min(totalPages - 1, current + 1);
  if (left > 2) pages.push('...');
  for (let i = left; i <= right; i++) pages.push(i);
  if (right < totalPages - 1) pages.push('...');
  pages.push(totalPages);
  return pages;
}

export default function Pagination({
  page,
  pageSize,
  total,
  onChange,
  onPageSizeChange,
  pageSizeOptions = [10, 20, 50, 100],
  isLoading = false,
}: PaginationProps) {
  const { t } = useTranslation('common');
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const safePage = Math.min(Math.max(1, page), totalPages);
  const from = total === 0 ? 0 : (safePage - 1) * pageSize + 1;
  const to = Math.min(safePage * pageSize, total);
  const pages = getPageNumbers(safePage, totalPages);
  const textColor = useColorModeValue('gray.600', 'gray.300');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.200');
  const hoverBg = useColorModeValue('gray.100', 'whiteAlpha.100');

  if (total === 0) return null;

  return (
    <Flex
      justify="center"
      align="center"
      wrap="wrap"
      gap={4}
      pt={4}
      pb={2}
      position="relative"
    >
      {isLoading && (
        <Spinner size="sm" color="brand.500" position="absolute" top="8px" right="0" />
      )}
      <Text fontSize="sm" color={textColor}>
        {t('pagination.showing', { from, to, total })}
      </Text>

      <Flex display={{ base: 'none', md: 'flex' }} align="center" gap={1}>
        <Button
          size="sm"
          variant="outline"
          minW="36px"
          minH="36px"
          isDisabled={safePage <= 1 || isLoading}
          onClick={() => onChange(safePage - 1)}
          aria-label={t('pagination.previous')}
          borderColor={borderColor}
          _hover={{ bg: hoverBg }}
        >
          ‹
        </Button>
        {pages.map((p, i) =>
          p === '...' ? (
            <Text key={`ellipsis-${i}`} px={1} fontSize="sm" color={textColor}>…</Text>
          ) : (
            <Button
              key={`page-${p}`}
              size="sm"
              minW="36px"
              minH="36px"
              variant={p === safePage ? 'solid' : 'outline'}
              colorScheme={p === safePage ? 'brand' : 'gray'}
              onClick={() => onChange(p)}
              isDisabled={isLoading}
              aria-label={t('pagination.goToPage', { page: p })}
              borderColor={p === safePage ? undefined : borderColor}
              _hover={p === safePage ? undefined : { bg: hoverBg }}
            >
              {p}
            </Button>
          ),
        )}
        <Button
          size="sm"
          variant="outline"
          minW="36px"
          minH="36px"
          isDisabled={safePage >= totalPages || isLoading}
          onClick={() => onChange(safePage + 1)}
          aria-label={t('pagination.next')}
          borderColor={borderColor}
          _hover={{ bg: hoverBg }}
        >
          ›
        </Button>
      </Flex>

      <Flex display={{ base: 'flex', md: 'none' }} align="center" gap={2}>
        <Button
          size="sm"
          variant="outline"
          minH="44px"
          minW="44px"
          isDisabled={safePage <= 1 || isLoading}
          onClick={() => onChange(safePage - 1)}
          aria-label={t('pagination.previous')}
          borderColor={borderColor}
        >
          ‹
        </Button>
        <Text fontSize="sm" color={textColor}>
          {t('pagination.pageOf', { page: safePage, totalPages })}
        </Text>
        <Button
          size="sm"
          variant="outline"
          minH="44px"
          minW="44px"
          isDisabled={safePage >= totalPages || isLoading}
          onClick={() => onChange(safePage + 1)}
          aria-label={t('pagination.next')}
          borderColor={borderColor}
        >
          ›
        </Button>
      </Flex>

      <Select
        size="sm"
        w="auto"
        value={pageSize}
        onChange={(e) => onPageSizeChange(Number(e.target.value))}
        isDisabled={isLoading}
        aria-label={t('pagination.pageSize')}
      >
        {pageSizeOptions.map((s) => (
          <option key={s} value={s}>
            {t('pagination.pageSize', { size: s })}
          </option>
        ))}
      </Select>
    </Flex>
  );
}
