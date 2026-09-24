import { EmptyInbox } from "../components/EmptyState";
import { ErrorBanner } from "../components/ErrorBanner";
import { Header } from "../components/Header";
import { ListArea } from "../components/ListArea";
import { MessagePane } from "../components/MessagePane";
import { MessageView } from "../components/MessageView";
import { ResizeHandle } from "../components/ResizeHandle";
import { Sidebar } from "../components/Sidebar";
import { SidebarFooter } from "../components/SidebarFooter";
import { useInbox } from "../context/inbox";
import { useLayout } from "../context/layout";
import { useI18n } from "../i18n/context";

export function InboxPage() {
  const inbox = useInbox();
  const layout = useLayout();
  const { t } = useI18n();
  const { message, server } = inbox;

  return (
    <div className="flex h-full flex-col">
      <Header
        query={inbox.query}
        onQueryChange={inbox.setQuery}
        onClear={inbox.clearAll}
        canClear={inbox.hasMessages}
        searchRef={layout.searchRef}
        soundEnabled={inbox.sound.enabled}
        onToggleSound={inbox.sound.toggle}
        listHidden={layout.listHidden}
        onToggleList={layout.toggleList}
        onSignOut={inbox.signOut}
      />
      {inbox.error && (
        <ErrorBanner>
          {t("app.loadError", { error: t(inbox.error.key, inbox.error.vars) })}
        </ErrorBanner>
      )}
      <div className="flex min-h-0 flex-1">
        <Sidebar
          count={inbox.total}
          width={layout.sidebar.width}
          hidden={layout.listHidden}
          hiddenOnPhone={layout.listHiddenOnPhone}
          animate={!layout.resizing}
          footer={
            inbox.hasMessages && (
              <SidebarFooter
                smtpAddress={server.smtpAddress}
                smtpAuth={server.smtpAuth}
                status={inbox.status}
              />
            )
          }
        >
          <ListArea
            content={inbox.listContent}
            messages={inbox.messages}
            hasMore={inbox.hasMore}
            onLoadMore={inbox.loadMore}
            selectedId={inbox.selectedId}
            isUnread={inbox.isUnread}
            onSelect={layout.open}
            search={inbox.search}
          />
        </Sidebar>
        {layout.showResizeHandle && (
          <ResizeHandle
            width={layout.sidebar.width}
            onResize={layout.sidebar.setWidth}
            onReset={layout.sidebar.reset}
            onDraggingChange={layout.setResizing}
          />
        )}
        <MessagePane
          hiddenOnPhone={layout.paneHiddenOnPhone}
          content={inbox.paneContent}
          emptyInbox={<EmptyInbox smtpAddress={server.smtpAddress} smtpAuth={server.smtpAuth} />}
          message={
            message && (
              <MessageView
                key={message.id}
                message={message}
                tab={layout.tab}
                onTabChange={layout.setTab}
                collapsed={layout.detailsCollapsed}
                onToggleCollapsed={layout.toggleDetails}
                onDelete={() => void inbox.remove(message.id)}
                onBack={layout.back}
              />
            )
          }
          error={
            <div className="p-8 text-sm text-zinc-500">
              {inbox.messageError &&
                t("app.messageError", {
                  error: t(inbox.messageError.key, inbox.messageError.vars),
                })}
            </div>
          }
        />
      </div>
    </div>
  );
}
