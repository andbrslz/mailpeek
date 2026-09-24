# Contributing

Thanks for helping. Bug reports, ideas and pull requests are welcome.

## Before you start

- For a bug, open an [issue](https://github.com/andbrslz/mailpeek/issues/new/choose) with the Mailpeek version, how you run it (binary, Docker, Compose) and the steps to reproduce.
- For a new feature, open an issue first so we can agree on the approach. Mailpeek deliberately stays small: see [Scope and alternatives](README.md#scope-and-alternatives) and [Not included](README.md#not-included-on-purpose).

## Development setup

You need Go 1.25+ and Node 20.19+.

```bash
npm ci
make build   # Web UI + binary in ./bin/mailpeek
make check   # every quality gate: lint, typecheck, build, tests, audit, React Doctor
make e2e     # Playwright suite against the binary
```

UI development with hot reload: `go run ./cmd/mailpeek` in one terminal and `npm run dev -w @mailpeek/web` in another. The UI conventions are in [web/README.md](web/README.md).

## Pull requests

- Keep each pull request focused on one change, with tests for new behaviour.
- `make check` must pass; CI runs the same gates and the E2E suite.
- Follow the style of the surrounding code. The code has no explanatory comments: names and tests carry the meaning.
- Use [Conventional Commits](https://www.conventionalcommits.org) for commit messages (`feat:`, `fix:`, `docs:`, `chore:`, `ci:`).
- User-visible changes get a line in [CHANGELOG.md](CHANGELOG.md).

## Releases

Maintainers bump `version` in `packages/client` and `packages/playwright`, add the changelog section and push a `vX.Y.Z` tag. The release workflow publishes the npm packages, the binaries and the Docker images.
