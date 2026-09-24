type RawToken =
  | { kind: "name"; text: string }
  | { kind: "value"; text: string }
  | { kind: "encoded"; text: string }
  | { kind: "boundary"; text: string }
  | { kind: "base64"; text: string }
  | { kind: "body"; text: string };

export type Token = RawToken & { start: number };

export const HIGHLIGHT_LIMIT = 256 * 1024;

const headerLine = /^([!-9;-~]+):(.*)$/;
const encodedWord = /=\?[^?\s]+\?[BbQq]\?[^?\s]*\?=/g;
const base64Line = /^[A-Za-z0-9+/]{40,}={0,2}$/;
const boundaryParam = /boundary\s*=\s*"?([^";\s]+)"?/gi;

function valueTokens(text: string): RawToken[] {
  const out: RawToken[] = [];
  let last = 0;
  for (const m of text.matchAll(encodedWord)) {
    const at = m.index ?? 0;
    if (at > last) out.push({ kind: "value", text: text.slice(last, at) });
    out.push({ kind: "encoded", text: m[0] });
    last = at + m[0].length;
  }
  if (last < text.length) out.push({ kind: "value", text: text.slice(last) });
  return out;
}

export interface Line {
  offset: number;
  tokens: Token[];
}

export function highlightMime(raw: string): Line[] {
  const lines: Line[] = [];
  let offset = 0;
  for (const raws of tokenize(raw)) {
    let start = 0;
    const tokens = raws.map((t) => {
      const token = { ...t, start };
      start += t.text.length;
      return token;
    });
    lines.push({ offset, tokens });
    offset += start + 1;
  }
  return lines;
}

function tokenize(raw: string): RawToken[][] {
  const boundaries = new Set<string>();
  for (const m of raw.matchAll(boundaryParam)) if (m[1]) boundaries.add(m[1]);

  const lines = raw.split(/\r?\n/);
  const out: RawToken[][] = [];
  let inHeaders = true;
  for (const line of lines) {
    if (line.startsWith("--")) {
      const name = line
        .slice(2)
        .replace(/--\s*$/, "")
        .trim();
      if (boundaries.has(name)) {
        out.push([{ kind: "boundary", text: line }]);
        inHeaders = !line.trimEnd().endsWith("--");
        continue;
      }
    }
    if (inHeaders) {
      if (line === "") {
        inHeaders = false;
        out.push([]);
        continue;
      }
      const m = line.match(headerLine);
      if (m) {
        out.push([{ kind: "name", text: `${m[1]}:` }, ...valueTokens(m[2] ?? "")]);
        continue;
      }
      if (/^[ \t]/.test(line)) {
        out.push(valueTokens(line));
        continue;
      }
    }
    out.push([{ kind: base64Line.test(line) ? "base64" : "body", text: line }]);
  }
  return out;
}
