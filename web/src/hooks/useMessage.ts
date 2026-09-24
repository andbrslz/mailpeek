import { useEffect, useState } from "react";
import { getMessage, type Message } from "../api";
import { describeError, type Problem } from "../lib/errors";

const cache = new Map<string, Message>();

export function pruneMessageCache(keep: Set<string>) {
  for (const id of cache.keys()) if (!keep.has(id)) cache.delete(id);
}

type State = { id?: string; message?: Message; error?: Problem };

export function useMessage(id: string | null) {
  const [state, setState] = useState<State>({});

  useEffect(() => {
    if (!id || cache.has(id)) return;
    let cancelled = false;
    getMessage(id)
      .then((message) => {
        cache.set(id, message);
        if (!cancelled) setState({ id, message });
      })
      .catch((err: unknown) => {
        if (!cancelled) setState({ id, error: describeError(err) });
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (!id) return { message: undefined, error: undefined, loading: false };
  const cached = cache.get(id);
  if (cached) return { message: cached, error: undefined, loading: false };
  const current = state.id === id ? state : {};
  return { message: current.message, error: current.error, loading: !current.error };
}
