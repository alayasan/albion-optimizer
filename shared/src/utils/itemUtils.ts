import type { ItemData, Item, Locale } from '../types/items';
import { TIER_PREFIXES } from '../constants/itemConstants';
import { CRAFTABLE_GEAR_NAMES } from '../constants';

/**
 * Check if an item is a craftable gear item
 * @param itemName - The localized name of the item (with tier prefix)
 * @returns True if the item is craftable gear, false otherwise
 */
export function isCraftableGear(itemName: string): boolean {
  // Remove tier prefix first
  const baseName = removeTierPrefix(itemName);

  // Check if the base name is in our craftable gear set
  return CRAFTABLE_GEAR_NAMES.has(baseName);
}

/**
 * Remove tier prefix from item name
 * @param name - The item name with tier prefix
 * @returns The item name without tier prefix
 * @example
 * removeTierPrefix("Adept's Knight Helmet") // Returns "Knight Helmet"
 */
export function removeTierPrefix(name: string): string {
  for (const prefix of TIER_PREFIXES) {
    if (name.startsWith(prefix)) {
      return name.substring(prefix.length);
    }
  }
  return name;
}

/**
 * Process raw item data into a filtered and formatted list
 * @param rawItems - Array of raw item data
 * @param locale - The locale to use for item names (defaults to 'EN-US')
 * @param removeDuplicates - Whether to remove duplicate items after removing tier prefixes (defaults to true)
 * @returns Filtered and formatted array of items
 */
export function processItems(
  rawItems: ItemData[],
  locale: Locale = 'EN-US',
  removeDuplicates: boolean = true
): Item[] {
  let processedItems = rawItems
    .filter(
      (item) =>
        item.LocalizedNames?.[locale] &&
        isCraftableGear(item.LocalizedNames[locale]!)
    )
    .map((item) => ({
      uniqueName: item.UniqueName,
      label: removeTierPrefix(item.LocalizedNames![locale]!),
    }));

  // Remove duplicates by label (since we removed tier prefixes)
  if (removeDuplicates) {
    processedItems = processedItems.filter(
      (item, index, self) =>
        index === self.findIndex((i) => i.label === item.label)
    );
  }

  // Sort alphabetically
  return processedItems.sort((a, b) => a.label.localeCompare(b.label));
}
