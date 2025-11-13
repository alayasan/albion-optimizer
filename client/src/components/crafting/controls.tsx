import { Field, FieldLabel } from '@/components/ui/field';
import { ItemSelect } from './item-select';
import { Checkbox } from '@/components/ui/checkbox';
import { Select } from '@/components/ui/select';

export const CraftingControls = () => {
  return (
    <header className="bg-[#0d0c1d] h-fit w-auto px-10 py-5">
      <section className="grid grid-cols-[auto_1fr] gap-2.5 max-w-[15%] items-center">
        <Field orientation="horizontal">
          <FieldLabel htmlFor="item" className="text-white">
            Item:
          </FieldLabel>
          <ItemSelect />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-white">Quality:</FieldLabel>
          <Select />
        </Field>
        <Field orientation="horizontal">
          <FieldLabel className="text-white">Refresh Data</FieldLabel>
          <Checkbox id="refresh-data" />
        </Field>
      </section>
    </header>
  );
};
