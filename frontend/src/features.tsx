import { createContext, useContext } from "react";

// Switches (backend features.go): which optional rooms this village has.
// App loads them with the session and provides them here. A room behind a
// switch needs only `feature` on its ROOMS line in App.tsx: Home, the bottom
// bar and the routes then leave it out while it is off. A door that is not a
// ROOMS entry (a chip, a link inside another room) asks `useFeature(key)`.
// set_by and set_at come with a flipped switch; a row with neither is one the
// village carried over from before switches existed (migration 025).
export type FeatureRow = { key: string; on: boolean; default: boolean; set_by?: string | null; set_at?: string | null };

export const FeaturesContext = createContext<{ list: FeatureRow[]; reload: () => Promise<void> }>({ list: [], reload: async () => {} });

export function useFeatures() {
  return useContext(FeaturesContext);
}

export function useFeature(key: string): boolean {
  return !!useContext(FeaturesContext).list.find((f) => f.key === key)?.on;
}
