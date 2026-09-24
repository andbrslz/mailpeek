# Changelog

All notable changes to Mailpeek, the `@mailpeek-dev/client` and `@mailpeek-dev/playwright` packages and the Docker image. Versions follow [semantic versioning](https://semver.org); while in 0.x, a minor version may contain breaking changes, and they are called out here.

## 0.2.0

### Added

- `--data-dir` / `MAILPEEK_DATA_DIR`: keep messages in a directory so they survive restarts. Each message is a plain `.eml` file with a small `.json` next to it. Off by default. The Docker image has a writable `/data` for a volume.
- The banner shows the data directory and how many messages were loaded.

### Changed

- `--max-messages` now defaults to 1000 (was 100), so large parallel suites no longer need configuration. Memory stays bounded by `--max-store-size` (256 MB).
- The Go module is now `github.com/andbrslz/mailpeek`, matching the repository.

## 0.1.1

### Fixed

- `mailpeekWebServer()` starts the published image `4ndbrslz/mailpeek` by default.
- The Web UI no longer overwrites the saved email list width when it opens.

## 0.1.0

First release: SMTP server, Web UI, REST and wait API, SMTP failure simulation, `@mailpeek-dev/client` and the `@mailpeek-dev/playwright` fixture.
