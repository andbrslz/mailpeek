import type { ReactNode } from "react";
import type { PaneContent } from "../hooks/useInboxState";
import { cn } from "../lib/cn";

export function MessagePane({
  hiddenOnPhone,
  content,
  emptyInbox,
  message,
  error,
}: {
  hiddenOnPhone: boolean;
  content: PaneContent;
  emptyInbox: ReactNode;
  message: ReactNode;
  error: ReactNode;
}) {
  const views: Record<PaneContent, ReactNode> = {
    "empty-inbox": emptyInbox,
    message,
    error,
    nothing: null,
  };
  return (
    <main className={cn("min-w-0 flex-1 flex-col", hiddenOnPhone ? "hidden md:flex" : "flex")}>
      {views[content]}
    </main>
  );
}
