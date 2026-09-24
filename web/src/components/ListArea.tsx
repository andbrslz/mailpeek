import type { MessageSummary } from "../api";
import type { ListContent } from "../hooks/useInboxState";
import { NoResults } from "./EmptyState";
import { MessageList } from "./MessageList";
import { SidebarPlaceholder } from "./Sidebar";

export function ListArea({
  content,
  messages,
  hasMore,
  onLoadMore,
  selectedId,
  isUnread,
  onSelect,
  search,
}: {
  content: ListContent;
  messages: MessageSummary[];
  hasMore: boolean;
  onLoadMore: () => void;
  selectedId: string | null;
  isUnread: (m: MessageSummary) => boolean;
  onSelect: (id: string) => void;
  search: string;
}) {
  switch (content) {
    case "messages":
      return (
        <MessageList
          messages={messages}
          hasMore={hasMore}
          onLoadMore={onLoadMore}
          selectedId={selectedId}
          unread={isUnread}
          onSelect={onSelect}
        />
      );
    case "no-results":
      return <NoResults query={search} />;
    case "placeholder":
      return <SidebarPlaceholder />;
  }
}
