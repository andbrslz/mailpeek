module github.com/andbrslz/mailpeek

go 1.25.0

require golang.org/x/net v0.58.0

// Keep Go tooling out of npm dependencies (some ship Go files).
ignore (
	./e2e/node_modules
	./examples/playwright/node_modules
	./node_modules
	./packages/client/node_modules
	./packages/playwright/node_modules
	./web/node_modules
)
