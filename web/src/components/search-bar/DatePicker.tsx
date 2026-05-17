import React, { forwardRef } from 'react';
import ReactDatePicker, { registerLocale } from 'react-datepicker';
import { Input } from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { zhCN, zhTW, enUS, type Locale } from 'date-fns/locale';
import type { LanguageCode } from '../../i18n/types';

import 'react-datepicker/dist/react-datepicker.css';

const localeMap: Record<LanguageCode, Locale> = {
  'zh-CN': zhCN,
  'zh-TW': zhTW,
  'en-US': enUS,
};

registerLocale('zh-CN', zhCN);
registerLocale('zh-TW', zhTW);
registerLocale('en-US', enUS);

interface DatePickerProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  maxW?: string;
}

const ChakraInput = forwardRef<HTMLInputElement, { value?: string; onClick?: () => void; placeholder?: string }>(
  ({ value, onClick, placeholder }, ref) => (
    <Input
      ref={ref}
      readOnly
      value={value || ''}
      onClick={onClick}
      placeholder={placeholder}
      cursor="pointer"
    />
  ),
);
ChakraInput.displayName = 'ChakraInput';

export function DatePicker({ value, onChange, placeholder, maxW }: DatePickerProps) {
  const { i18n } = useTranslation();
  const locale = (i18n.language as LanguageCode) || 'zh-CN';

  const selected = value ? new Date(value) : null;

  const handleChange = (date: Date | null) => {
    if (date) {
      const y = date.getFullYear();
      const m = String(date.getMonth() + 1).padStart(2, '0');
      const d = String(date.getDate()).padStart(2, '0');
      onChange(`${y}-${m}-${d}`);
    } else {
      onChange('');
    }
  };

  const displayValue = selected
    ? `${selected.getFullYear()}-${String(selected.getMonth() + 1).padStart(2, '0')}-${String(selected.getDate()).padStart(2, '0')}`
    : '';

  return (
    <ReactDatePicker
      selected={selected}
      onChange={handleChange}
      locale={locale}
      dateFormat="yyyy-MM-dd"
      customInput={<ChakraInput value={displayValue} placeholder={placeholder} />}
      isClearable
      showPopperArrow={false}
      wrapperClassName="date-picker-wrapper"
    />
  );
}
