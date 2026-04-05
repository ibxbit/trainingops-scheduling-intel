import { create } from "zustand";

import type { SessionUser } from "../auth/roles";

type SessionState = {
  user: SessionUser | null;
  isReady: boolean;
  setUser: (user: SessionUser | null) => void;
  setReady: (ready: boolean) => void;
  clearSession: () => void;
};

export const useSessionStore = create<SessionState>((set) => ({
  user: null,
  isReady: false,
  setUser: (user) => set({ user }),
  setReady: (isReady) => set({ isReady }),
  clearSession: () => set({ user: null, isReady: true }),
}));
