import { CustomSelect } from '@/components/ui/custom-select';
import { useItemStore } from '@/lib/store';
import { DAILY_BONUSES } from '@shared/constants';

export const DailyBonusSelect = () => {
  const dailyBonus = useItemStore((state) => state.dailyBonus);
  const setDailyBonus = useItemStore((state) => state.setDailyBonus);

  return (
    <CustomSelect
      options={DAILY_BONUSES}
      value={dailyBonus}
      onValueChange={(value) => setDailyBonus(value)}
      placeholder="Select daily bonus"
    />
  );
};
