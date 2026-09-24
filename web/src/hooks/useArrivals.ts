import { useCallback, useState } from "react";
import type { EventHandler } from "./useEvents";

export function useArrivals(onCreated: () => void) {
  const [arrived, setArrived] = useState<string[]>([]);

  const onEvent = useCallback<EventHandler>(
    (type, id) => {
      if (type === "message.created" && id) {
        setArrived((a) => [...a, id]);
        onCreated();
      } else if (type === "message.deleted" && id) {
        setArrived((a) => a.filter((x) => x !== id));
      } else if (type === "messages.cleared") {
        setArrived([]);
      }
    },
    [onCreated],
  );

  return { arrived, onEvent };
}
