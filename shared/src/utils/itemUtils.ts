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
 * Extract tier number from UniqueName
 * @param uniqueName - The unique name (e.g., "T3_MAIN_ARCANESTAFF")
 * @returns The tier number (3-8) or 0 if not found
 */
export function extractTier(uniqueName: string): number {
  const match = uniqueName.match(/^T(\d+)_/);
  return match ? parseInt(match[1] ?? '0', 10) : 0;
}

/**
 * Remove tier prefix from UniqueName
 * @param uniqueName - The unique name (e.g., "T3_MAIN_ARCANESTAFF")
 * @returns The unique name without tier prefix (e.g., "MAIN_ARCANESTAFF")
 */
export function stripTierFromUniqueName(uniqueName: string): string {
  return uniqueName.replace(/^T\d+_/, '');
}

/**
 * Get the base name for grouping items (without tier)
 * @param uniqueName - The unique name (e.g., "T3_MAIN_ARCANESTAFF")
 * @returns The base name for grouping
 */
export function getBaseName(uniqueName: string): string {
  return stripTierFromUniqueName(uniqueName);
}

/**
 * Process raw item data into a filtered and formatted list
 * @param rawItems - Array of raw item data
 * @param locale - The locale to use for item names (defaults to 'EN-US')
 * @param keepHighestTierOnly - Whether to keep only the highest tier version of each item (defaults to true)
 * @returns Filtered and formatted array of items
 */
// Items to exclude from the list
const EXCLUDED_ITEMS = ['Black Hands'];

export function processItems(
  rawItems: ItemData[],
  locale: Locale = 'EN-US',
  keepHighestTierOnly: boolean = true
): Item[] {
  const filteredItems = rawItems
    .filter(
      (item) =>
        item.LocalizedNames?.[locale] &&
        isCraftableGear(item.LocalizedNames[locale]!)
    )
    .map((item) => ({
      uniqueName: item.UniqueName,
      baseUniqueName: stripTierFromUniqueName(item.UniqueName),
      label: removeTierPrefix(item.LocalizedNames![locale]!),
      tier: extractTier(item.UniqueName),
    }))
    .filter((item) => !EXCLUDED_ITEMS.includes(item.label));

  let processedItems: Item[];

  // Keep only the highest tier version of each item
  if (keepHighestTierOnly) {
    const itemMap = new Map<string, Item & { tier: number }>();

    for (const item of filteredItems) {
      // Group by label to handle different variants (e.g., MAIN_ARCANESTAFF vs 2H_ARCANESTAFF)
      const key = item.label;
      const existing = itemMap.get(key);
      if (!existing || item.tier > existing.tier) {
        itemMap.set(key, item);
      }
    }

    processedItems = Array.from(itemMap.values()).map(({ uniqueName, baseUniqueName, label }) => ({
      uniqueName,
      baseUniqueName,
      label,
    }));
  } else {
    processedItems = filteredItems.map(({ uniqueName, baseUniqueName, label }) => ({
      uniqueName,
      baseUniqueName,
      label,
    }));
  }

  // Sort alphabetically
  return processedItems.sort((a, b) => a.label.localeCompare(b.label));
}
