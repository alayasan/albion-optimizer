import { CustomSelect } from '@/components/ui/custom-select';
import { useItemStore } from '@/lib/store';
import { HIDEOUT_POWERS } from '@shared/constants';

export const HideoutPowerSelect = () => {
  const hideoutPower = useItemStore((state) => state.hideoutPower);
  const setHideoutPower = useItemStore((state) => state.setHideoutPower);

  return (
    <CustomSelect
      options={HIDEOUT_POWERS}
      value={hideoutPower}
      onValueChange={(value) => setHideoutPower(parseInt(value))}
      placeholder="Select hideout power"
    />
  );
};
