import type { Message } from "../../api";
import { useI18n } from "../../i18n/context";
import { Empty } from "./Empty";

export function TextPanel({ message }: { message: Message }) {
  const { t } = useI18n();
  if (!message.text) return <Empty>{t("text.none")}</Empty>;
  return (
    <pre className="p-5 font-mono text-[13px] leading-relaxed break-words whitespace-pre-wrap">
      {message.text}
    </pre>
  );
}
