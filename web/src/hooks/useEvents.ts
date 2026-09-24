import { useEffect, useRef, useState } from "react";
import { eventsUrl } from "../api";
import {
  connectLiveEvents,
  type ConnectionStatus,
  type EventHandler,
  type MailpeekEvent,
} from "../lib/liveEvents";

export type { ConnectionStatus, EventHandler, MailpeekEvent };

export function useEvents(onEvent: EventHandler): ConnectionStatus {
  const [status, setStatus] = useState<ConnectionStatus>("connecting");
  const handler = useRef(onEvent);
  useEffect(() => {
    handler.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    const connection = connectLiveEvents(
      eventsUrl,
      (type, id) => handler.current(type, id),
      setStatus,
    );
    return () => connection.close();
  }, []);

  return status;
}
