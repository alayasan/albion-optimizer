import { Field, FieldGroup, FieldLabel, FieldSet } from '@/components/ui/field';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover';
import { Input } from '@/components/ui/input';
import { ChevronDown } from 'lucide-react';
import { useState } from 'react';

export const CraftingControls = () => {
  const [open, setOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // Mock data - replace with your actual items
  const items = [
    'Calendar',
    'Search Emoji',
    'Calculator',
    'Settings',
    'Profile',
    'Keyboard',
  ];

  const filteredItems = items.filter((item) =>
    item.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleSelect = (selectedValue: string) => {
    setSearchQuery(selectedValue);
    setOpen(false);
  };

  return (
    <form>
      <FieldGroup className="text-white grid grid-cols-4 grid-rows-2">
        <FieldSet>
          <Field
            orientation="horizontal"
            className="gap-2 [&>[data-slot=field-label]]:flex-none"
          >
            <FieldLabel htmlFor="item" className="capitalize w-fit">
              item:
            </FieldLabel>
            <Popover open={open} onOpenChange={setOpen}>
              <PopoverTrigger asChild>
                <div className="relative flex items-center">
                  <Input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => {
                      setSearchQuery(e.target.value);
                      setOpen(true);
                    }}
                    onClick={() => setOpen(true)}
                    placeholder="Enter item..."
                    className="pr-8 text-white cursor-text"
                  />
                  <ChevronDown className="absolute right-3 size-4 shrink-0 opacity-50 pointer-events-none" />
                </div>
              </PopoverTrigger>
              <PopoverContent
                className="p-0 w-[--radix-popover-trigger-width]"
                align="start"
                onOpenAutoFocus={(e) => e.preventDefault()}
              >
                <Command shouldFilter={false}>
                  <CommandList>
                    {filteredItems.length === 0 ? (
                      <CommandEmpty>No results found.</CommandEmpty>
                    ) : (
                      <CommandGroup>
                        {filteredItems.map((item) => (
                          <CommandItem
                            key={item}
                            value={item}
                            onSelect={() => handleSelect(item)}
                          >
                            <span>{item}</span>
                          </CommandItem>
                        ))}
                      </CommandGroup>
                    )}
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </Field>
        </FieldSet>
      </FieldGroup>
    </form>
  );
};
