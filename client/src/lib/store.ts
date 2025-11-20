import { create } from 'zustand';
import type { Item } from '@shared/types/items';
import { processedItems } from './items-cache';

// Get the first item as default from cached processed items
const defaultItem = processedItems.length > 0 ? processedItems[0] : null;

// Load from localStorage if available
const loadFromStorage = () => {
  try {
    const stored = localStorage.getItem('albion-optimizer-storage');
    if (stored) {
      return JSON.parse(stored);
    }
  } catch (error) {
    console.error('Failed to load from localStorage:', error);
  }
  return null;
};

// Save to localStorage
const saveToStorage = (state: Partial<ItemStore>) => {
  try {
    localStorage.setItem('albion-optimizer-storage', JSON.stringify(state));
  } catch (error) {
    console.error('Failed to save to localStorage:', error);
  }
};

const storedState = loadFromStorage();

interface ItemStore {
  selectedItem: Item | null;
  selectedQuality: number;
  craftingLocation: string;
  quantity: number;
  stationFee: number;
  refreshData: boolean;
  useFocus: boolean;
  zoneQuality: number;
  hideoutPower: number;
  dailyBonus: string;
  setSelectedItem: (item: Item | null) => void;
  setSelectedQuality: (quality: number) => void;
  setCraftingLocation: (location: string) => void;
  setQuantity: (quantity: number) => void;
  setStationFee: (fee: number) => void;
  setRefreshData: (refresh: boolean) => void;
  setUseFocus: (focus: boolean) => void;
  setZoneQuality: (quality: number) => void;
  setHideoutPower: (power: number) => void;
  setDailyBonus: (bonus: string) => void;
}

export const useItemStore = create<ItemStore>((set, get) => ({
  selectedItem: storedState?.selectedItem ?? defaultItem,
  selectedQuality: storedState?.selectedQuality ?? 2, // Default to Good quality
  craftingLocation: storedState?.craftingLocation ?? 'bonus-city',
  quantity: storedState?.quantity ?? 1,
  stationFee: storedState?.stationFee ?? 0,
  refreshData: storedState?.refreshData ?? false,
  useFocus: storedState?.useFocus ?? false,
  zoneQuality: storedState?.zoneQuality ?? 2,
  hideoutPower: storedState?.hideoutPower ?? 1,
  dailyBonus: storedState?.dailyBonus ?? 'none',
  setSelectedItem: (item: Item | null) => {
    set({ selectedItem: item });
    saveToStorage({ ...get(), selectedItem: item });
  },
  setSelectedQuality: (quality: number) => {
    set({ selectedQuality: quality });
    saveToStorage({ ...get(), selectedQuality: quality });
  },
  setCraftingLocation: (location: string) => {
    set({ craftingLocation: location });
    saveToStorage({ ...get(), craftingLocation: location });
  },
  setQuantity: (quantity: number) => {
    set({ quantity });
    saveToStorage({ ...get(), quantity });
  },
  setStationFee: (fee: number) => {
    set({ stationFee: fee });
    saveToStorage({ ...get(), stationFee: fee });
  },
  setRefreshData: (refresh: boolean) => {
    set({ refreshData: refresh });
    saveToStorage({ ...get(), refreshData: refresh });
  },
  setUseFocus: (focus: boolean) => {
    set({ useFocus: focus });
    saveToStorage({ ...get(), useFocus: focus });
  },
  setZoneQuality: (quality: number) => {
    set({ zoneQuality: quality });
    saveToStorage({ ...get(), zoneQuality: quality });
  },
  setHideoutPower: (power: number) => {
    set({ hideoutPower: power });
    saveToStorage({ ...get(), hideoutPower: power });
  },
  setDailyBonus: (bonus: string) => {
    set({ dailyBonus: bonus });
    saveToStorage({ ...get(), dailyBonus: bonus });
  },
}));
