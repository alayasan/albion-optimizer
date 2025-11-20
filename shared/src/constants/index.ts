export * from "./itemConstants"
export * from "./qualityConstants"

import craftableGearList from './craftableGear.json';

/**
 * Set of craftable gear item names (without tier prefixes)
 * This is used for fast lookups to determine if an item is craftable gear
 */
export const CRAFTABLE_GEAR_NAMES = new Set(craftableGearList);

export { craftableGearList };
