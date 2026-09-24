export type ConnectionStatus = "connecting" | "live" | "offline";

export const eventTypes = [
  "message.created",
  "message.deleted",
  "messages.cleared",
  "resync",
] as const;
export type MailpeekEvent = (typeof eventTypes)[number] | "connected";
export type EventHandler = (type: MailpeekEvent, id?: string) => void;

function eventId(e: MessageEvent): string | undefined {
  try {
    return (JSON.parse(e.data as string) as { id?: string }).id;
  } catch {
    return undefined;
  }
}

export function connectLiveEvents(
  url: string,
  onEvent: EventHandler,
  onStatus: (status: ConnectionStatus) => void,
): { close: () => void } {
  let source: EventSource | undefined;
  let retryTimer: number | undefined;
  let attempt = 0;
  let closed = false;
  const listeners = eventTypes.map((type) => ({
    type,
    listener: (e: MessageEvent) => onEvent(type, eventId(e)),
  }));

  const disconnect = () => {
    if (!source) return;
    for (const { type, listener } of listeners) source.removeEventListener(type, listener);
    source.onopen = null;
    source.onerror = null;
    source.close();
    source = undefined;
  };

  const connect = () => {
    const es = new EventSource(url);
    source = es;
    es.onopen = () => {
      attempt = 0;
      onStatus("live");
      onEvent("connected");
    };
    for (const { type, listener } of listeners) es.addEventListener(type, listener);
    es.onerror = () => {
      if (es.readyState !== EventSource.CLOSED) {
        onStatus("connecting");
        return;
      }
      onStatus("offline");
      disconnect();
      if (!closed) retryTimer = window.setTimeout(connect, Math.min(1000 * 2 ** attempt++, 15000));
    };
  };

  connect();
  return {
    close() {
      closed = true;
      window.clearTimeout(retryTimer);
      disconnect();
    },
  };
}
