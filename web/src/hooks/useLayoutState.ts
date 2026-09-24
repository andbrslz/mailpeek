import { useCallback, useMemo, useRef, useState } from "react";
import type { Tab } from "../lib/tabs";
import type { InboxState } from "./useInboxState";
import { useKeyboardShortcuts } from "./useKeyboardShortcuts";
import { useSidebarWidth } from "./useSidebarWidth";
import { useStoredToggle } from "./useStoredToggle";

export function useLayoutState(inbox: InboxState) {
  const [tab, setTab] = useState<Tab>("html");
  const [showDetail, setShowDetail] = useState(false);
  const [listHidden, toggleList] = useStoredToggle("mailpeek:sidebar-hidden", false);
  const [detailsCollapsed, toggleDetails] = useStoredToggle("mailpeek:header-collapsed", false);
  const sidebar = useSidebarWidth();
  const [resizing, setResizing] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);

  const { select, remove, messages, selectedId, hasMessages, isEmptyInbox } = inbox;
  const open = useCallback(
    (id: string) => {
      select(id);
      setShowDetail(true);
    },
    [select],
  );
  const removeOne = useCallback((id: string) => void remove(id), [remove]);
  const ids = useMemo(() => messages.map((m) => m.id), [messages]);
  useKeyboardShortcuts({ ids, selectedId, open, remove: removeOne, searchRef, toggleList });

  return {
    tab,
    setTab,
    showDetail,
    open,
    back: () => setShowDetail(false),
    listHidden,
    toggleList,
    detailsCollapsed,
    toggleDetails,
    sidebar,
    resizing,
    setResizing,
    searchRef,
    listHiddenOnPhone: (showDetail && hasMessages) || isEmptyInbox,
    paneHiddenOnPhone: !showDetail && hasMessages,
    showResizeHandle: !listHidden && !isEmptyInbox,
  };
}

export type LayoutState = ReturnType<typeof useLayoutState>;
