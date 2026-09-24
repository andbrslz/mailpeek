import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FocusEvent,
  type KeyboardEvent,
} from "react";

export function useListboxMenu<T extends string>({
  options,
  value,
  onSelect,
}: {
  options: readonly T[];
  value: T;
  onSelect: (value: T) => void;
}) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const list = useRef<HTMLDivElement>(null);

  const show = useCallback(() => {
    setActive(Math.max(0, options.indexOf(value)));
    setOpen(true);
  }, [options, value]);

  const hide = useCallback((refocus: boolean) => {
    setOpen(false);
    if (refocus) trigger.current?.focus();
  }, []);

  useEffect(() => {
    if (open) list.current?.querySelectorAll<HTMLElement>('[role="option"]')[active]?.focus();
  }, [open, active]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [open]);

  const choose = useCallback(
    (option: T) => {
      onSelect(option);
      hide(true);
    },
    [onSelect, hide],
  );

  const onTriggerKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key !== "ArrowDown" && e.key !== "ArrowUp") return;
      e.preventDefault();
      show();
    },
    [show],
  );

  const onListKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const last = options.length - 1;
      const moves: Record<string, (i: number) => number> = {
        ArrowDown: (i) => Math.min(i + 1, last),
        ArrowUp: (i) => Math.max(i - 1, 0),
        Home: () => 0,
        End: () => last,
      };
      const move = moves[e.key];
      if (e.key === "Escape") {
        e.preventDefault();
        hide(true);
      } else if (move) {
        e.preventDefault();
        setActive(move);
      }
    },
    [options.length, hide],
  );

  const onBlur = useCallback((e: FocusEvent) => {
    if (!root.current?.contains(e.relatedTarget as Node | null)) setOpen(false);
  }, []);

  return {
    open,
    active,
    rootRef: root,
    triggerRef: trigger,
    listRef: list,
    toggle: () => (open ? hide(false) : show()),
    choose,
    onTriggerKeyDown,
    onListKeyDown,
    onBlur,
  };
}
