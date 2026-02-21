"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";

interface TenantStore {
  selectedTenantId: string | null;
  setSelectedTenantId: (id: string | null) => void;
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
}

export const useTenantStore = create<TenantStore>()(
  persist(
    (set) => ({
      selectedTenantId: null,
      setSelectedTenantId: (id) => set({ selectedTenantId: id }),
      sidebarCollapsed: false,
      toggleSidebar: () =>
        set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
    }),
    { name: "zanguard-console" }
  )
);
