import { createContext, useContext } from "react";
import type { InboxState } from "../hooks/useInboxState";

export const InboxContext = createContext<InboxState | null>(null);

export function useInbox(): InboxState {
  const inbox = useContext(InboxContext);
  if (!inbox) throw new Error("useInbox must be used inside <InboxProvider>");
  return inbox;
}
