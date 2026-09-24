import { useMemo } from "react";
import { HIGHLIGHT_LIMIT, highlightMime, type Token } from "../../lib/mimeHighlight";

const tokenClass: Record<Token["kind"], string> = {
  name: "font-semibold text-blue-700 dark:text-blue-400",
  value: "text-zinc-800 dark:text-zinc-200",
  encoded: "text-amber-700 dark:text-amber-400",
  boundary: "font-semibold text-violet-700 dark:text-violet-400",
  base64: "text-zinc-400 dark:text-zinc-600",
  body: "text-zinc-700 dark:text-zinc-300",
};

export function HighlightedMime({ raw }: { raw: string }) {
  const [head, rest] = useMemo(() => {
    const cut =
      raw.length > HIGHLIGHT_LIMIT ? raw.lastIndexOf("\n", HIGHLIGHT_LIMIT) + 1 : raw.length;
    return [highlightMime(raw.slice(0, cut)), raw.slice(cut)];
  }, [raw]);
  return (
    <pre className="p-5 pt-10 font-mono text-xs leading-relaxed break-all whitespace-pre-wrap">
      {head.map((line) => (
        <span key={line.offset}>
          {line.tokens.map((t) => (
            <span
              key={`${line.offset}:${t.start}`}
              data-token={t.kind}
              className={tokenClass[t.kind]}
            >
              {t.text}
            </span>
          ))}
          {"\n"}
        </span>
      ))}
      {rest && <span className={tokenClass.body}>{rest}</span>}
    </pre>
  );
}
