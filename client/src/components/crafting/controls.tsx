import { Field, FieldLabel } from '@/components/ui/field';
import { ItemSelect } from './item-select';
import { QualitySelect } from './quality-select';
import { ZoneQualitySelect } from './zone-quality-select';
import { HideoutPowerSelect } from './hideout-power-select';
import { DailyBonusSelect } from './daily-bonus-select';
import { Checkbox } from '@/components/ui/checkbox';
import { CheckboxRadioGroup } from '@/components/ui/checkbox-radio-group';
import { ItemImage } from '@/components/ui/item-image';
import { useItemStore } from '@/lib/store';
import { Input } from '@/components/ui/input';
import { CRAFTING_LOCATIONS } from '@shared/constants';

export const CraftingControls = () => {
  const selectedItem = useItemStore((state) => state.selectedItem);
  const craftingLocation = useItemStore((state) => state.craftingLocation);
  const quantity = useItemStore((state) => state.quantity);
  const stationFee = useItemStore((state) => state.stationFee);
  const refreshData = useItemStore((state) => state.refreshData);
  const useFocus = useItemStore((state) => state.useFocus);
  const setCraftingLocation = useItemStore(
    (state) => state.setCraftingLocation
  );
  const setQuantity = useItemStore((state) => state.setQuantity);
  const setStationFee = useItemStore((state) => state.setStationFee);
  const setRefreshData = useItemStore((state) => state.setRefreshData);
  const setUseFocus = useItemStore((state) => state.setUseFocus);

  return (
    <header className="flex gap-10 items-center justify-around bg-primary-foreground px-10 py-5">
      <section className="flex flex-col gap-1 items-center [&>*]:h-[32px]">
        <Field orientation="horizontal">
          <FieldLabel htmlFor="item" className="text-primary w-24">
            Item:
          </FieldLabel>
          <ItemSelect />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24">Quality:</FieldLabel>
          <QualitySelect />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24 max-w-[308px] !flex-initial">
            Refresh Data
          </FieldLabel>
          <Checkbox
            id="refresh-data"
            checked={refreshData}
            onCheckedChange={setRefreshData}
            className="border-muted data-[state=checked]:border-muted data-[state=checked]:bg-muted"
          />
        </Field>
      </section>
      {selectedItem && (
        <ItemImage
          src={`https://render.albiononline.com/v1/item/T8_${selectedItem.baseUniqueName}`}
          alt={selectedItem.label}
          className="w-24 h-24"
        />
      )}
      <section className="flex flex-col gap-1 items-center [&>*]:h-[32px]">
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24">Quantity:</FieldLabel>
          <Input
            id="quantity"
            type="number"
            value={quantity}
            onChange={(e) => setQuantity(Number(e.target.value))}
            className="w-[200px] border-none p-0 text-muted focus-visible:ring-0 focus-visible:ring-offset-0 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
            placeholder="999"
          />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24">Station Fee:</FieldLabel>
          <Input
            id="station-fee"
            type="number"
            value={stationFee}
            onChange={(e) => setStationFee(Number(e.target.value))}
            className="w-[200px] border-none p-0 text-muted focus-visible:ring-0 focus-visible:ring-offset-0 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
            placeholder="999"
          />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24 max-w-[308px] !flex-initial">
            Use Focus
          </FieldLabel>
          <Checkbox
            id="use-focus"
            checked={useFocus}
            onCheckedChange={setUseFocus}
            className="border-muted data-[state=checked]:border-muted data-[state=checked]:bg-muted"
          />
        </Field>
      </section>
      <CheckboxRadioGroup
        options={CRAFTING_LOCATIONS}
        name="crafting-location"
        defaultValue={craftingLocation}
        onValueChange={setCraftingLocation}
      />
      <section className="flex flex-col gap-1 items-center [&>*]:h-[32px]">
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24">Zone Quality:</FieldLabel>
          <ZoneQualitySelect />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24">Hideout Power:</FieldLabel>
          <HideoutPowerSelect />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-primary w-24">Daily Bonus:</FieldLabel>
          <DailyBonusSelect />
        </Field>
      </section>
    </header>
  );
};
