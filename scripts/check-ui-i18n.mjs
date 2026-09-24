// Quality gate: every text the Web UI shows goes through t(), so it is
// translated. Flags JSX text and user-facing attributes written as literals.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { createRequire } from "node:module";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const ts = createRequire(import.meta.url)("typescript");
const root = fileURLToPath(new URL("../web/src", import.meta.url));

// Names and protocol terms that read the same in every language.
const allowed = new Set(["Mailpeek", "SMTP", "MAILPEEK_SMTP_AUTH", ".eml"]);
const textAttributes = new Set([
  "title",
  "aria-label",
  "aria-description",
  "placeholder",
  "alt",
  "label",
]);
const hasWords = (s) => /\p{L}{2,}/u.test(s) && !allowed.has(s.trim());

function* files(dir) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) {
      if (name !== "i18n") yield* files(path);
    } else if (name.endsWith(".tsx")) yield path;
  }
}

const found = [];
for (const path of files(root)) {
  const source = ts.createSourceFile(
    path,
    readFileSync(path, "utf8"),
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TSX,
  );
  const report = (node, text) => {
    const { line } = source.getLineAndCharacterOfPosition(node.getStart());
    found.push(`  ${relative(root, path)}:${line + 1}  ${JSON.stringify(text.trim())}`);
  };
  const visit = (node) => {
    if (ts.isJsxText(node) && hasWords(node.text)) report(node, node.text);
    if (
      ts.isJsxAttribute(node) &&
      textAttributes.has(node.name.getText()) &&
      node.initializer &&
      ts.isStringLiteral(node.initializer) &&
      hasWords(node.initializer.text)
    ) {
      report(node, node.initializer.text);
    }
    ts.forEachChild(node, visit);
  };
  visit(source);
}

if (found.length) {
  console.error("Untranslated UI text (add a key to web/src/i18n/locales and use t()):");
  console.error(found.join("\n"));
  process.exit(1);
}
console.log("UI i18n OK: no literal text in web/src components.");
