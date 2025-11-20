export interface ItemData {
  UniqueName: string;
  LocalizationNameVariable?: string;
  LocalizationDescriptionVariable?: string;
  LocalizedNames?: {
    'EN-US'?: string;
    'DE-DE'?: string;
    'FR-FR'?: string;
    'RU-RU'?: string;
    'PL-PL'?: string;
    'ES-ES'?: string;
    'PT-BR'?: string;
    'IT-IT'?: string;
    'ZH-CN'?: string;
    'KO-KR'?: string;
    'JA-JP'?: string;
    'ZH-TW'?: string;
    'ID-ID'?: string;
    'TR-TR'?: string;
    'AR-SA'?: string;
    [key: string]: string | undefined;
  };
  LocalizedDescriptions?: {
    [key: string]: string | undefined;
  };
  Index?: string;
  [key: string]: unknown;
}

export interface Item {
  uniqueName: string;
  baseUniqueName: string;
  label: string;
}

export type Locale =
  | 'EN-US'
  | 'DE-DE'
  | 'FR-FR'
  | 'RU-RU'
  | 'PL-PL'
  | 'ES-ES'
  | 'PT-BR'
  | 'IT-IT'
  | 'ZH-CN'
  | 'KO-KR'
  | 'JA-JP'
  | 'ZH-TW'
  | 'ID-ID'
  | 'TR-TR'
  | 'AR-SA';
