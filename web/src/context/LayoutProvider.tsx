import type { ReactNode } from "react";
import { useLayoutState } from "../hooks/useLayoutState";
import { useInbox } from "./inbox";
import { LayoutContext } from "./layout";

export function LayoutProvider({ children }: { children: ReactNode }) {
  return (
    <LayoutContext.Provider value={useLayoutState(useInbox())}>{children}</LayoutContext.Provider>
  );
}
