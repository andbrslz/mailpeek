import { createContext, useContext } from "react";
import type { LayoutState } from "../hooks/useLayoutState";

export const LayoutContext = createContext<LayoutState | null>(null);

export function useLayout(): LayoutState {
  const layout = useContext(LayoutContext);
  if (!layout) throw new Error("useLayout must be used inside <LayoutProvider>");
  return layout;
}
