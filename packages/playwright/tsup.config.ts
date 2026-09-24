import { defineConfig } from "tsup";

export default defineConfig({
  entry: ["src/index.ts"],
  format: ["esm", "cjs"],
  // tsup sets the deprecated baseUrl option internally when emitting types.
  dts: { compilerOptions: { ignoreDeprecations: "6.0" } },
  clean: true,
  target: "es2022",
  external: ["@playwright/test", "@mailpeek/client"],
});
