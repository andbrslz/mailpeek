import { useCallback, useEffect, useMemo } from "react";
import { clearMessages, deleteMessage, signOut, type MessageSummary } from "../api";
import { playChime } from "../lib/sound";
import { useArrivals } from "./useArrivals";
import { useEvents, type EventHandler } from "./useEvents";
import { useFaviconBadge } from "./useFaviconBadge";
import { pruneMessageCache, useMessage } from "./useMessage";
import { useMessages } from "./useMessages";
import { useSearchQuery } from "./useSearchQuery";
import { useSelection } from "./useSelection";
import { useServerInfo } from "./useServerInfo";
import { useSoundSetting } from "./useSoundSetting";

export type ListContent = "messages" | "no-results" | "placeholder";
export type PaneContent = "empty-inbox" | "message" | "error" | "nothing";

function listContent(hasMessages: boolean, loaded: boolean, search: string): ListContent {
  if (hasMessages) return "messages";
  return loaded && search ? "no-results" : "placeholder";
}

function paneContent(emptyInbox: boolean, hasMessage: boolean, hasError: boolean): PaneContent {
  if (emptyInbox) return "empty-inbox";
  if (hasMessage) return "message";
  return hasError ? "error" : "nothing";
}

export function useInboxState() {
  const { query, setQuery, search } = useSearchQuery();
  const { messages, total, hasMore, loaded, error, refresh, loadMore } = useMessages(search);
  const server = useServerInfo();
  const sound = useSoundSetting();

  const chime = useCallback(() => {
    if (sound.enabled) playChime();
  }, [sound.enabled]);
  const arrivals = useArrivals(chime);
  const onEvent = useCallback<EventHandler>(
    (type, id) => {
      refresh();
      arrivals.onEvent(type, id);
    },
    [refresh, arrivals],
  );
  const status = useEvents(onEvent);

  const { selectedId, seen, select, neighbourOf } = useSelection(messages, loaded);
  const { message, error: messageError } = useMessage(selectedId);
  useEffect(() => {
    pruneMessageCache(new Set(messages.map((m) => m.id)));
  }, [messages]);

  const unreadIds = useMemo(
    () => new Set(arrivals.arrived.filter((id) => !seen.has(id) && id !== selectedId)),
    [arrivals.arrived, seen, selectedId],
  );
  const isUnread = useCallback((m: MessageSummary) => unreadIds.has(m.id), [unreadIds]);
  useFaviconBadge(unreadIds.size);

  const remove = useCallback(
    async (id: string) => {
      select(neighbourOf(id));
      await deleteMessage(id).catch(() => undefined);
      refresh();
    },
    [select, neighbourOf, refresh],
  );
  const clearAll = useCallback(async () => {
    await clearMessages().catch(() => undefined);
    refresh();
  }, [refresh]);

  return {
    query,
    setQuery,
    search,
    messages,
    total,
    hasMore,
    loadMore,
    loaded,
    error,
    status,
    server,
    sound,
    selectedId,
    select,
    message,
    messageError,
    isUnread,
    remove,
    clearAll,
    signOut: server.uiAuth ? signOut : undefined,
    hasMessages: messages.length > 0,
    isEmptyInbox: loaded && messages.length === 0 && !search,
    listContent: listContent(messages.length > 0, loaded, search),
    paneContent: paneContent(loaded && messages.length === 0 && !search, !!message, !!messageError),
  };
}

export type InboxState = ReturnType<typeof useInboxState>;
