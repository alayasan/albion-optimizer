import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { useState } from 'react';
import { ChevronsUpDown, X } from 'lucide-react';
import { useItemStore } from '@/lib/store';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Button } from '@/components/ui/button';
import { processedItems as items } from '@/lib/items-cache';

export const ItemSelect = () => {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const selectedItem = useItemStore((state) => state.selectedItem);
  const setSelectedItem = useItemStore((state) => state.setSelectedItem);

  const filteredItems = searchQuery
    ? items.filter((item) =>
        item.label.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : items;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-[200px] h-[32px] justify-between text-muted font-normal bg-transparent border-none p-0 has-[>svg]:px-0 focus-visible:ring-0 focus-visible:ring-offset-0 hover:bg-transparent hover:text-muted"
        >
          {selectedItem ? selectedItem.label : 'Select an item'}
          <ChevronsUpDown strokeWidth={1} />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0">
        <Command>
          <CommandInput
            id="item"
            placeholder="Search items..."
            value={searchQuery}
            onValueChange={setSearchQuery}
            showSearchIcon={false}
            suffix={
              <X
                size={16}
                className="cursor-pointer opacity-50 hover:opacity-100 transition-opacity"
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setSearchQuery('');
                }}
              />
            }
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
                  onSelect={() => {
                    setSearchQuery(item.label);
                    setSelectedItem(item);
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
    </Popover>
  );
};
