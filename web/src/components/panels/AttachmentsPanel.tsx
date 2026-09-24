import { Download, File, FileArchive, FileImage, FileText } from "lucide-react";
import { attachmentUrl, type Attachment, type Message } from "../../api";
import { useI18n } from "../../i18n/context";
import { fileTypeLabel, formatBytes } from "../../lib/format";
import { Empty } from "./Empty";

function AttachmentIcon({ a }: { a: Attachment }) {
  const cls = "size-4 text-zinc-500";
  if (a.contentType.startsWith("image/")) return <FileImage className={cls} />;
  if (a.contentType.startsWith("text/") || a.contentType === "application/pdf")
    return <FileText className={cls} />;
  if (/zip|tar|gzip|compressed/.test(a.contentType)) return <FileArchive className={cls} />;
  return <File className={cls} />;
}

export function AttachmentsPanel({ message }: { message: Message }) {
  const { t, locale } = useI18n();
  if (message.attachments.length === 0) return <Empty>{t("attachments.none")}</Empty>;
  return (
    <ul className="divide-y divide-zinc-100 dark:divide-zinc-900">
      {message.attachments.map((a) => (
        <li key={a.id} className="flex items-center gap-3 px-5 py-3">
          <div className="flex size-9 shrink-0 items-center justify-center rounded-md border border-zinc-200 dark:border-zinc-800">
            <AttachmentIcon a={a} />
          </div>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-medium" title={a.filename}>
              {a.filename}
            </div>
            <div className="mt-0.5 flex items-center gap-2 text-xs text-zinc-500">
              <span className="font-medium">
                {fileTypeLabel(a.filename, a.contentType, t("attachment.file"))}
              </span>
              <span>{formatBytes(a.size, locale)}</span>
              {a.inline && (
                <span className="rounded bg-zinc-100 px-1.5 py-px dark:bg-zinc-800">
                  {t("attachments.inline")}
                </span>
              )}
            </div>
          </div>
          <a
            href={attachmentUrl(message.id, a.id)}
            download={a.filename}
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-zinc-200 px-2.5 text-sm font-medium hover:bg-zinc-50 dark:border-zinc-800 dark:hover:bg-zinc-900"
          >
            <Download className="size-3.5" /> {t("attachments.download")}
          </a>
        </li>
      ))}
    </ul>
  );
}
