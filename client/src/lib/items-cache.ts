import itemsData from '@shared/constants/items.json';
import { processItems } from '@shared/utils/itemUtils';
import type { Item, ItemData } from '@shared/types/items';

/**
 * Cached processed items to avoid reprocessing on every import
 * This is computed once at module load time
 */
export const processedItems: Item[] = processItems(itemsData as ItemData[]);
