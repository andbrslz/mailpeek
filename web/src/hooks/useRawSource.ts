import { useEffect, useState } from "react";
import { getRaw } from "../api";
import { describeError, type Problem } from "../lib/errors";

type State = { id: string; text?: string; error?: Problem };

export function useRawSource(id: string) {
  const [state, setState] = useState<State>();
  useEffect(() => {
    let cancelled = false;
    getRaw(id)
      .then((text) => {
        if (!cancelled) setState({ id, text });
      })
      .catch((err: unknown) => {
        if (!cancelled) setState({ id, error: describeError(err) });
      });
    return () => {
      cancelled = true;
    };
  }, [id]);
  const current = state?.id === id ? state : undefined;
  return { text: current?.text, error: current?.error, loading: !current };
}
