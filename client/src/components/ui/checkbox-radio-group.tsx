import { useState } from 'react';
import { Checkbox } from '@/components/ui/checkbox';
import { Field, FieldLabel } from '@/components/ui/field';

interface CheckboxRadioOption {
  id: string;
  label: string;
}

interface CheckboxRadioGroupProps {
  options: CheckboxRadioOption[];
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  name: string;
}

export const CheckboxRadioGroup = ({
  options,
  defaultValue,
  onValueChange,
  name,
}: CheckboxRadioGroupProps) => {
  const [selectedValue, setSelectedValue] = useState<string>(
    defaultValue || options[0]?.id || ''
  );

  const handleChange = (optionId: string) => {
    setSelectedValue(optionId);
    onValueChange?.(optionId);
  };

  return (
    <section className="flex flex-col gap-1 items-center [&>*]:h-[32px]">
      {options.map((option) => (
        <Field key={option.id} orientation="horizontal">
          <FieldLabel className="text-primary w-24 max-w-[308px] !flex-initial">
            {option.label}
          </FieldLabel>
          <Checkbox
            id={option.id}
            name={name}
            checked={selectedValue === option.id}
            onCheckedChange={() => handleChange(option.id)}
            className="border-muted data-[state=checked]:border-muted data-[state=checked]:bg-muted"
          />
        </Field>
      ))}
    </section>
  );
};
