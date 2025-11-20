import { CustomSelect } from '@/components/ui/custom-select';
import { useItemStore } from '@/lib/store';
import { QUALITIES } from '@shared/constants';

export const QualitySelect = () => {
  const selectedQuality = useItemStore((state) => state.selectedQuality);
  const setSelectedQuality = useItemStore((state) => state.setSelectedQuality);

  return (
    <CustomSelect
      options={QUALITIES}
      value={selectedQuality}
      onValueChange={(value) => setSelectedQuality(parseInt(value))}
      placeholder="Select quality"
    />
  );
};
