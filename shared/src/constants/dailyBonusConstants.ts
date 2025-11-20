export const DAILY_BONUS_LEVELS = {
  NONE: 'none',
  TEN_PERCENT: '10',
  TWENTY_PERCENT: '20',
} as const;

export type DailyBonusLevel = typeof DAILY_BONUS_LEVELS[keyof typeof DAILY_BONUS_LEVELS];

export interface DailyBonus {
  value: string;
  label: string;
}

export const DAILY_BONUSES: DailyBonus[] = [
  { value: DAILY_BONUS_LEVELS.NONE, label: 'None' },
  { value: DAILY_BONUS_LEVELS.TEN_PERCENT, label: '10%' },
  { value: DAILY_BONUS_LEVELS.TWENTY_PERCENT, label: '20%' },
];
