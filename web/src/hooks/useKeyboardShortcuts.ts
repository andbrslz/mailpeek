import { useEffect, useMemo, type RefObject } from "react";

const ignoredTargets =
  'input, textarea, select, [contenteditable], dialog, [role="dialog"], [role="listbox"], [aria-haspopup]';

export function useKeyboardShortcuts({
  ids,
  selectedId,
  open,
  remove,
  searchRef,
  toggleList,
}: {
  ids: string[];
  selectedId: string | null;
  open: (id: string) => void;
  remove: (id: string) => void;
  searchRef: RefObject<HTMLInputElement | null>;
  toggleList: () => void;
}) {
  const actions = useMemo(
    () => ({
      move(delta: number) {
        const index = selectedId ? ids.indexOf(selectedId) : -1;
        const id = ids[Math.min(Math.max(index + delta, 0), ids.length - 1)];
        if (id) open(id);
      },
      focusSearch: () => searchRef.current?.focus(),
      toggleList,
      remove: () => selectedId && remove(selectedId),
    }),
    [ids, selectedId, open, remove, searchRef, toggleList],
  );

  useEffect(() => {
    const keys: Record<string, () => void> = {
      j: () => actions.move(1),
      ArrowDown: () => actions.move(1),
      k: () => actions.move(-1),
      ArrowUp: () => actions.move(-1),
      "/": actions.focusSearch,
      "[": actions.toggleList,
      Delete: () => void actions.remove(),
    };
    const onKey = (e: KeyboardEvent) => {
      const action = keys[e.key];
      const target = e.target as HTMLElement;
      if (!action || e.metaKey || e.ctrlKey || e.altKey || target.closest(ignoredTargets)) return;
      e.preventDefault();
      action();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [actions]);
}
