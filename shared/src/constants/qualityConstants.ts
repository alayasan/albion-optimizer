export const QUALITY_LEVELS = {
  NORMAL: 1,
  GOOD: 2,
  OUTSTANDING: 3,
  EXCELLENT: 4,
  MASTERPIECE: 5,
} as const;

export type QualityLevel = typeof QUALITY_LEVELS[keyof typeof QUALITY_LEVELS];

export interface Quality {
  value: number;
  label: string;
}

export const QUALITIES: Quality[] = [
  { value: QUALITY_LEVELS.NORMAL, label: 'Normal' },
  { value: QUALITY_LEVELS.GOOD, label: 'Good' },
  { value: QUALITY_LEVELS.OUTSTANDING, label: 'Outstanding' },
  { value: QUALITY_LEVELS.EXCELLENT, label: 'Excellent' },
  { value: QUALITY_LEVELS.MASTERPIECE, label: 'Masterpiece' },
];
