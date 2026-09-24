import type { ReactNode } from "react";
import { useInboxState } from "../hooks/useInboxState";
import { InboxContext } from "./inbox";

export function InboxProvider({ children }: { children: ReactNode }) {
  return <InboxContext.Provider value={useInboxState()}>{children}</InboxContext.Provider>;
}
