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
import { ChevronsUpDown, X } from 'lucide-react';

// Process items using shared utility
const items: Item[] = processItems(itemsData as ItemData[]);

export const ItemSelect = () => {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const filteredItems = searchQuery
    ? items.filter((item) =>
        item.label.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : items;

  const handleBlur = () => {
    setTimeout(() => setOpen(false), 200);
  };

  return (
    <>
      <Command>
        <CommandInput
          id="item"
          placeholder="Search items..."
          value={searchQuery}
          onValueChange={setSearchQuery}
          onFocus={() => setOpen(true)}
          showSearchIcon={false}
          suffix={
            <>
              {searchQuery ? (
                <X
                  size={16}
                  className="cursor-pointer opacity-50 hover:opacity-100 transition-opacity"
                  onClick={() => {
                    setSearchQuery('');
                    setOpen(true);
                  }}
                />
              ) : (
                <ChevronsUpDown
                  size={16}
                  className="cursor-pointer"
                  onClick={() => setOpen(true)}
                />
              )}
            </>
          }
          onBlur={handleBlur}
        />
        <CommandList
          className={`transition-all duration-300 ease-in-out no-scrollbar ${
            open
              ? 'max-h-[300px] opacity-100'
              : 'max-h-0 opacity-0 overflow-hidden'
          }`}
        >
          <CommandEmpty>No items found.</CommandEmpty>
          <CommandGroup>
            {filteredItems.map((item) => (
              <CommandItem
                key={item.uniqueName}
                value={item.label}
                onSelect={(currentValue) => {
                  setSearchQuery(currentValue);
                  setOpen(false);
                }}
              >
                {item.label}
              </CommandItem>
            ))}
          </CommandGroup>
        </CommandList>
      </Command>
    </>
  );
};
