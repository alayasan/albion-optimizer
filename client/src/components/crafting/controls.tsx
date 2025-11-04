import { FieldGroup, FieldSet } from '@/components/ui/field';
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group';
import { Label } from '@/components/ui/label';
import { ChevronDown } from 'lucide-react';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { useState } from 'react';
import itemsData from '@shared/constants/items.json';
import { processItems } from '@shared/utils/itemUtils';
import type { Item, ItemData } from '@shared/types/items';

// Process items using shared utility
const items: Item[] = processItems(itemsData as ItemData[]);

export const CraftingControls = () => {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  // Filter items based on search query
  const filteredItems = searchQuery
    ? items.filter((item) =>
        item.label.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : items;

  return (
    <form>
      <FieldGroup className="text-white grid grid-cols-4 grid-rows-2">
        <FieldSet className="flex-row gap-2">
          <Label htmlFor="item" className="capitalize w-fit">
            item:
          </Label>
          <Popover open={open} onOpenChange={setOpen}>
            <InputGroup className="border-0">
              <InputGroupInput
                id="item"
                placeholder="Enter item..."
                value={value}
                onChange={(e) => {
                  setValue(e.target.value);
                  setSearchQuery(e.target.value);
                }}
                onFocus={() => setOpen(true)}
              />
              <InputGroupAddon align="inline-end" className="cursor-pointer">
                <PopoverTrigger>
                  <ChevronDown size={16} />
                </PopoverTrigger>
              </InputGroupAddon>
              <PopoverContent className="w-[400px] p-0" align="start">
                <Command>
                  <CommandInput
                    placeholder="Search items..."
                    value={searchQuery}
                    onValueChange={setSearchQuery}
                  />
                  <CommandList>
                    <CommandEmpty>No items found.</CommandEmpty>
                    <CommandGroup>
                      {filteredItems.map((item) => (
                        <CommandItem
                          key={item.uniqueName}
                          value={item.label}
                          onSelect={(currentValue) => {
                            setValue(currentValue);
                            setSearchQuery('');
                            setOpen(false);
                          }}
                        >
                          {item.label}
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </CommandList>
                </Command>
              </PopoverContent>
            </InputGroup>
          </Popover>
        </FieldSet>
      </FieldGroup>
    </form>
  );
};
