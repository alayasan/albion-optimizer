import { Field, FieldLabel } from '@/components/ui/field';
import { ItemSelect } from './item-select';
import { QualitySelect } from './quality-select';
import { Checkbox } from '@/components/ui/checkbox';
import { ItemImage } from '@/components/ui/item-image';
import { useItemStore } from '@/lib/store';

export const CraftingControls = () => {
  const selectedItem = useItemStore((state) => state.selectedItem);

  return (
    <header className="flex gap-10 items-center justify-around bg-primary-foreground px-10 py-5">
      <section className="flex flex-col gap-1 items-center">
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
        <Field orientation="horizontal" className="h-[32px]">
          <FieldLabel className="text-primary w-24 max-w-[308px] !flex-initial">
            Refresh Data
          </FieldLabel>
          <Checkbox
            id="refresh-data"
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
    </header>
  );
};
