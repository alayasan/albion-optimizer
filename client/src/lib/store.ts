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
  setSelectedItem: (item: Item | null) => void;
  setSelectedQuality: (quality: number) => void;
}

export const useItemStore = create<ItemStore>((set) => ({
  selectedItem: storedState?.selectedItem ?? defaultItem,
  selectedQuality: storedState?.selectedQuality ?? 2, // Default to Good quality
  setSelectedItem: (item: Item | null) => {
    set({ selectedItem: item });
    saveToStorage({ selectedItem: item });
  },
  setSelectedQuality: (quality: number) => {
    set({ selectedQuality: quality });
    saveToStorage({ selectedQuality: quality });
  },
}));
