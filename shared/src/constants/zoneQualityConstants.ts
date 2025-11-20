export const ZONE_QUALITY_LEVELS = {
  LEVEL_1: 1,
  LEVEL_2: 2,
  LEVEL_3: 3,
  LEVEL_4: 4,
  LEVEL_5: 5,
  LEVEL_6: 6,
} as const;

export type ZoneQualityLevel = typeof ZONE_QUALITY_LEVELS[keyof typeof ZONE_QUALITY_LEVELS];

export interface ZoneQuality {
  value: number;
  label: string;
}

export const ZONE_QUALITIES: ZoneQuality[] = [
  { value: ZONE_QUALITY_LEVELS.LEVEL_1, label: '1' },
  { value: ZONE_QUALITY_LEVELS.LEVEL_2, label: '2' },
  { value: ZONE_QUALITY_LEVELS.LEVEL_3, label: '3' },
  { value: ZONE_QUALITY_LEVELS.LEVEL_4, label: '4' },
  { value: ZONE_QUALITY_LEVELS.LEVEL_5, label: '5' },
  { value: ZONE_QUALITY_LEVELS.LEVEL_6, label: '6' },
];
