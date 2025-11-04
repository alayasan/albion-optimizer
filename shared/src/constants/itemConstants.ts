/**
 * Tier prefixes used in Albion Online item names
 */
export const TIER_PREFIXES = [
  "Beginner's ",
  "Novice's ",
  "Journeyman's ",
  "Adept's ",
  "Expert's ",
  "Master's ",
  "Grandmaster's ",
  "Elder's ",
] as const;

/**
 * Item types to exclude from craftable gear
 * Add any UniqueName patterns here that should not be considered craftable gear
 */
export const EXCLUDED_ITEM_TYPES = [
  'JOURNAL',
  'TOME',
  'CONTRACT',
  'ARENA',
  'CAPEITEM',
  'CAPE_CLOTH',
  'SKILLBOOK',
  'TOOL',
  'TRACKINGITEM',
  'ARTEFACT',
  'ESSENCE',
  'SHARD',
  'MOUNT',
  'SILVERBAG',
  'LOOTCHEST',
  'CORRUPTED',
  'DUNGEON',
  'FARM',
  'POTION',
  'SKIN',
  'UNIQUE',
  'HELLGATE',
  'RUNE',
  'SOUL',
  'RELIC',
  'FURNITURE',
] as const;
