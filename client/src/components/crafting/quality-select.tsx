import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useItemStore } from '@/lib/store';
import { QUALITIES } from '@shared/constants';

export const QualitySelect = () => {
  const setSelectedQuality = useItemStore((state) => state.setSelectedQuality);

  return (
    <Select
      defaultValue="2"
      onValueChange={(value) => setSelectedQuality(parseInt(value))}
    >
      <SelectTrigger
        size="sm"
        className="w-[200px] text-muted p-0 border-none [&_svg:not([class*='text-'])]:text-muted [&_svg:not([class*='text-'])]:opacity-100 data-[placeholder]:text-muted focus-visible:ring-0 focus-visible:ring-offset-0 cursor-pointer"
      >
        <SelectValue placeholder="Select quality" />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          {QUALITIES.map((quality) => (
            <SelectItem key={quality.value} value={quality.value.toString()}>
              {quality.label}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
};
