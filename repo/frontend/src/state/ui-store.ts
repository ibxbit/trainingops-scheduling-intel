import { create } from "zustand";

type UiState = {
  activeTenantDate: string | null;
  contentSearch: string;
  setActiveTenantDate: (date: string | null) => void;
  setContentSearch: (query: string) => void;
};

export const useUiStore = create<UiState>((set) => ({
  activeTenantDate: null,
  contentSearch: "",
  setActiveTenantDate: (activeTenantDate) => set({ activeTenantDate }),
  setContentSearch: (contentSearch) => set({ contentSearch }),
}));
