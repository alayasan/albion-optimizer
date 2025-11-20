import { CustomSelect } from '@/components/ui/custom-select';
import { useItemStore } from '@/lib/store';
import { ZONE_QUALITIES } from '@shared/constants';

export const ZoneQualitySelect = () => {
  const zoneQuality = useItemStore((state) => state.zoneQuality);
  const setZoneQuality = useItemStore((state) => state.setZoneQuality);

  return (
    <CustomSelect
      options={ZONE_QUALITIES}
      value={zoneQuality}
      onValueChange={(value) => setZoneQuality(parseInt(value))}
      placeholder="Select zone quality"
    />
  );
};
