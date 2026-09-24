import { useCallback, useState } from "react";
import type { MessageSummary } from "../api";

const addTo = (set: Set<string>, id: string) => (set.has(id) ? set : new Set(set).add(id));

export function useSelection(messages: MessageSummary[], loaded: boolean) {
  const [chosenId, setChosenId] = useState<string | null>(null);
  const [seen, setSeen] = useState<Set<string>>(() => new Set());

  const selectedId = messages.some((m) => m.id === chosenId) ? chosenId : (messages[0]?.id ?? null);
  if (loaded && selectedId !== chosenId) {
    setChosenId(selectedId);
    if (selectedId) setSeen((s) => addTo(s, selectedId));
  }

  const select = useCallback((id: string | null) => {
    setChosenId(id);
    if (id) setSeen((s) => addTo(s, id));
  }, []);

  const neighbourOf = useCallback(
    (id: string) => {
      const index = messages.findIndex((m) => m.id === id);
      return (messages[index + 1] ?? messages[index - 1])?.id ?? null;
    },
    [messages],
  );

  return { selectedId, seen, select, neighbourOf };
}
