export const HIDEOUT_POWER_LEVELS = {
  LEVEL_1: 1,
  LEVEL_2: 2,
  LEVEL_3: 3,
  LEVEL_4: 4,
  LEVEL_5: 5,
  LEVEL_6: 6,
  LEVEL_7: 7,
  LEVEL_8: 8,
  LEVEL_9: 9,
} as const;

export type HideoutPowerLevel = typeof HIDEOUT_POWER_LEVELS[keyof typeof HIDEOUT_POWER_LEVELS];

export interface HideoutPower {
  value: number;
  label: string;
}

export const HIDEOUT_POWERS: HideoutPower[] = [
  { value: HIDEOUT_POWER_LEVELS.LEVEL_1, label: '1' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_2, label: '2' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_3, label: '3' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_4, label: '4' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_5, label: '5' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_6, label: '6' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_7, label: '7' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_8, label: '8' },
  { value: HIDEOUT_POWER_LEVELS.LEVEL_9, label: '9' },
];
