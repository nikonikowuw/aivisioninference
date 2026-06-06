import {
  Box,
  FormControl,
  FormLabel,
  FormHelperText,
  Input,
  NumberInput,
  NumberInputField,
  NumberInputStepper,
  NumberIncrementStepper,
  NumberDecrementStepper,
  Slider,
  SliderTrack,
  SliderFilledTrack,
  SliderThumb,
  SliderMark,
  Switch,
  Select,
  Text,
  VStack,
  HStack,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';

interface SchemaProperty {
  type: string;
  title?: string;
  description?: string;
  default?: any;
  minimum?: number;
  maximum?: number;
  step?: number;
  enum?: string[];
  enumNames?: string[];
}

interface DynamicParamsFormProps {
  schema: Record<string, any> | null | undefined;
  value: Record<string, any>;
  onChange: (value: Record<string, any>) => void;
}

export default function DynamicParamsForm({ schema, value, onChange }: DynamicParamsFormProps) {
  const { t } = useTranslation(['modules/ai-tasks']);
  if (!schema || !schema.properties) {
    return (
      <Box p={3} bg="gray.50" borderRadius="md">
        <Text color="gray.500" fontSize="sm">{t('form.noParams')}</Text>
      </Box>
    );
  }

  const properties = schema.properties as Record<string, SchemaProperty>;
  const required = (schema.required as string[]) || [];

  const updateParam = (key: string, val: any) => {
    onChange({ ...value, [key]: val });
  };

  const getValue = (key: string, prop: SchemaProperty) => {
    if (value[key] !== undefined) return value[key];
    if (prop.default !== undefined) return prop.default;
    if (prop.type === 'boolean') return false;
    if (prop.type === 'number' || prop.type === 'integer') return prop.minimum ?? 0;
    return '';
  };

  const renderField = (key: string, prop: SchemaProperty) => {
    const currentValue = getValue(key, prop);
    const isRequired = required.includes(key);

    switch (prop.type) {
      case 'number':
        return renderNumberField(key, prop, currentValue, isRequired);
      case 'integer':
        return renderIntegerField(key, prop, currentValue, isRequired);
      case 'boolean':
        return renderBooleanField(key, prop, currentValue, isRequired);
      case 'string':
        if (prop.enum) {
          return renderSelectField(key, prop, currentValue, isRequired);
        }
        return renderStringField(key, prop, currentValue, isRequired);
      default:
        return renderStringField(key, prop, currentValue, isRequired);
    }
  };

  const renderNumberField = (key: string, prop: SchemaProperty, currentValue: number, isRequired: boolean) => {
    const min = prop.minimum ?? 0;
    const max = prop.maximum ?? 1;
    const step = prop.step ?? 0.01;

    return (
      <FormControl key={key} isRequired={isRequired}>
        <FormLabel fontSize="sm">{prop.title || key}</FormLabel>
        {prop.description && <FormHelperText fontSize="xs" mb={2}>{prop.description}</FormHelperText>}
        <HStack spacing={4}>
          <Slider
            flex={1}
            min={min}
            max={max}
            step={step}
            value={currentValue}
            onChange={(val) => updateParam(key, val)}
          >
            <SliderTrack>
              <SliderFilledTrack />
            </SliderTrack>
            <SliderThumb />
            <SliderMark value={min} mt={2} fontSize="xs" color="gray.500">{min}</SliderMark>
            <SliderMark value={max} mt={2} fontSize="xs" color="gray.500">{max}</SliderMark>
          </Slider>
          <NumberInput
            w="100px"
            min={min}
            max={max}
            step={step}
            value={currentValue}
            onChange={(_, val) => updateParam(key, isNaN(val) ? currentValue : val)}
            precision={2}
          >
            <NumberInputField />
            <NumberInputStepper>
              <NumberIncrementStepper />
              <NumberDecrementStepper />
            </NumberInputStepper>
          </NumberInput>
        </HStack>
      </FormControl>
    );
  };

  const renderIntegerField = (key: string, prop: SchemaProperty, currentValue: number, isRequired: boolean) => {
    const min = prop.minimum ?? 0;
    const max = prop.maximum ?? 100;

    return (
      <FormControl key={key} isRequired={isRequired}>
        <FormLabel fontSize="sm">{prop.title || key}</FormLabel>
        {prop.description && <FormHelperText fontSize="xs" mb={2}>{prop.description}</FormHelperText>}
        <NumberInput
          min={min}
          max={max}
          value={currentValue}
          onChange={(_, val) => updateParam(key, isNaN(val) ? currentValue : val)}
        >
          <NumberInputField />
          <NumberInputStepper>
            <NumberIncrementStepper />
            <NumberDecrementStepper />
          </NumberInputStepper>
        </NumberInput>
      </FormControl>
    );
  };

  const renderBooleanField = (key: string, prop: SchemaProperty, currentValue: boolean, isRequired: boolean) => {
    return (
      <FormControl key={key} isRequired={isRequired} display="flex" alignItems="center">
        <Box flex={1}>
          <FormLabel fontSize="sm" mb={0}>{prop.title || key}</FormLabel>
          {prop.description && <FormHelperText fontSize="xs">{prop.description}</FormHelperText>}
        </Box>
        <Switch
          isChecked={currentValue}
          onChange={(e) => updateParam(key, e.target.checked)}
        />
      </FormControl>
    );
  };

  const renderSelectField = (key: string, prop: SchemaProperty, currentValue: string, isRequired: boolean) => {
    return (
      <FormControl key={key} isRequired={isRequired}>
        <FormLabel fontSize="sm">{prop.title || key}</FormLabel>
        {prop.description && <FormHelperText fontSize="xs" mb={2}>{prop.description}</FormHelperText>}
        <Select
          value={currentValue}
          onChange={(e) => updateParam(key, e.target.value)}
        >
          {prop.enum?.map((opt, idx) => (
            <option key={opt} value={opt}>
              {prop.enumNames?.[idx] || opt}
            </option>
          ))}
        </Select>
      </FormControl>
    );
  };

  const renderStringField = (key: string, prop: SchemaProperty, currentValue: string, isRequired: boolean) => {
    return (
      <FormControl key={key} isRequired={isRequired}>
        <FormLabel fontSize="sm">{prop.title || key}</FormLabel>
        {prop.description && <FormHelperText fontSize="xs" mb={2}>{prop.description}</FormHelperText>}
        <Input
          value={currentValue}
          onChange={(e) => updateParam(key, e.target.value)}
          placeholder={prop.description}
        />
      </FormControl>
    );
  };

  return (
    <VStack spacing={4} align="stretch">
      {Object.entries(properties).map(([key, prop]) => renderField(key, prop))}
    </VStack>
  );
}
