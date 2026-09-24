// Quality gate: no Web UI component, page, hook or module over 300 lines.
// Anything longer is a sign it should be split into smaller pieces.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const LIMIT = 300;
const root = fileURLToPath(new URL("../web/src", import.meta.url));

function* files(dir) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) yield* files(path);
    else if (/\.(tsx?|css)$/.test(name)) yield path;
  }
}

const tooLong = [];
for (const path of files(root)) {
  const lines = readFileSync(path, "utf8").split("\n").length;
  if (lines > LIMIT) tooLong.push(`  ${relative(root, path)}: ${lines} lines`);
}
if (tooLong.length) {
  console.error(`UI files over ${LIMIT} lines (split them into smaller components/hooks):`);
  console.error(tooLong.join("\n"));
  process.exit(1);
}
console.log(`UI size OK: every file in web/src is at most ${LIMIT} lines.`);
