import { useCallback, useEffect, useRef, useState } from "react";
import { countMessages, listMessages, type MessageSummary } from "../api";
import { describeError, type Problem } from "../lib/errors";

export const PAGE_SIZE = 50;

interface State {
  messages: MessageSummary[];
  total: number;
  nextCursor?: string;
  loaded: boolean;
  error?: Problem;
}

export function useMessages(query: string) {
  const [state, setState] = useState<State>({ messages: [], total: 0, loaded: false });
  const current = useRef(state);
  const wanted = useRef(PAGE_SIZE);
  const generation = useRef(0);
  const synced = useRef(-1);
  const controller = useRef<AbortController | undefined>(undefined);
  const pending = useRef<number | undefined>(undefined);
  const queryRef = useRef(query);

  const load = useCallback(async () => {
    controller.current?.abort();
    const ac = new AbortController();
    controller.current = ac;
    const apply = (next: State) => {
      current.current = next;
      setState(next);
    };
    const { messages, nextCursor } = current.current;
    const append =
      synced.current === generation.current &&
      nextCursor !== undefined &&
      wanted.current > messages.length;
    const q = queryRef.current;
    try {
      if (append) {
        const limit = wanted.current - messages.length;
        const page = await listMessages(q, { cursor: nextCursor, limit }, ac.signal);
        if (ac.signal.aborted) return;
        apply({
          ...current.current,
          messages: [...messages, ...page.messages],
          nextCursor: page.nextCursor,
        });
      } else {
        const gen = generation.current;
        const [page, total] = await Promise.all([
          listMessages(q, { limit: Math.max(wanted.current, PAGE_SIZE) }, ac.signal),
          countMessages(q, ac.signal),
        ]);
        if (ac.signal.aborted) return;
        synced.current = gen;
        apply({ messages: page.messages, total, nextCursor: page.nextCursor, loaded: true });
      }
    } catch (err) {
      if (ac.signal.aborted) return;
      wanted.current = current.current.messages.length;
      apply({ ...current.current, loaded: true, error: describeError(err) });
    }
  }, []);

  const refresh = useCallback(() => {
    generation.current++;
    if (pending.current !== undefined) return;
    pending.current = window.setTimeout(() => {
      pending.current = undefined;
      void load();
    }, 30);
  }, [load]);

  const loadMore = useCallback(() => {
    const { messages, nextCursor } = current.current;
    if (nextCursor === undefined || wanted.current > messages.length) return;
    wanted.current = messages.length + PAGE_SIZE;
    void load();
  }, [load]);

  useEffect(() => {
    queryRef.current = query;
    wanted.current = PAGE_SIZE;
    generation.current++;
    void load();
  }, [query, load]);

  useEffect(
    () => () => {
      controller.current?.abort();
      window.clearTimeout(pending.current);
    },
    [],
  );

  return { ...state, hasMore: state.nextCursor !== undefined, refresh, loadMore };
}

export function useDebounced<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const t = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(t);
  }, [value, delay]);
  return debounced;
}
