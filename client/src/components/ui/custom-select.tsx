import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface SelectOption {
  value: number | string;
  label: string;
}

interface CustomSelectProps {
  options: SelectOption[];
  value?: number | string;
  defaultValue?: number | string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}

export const CustomSelect = ({
  options,
  value,
  defaultValue,
  onValueChange,
  placeholder = 'Select an option',
  className,
}: CustomSelectProps) => {
  return (
    <Select
      value={value?.toString()}
      defaultValue={defaultValue?.toString()}
      onValueChange={onValueChange}
    >
      <SelectTrigger
        size="sm"
        className={`w-[200px] text-muted p-0 border-none [&_svg:not([class*='text-'])]:text-muted [&_svg:not([class*='text-'])]:opacity-100 data-[placeholder]:text-muted focus-visible:ring-0 focus-visible:ring-offset-0 cursor-pointer ${
          className || ''
        }`}
      >
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value.toString()}>
              {option.label}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
};
