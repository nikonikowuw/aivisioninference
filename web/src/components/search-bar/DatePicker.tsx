import { useState } from 'react';
import Calendar from 'react-calendar';
import {
  Input,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
  useColorModeValue,
  Box,
  IconButton,
  HStack,
} from '@chakra-ui/react';
import { ChevronLeftIcon, ChevronRightIcon } from '@chakra-ui/icons';
import { useTranslation } from 'react-i18next';
import { zhCN, zhTW, enUS, id, ja, ko, type Locale } from 'date-fns/locale';
import type { LanguageCode } from '../../i18n/types';
import 'react-calendar/dist/Calendar.css';
import '../../assets/css/MiniCalendar.css';

type Value = Date | null;

const localeMap: Record<LanguageCode, Locale> = {
  'zh-CN': zhCN,
  'zh-TW': zhTW,
  'en-US': enUS,
  'id-ID': id,
  'ja-JP': ja,
  'ko-KR': ko,
};

const clearLabelMap: Record<LanguageCode, string> = {
  'zh-CN': '清除',
  'zh-TW': '清除',
  'en-US': 'Clear',
  'id-ID': 'Bersihkan',
  'ja-JP': 'クリア',
  'ko-KR': '초기화',
};

interface DatePickerProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

function formatDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function DatePicker({ value, onChange, placeholder }: DatePickerProps) {
  const { i18n, t } = useTranslation();
  const lang = (i18n.language as LanguageCode) || 'zh-CN';
  const [isOpen, setIsOpen] = useState(false);
  const selected = value ? new Date(value) : null;

  const popBg = useColorModeValue('white', 'navy.800');
  const popBorder = useColorModeValue('gray.200', 'whiteAlpha.100');
  const popShadow = useColorModeValue('lg', '2xl');

  const handleChange = (val: Value | [Value, Value]) => {
    const d = Array.isArray(val) ? val[0] : val;
    if (d) {
      onChange(formatDate(d));
    } else {
      onChange('');
    }
    setIsOpen(false);
  };

  return (
    <Popover isOpen={isOpen} onClose={() => setIsOpen(false)} placement="bottom-start" isLazy>
      <PopoverTrigger>
        <Input
          variant="main"
          readOnly
          value={value || ''}
          onClick={() => setIsOpen(!isOpen)}
          placeholder={placeholder || t('searchBar.selectDate', '选择日期')}
          cursor="pointer"
        />
      </PopoverTrigger>
      <PopoverContent
        bg={popBg}
        border="1px solid"
        borderColor={popBorder}
        borderRadius="16px"
        shadow={popShadow}
        w="auto"
        _focus={{ boxShadow: 'none' }}
      >
        <PopoverBody p={2}>
          <Calendar
            onChange={handleChange}
            value={selected}
            prevLabel={<IconButton aria-label="prev" icon={<ChevronLeftIcon />} size="sm" variant="ghost" />}
            nextLabel={<IconButton aria-label="next" icon={<ChevronRightIcon />} size="sm" variant="ghost" />}
            prev2Label={null}
            next2Label={null}
            showNeighboringMonth={false}
            locale={lang}
          />
        </PopoverBody>
        {selected && (
          <HStack justify="center" pb={2} px={3}>
            <Box
              as="button"
              fontSize="xs"
              color="red.400"
              _hover={{ color: 'red.300' }}
              onClick={() => { onChange(''); setIsOpen(false); }}
            >
              {clearLabelMap[lang]}
            </Box>
          </HStack>
        )}
      </PopoverContent>
    </Popover>
  );
}
