import { createContext, useContext } from "react";

// Switches (backend features.go): which optional rooms this village has.
// App loads them with the session and provides them here; a room or a
// door that belongs to a switch asks `useFeature("camp")` before it draws.
export type FeatureRow = { key: string; on: boolean; default: boolean; set_by?: string | null; set_at?: string };

export const FeaturesContext = createContext<{ list: FeatureRow[]; reload: () => Promise<void> }>({ list: [], reload: async () => {} });

export function useFeatures() {
  return useContext(FeaturesContext);
}

export function useFeature(key: string): boolean {
  return !!useContext(FeaturesContext).list.find((f) => f.key === key)?.on;
}
