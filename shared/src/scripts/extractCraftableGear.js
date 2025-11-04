/**
 * Extract Craftable Gear Script
 * 
 * This script extracts all craftable gear items from items.json
 * and generates a craftableGear.json file with unique item names (without tier prefixes).
 * 
 * This pre-processed list allows for:
 * - Fast O(1) lookups using Set
 * - Smaller exclusion list (no need for long EXCLUDED_ITEM_TYPES)
 * - Easier maintenance (just re-run this script when items.json updates)
 * 
 * Run: node extractCraftableGear.js
 */

const fs = require('fs');
const path = require('path');

const items = JSON.parse(
  fs.readFileSync(
    path.join(__dirname, '../constants/items.json'),
    'utf-8'
  )
);

const EXCLUDED_ITEM_TYPES = [
  'JOURNAL', 'TOME', 'CONTRACT', 'ARENA', 'CAPEITEM', 'CAPE_CLOTH',
  'SKILLBOOK', 'TOOL', 'TRACKINGITEM', 'ARTEFACT', 'ESSENCE', 'SHARD',
  'MOUNT', 'SILVERBAG', 'LOOTCHEST', 'CORRUPTED', 'DUNGEON', 'FARM',
  'POTION', 'SKIN', 'UNIQUE', 'HELLGATE', 'RUNE', 'SOUL', 'RELIC', 'FURNITURE'
];

const TIER_PREFIXES = [
  "Beginner's ",
  "Novice's ",
  "Journeyman's ",
  "Adept's ",
  "Expert's ",
  "Master's ",
  "Grandmaster's ",
  "Elder's "
];

function removeTierPrefix(name) {
  for (const prefix of TIER_PREFIXES) {
    if (name.startsWith(prefix)) {
      return name.substring(prefix.length);
    }
  }
  return name;
}

const craftableGearSet = new Set();

items.forEach(item => {
  const uniqueName = item.UniqueName;
  const itemName = item.LocalizedNames && item.LocalizedNames['EN-US'];
  
  if (!itemName) return;
  
  // Check if excluded
  const isExcluded = EXCLUDED_ITEM_TYPES.some(type => uniqueName.includes(type));
  if (isExcluded) return;
  
  // Check if has tier prefix
  const hasTierPrefix = TIER_PREFIXES.some(prefix => itemName.startsWith(prefix));
  if (!hasTierPrefix) return;
  
  // Add the item name without tier prefix
  const baseName = removeTierPrefix(itemName);
  craftableGearSet.add(baseName);
});

const craftableGearList = Array.from(craftableGearSet).sort();

console.log(`Found ${craftableGearList.length} unique craftable gear items`);

// Write to file
const outputPath = path.join(__dirname, '../constants/craftableGear.json');
fs.writeFileSync(outputPath, JSON.stringify(craftableGearList, null, 2));

console.log(`Written to ${outputPath}`);
