# Performance and footprint

Measured on 2026-09-23 (re-checked after the Web UI additions: badge, sound, full screen and tablet preview, collapsible header) on an Apple Silicon (arm64) MacBook running macOS, Go 1.27.1, Docker Desktop (linux/arm64 VM). Numbers are indicative, not guarantees. Re-run the commands below on your own hardware.

## Results

| Metric | Target | Measured |
| --- | --- | --- |
| Binary size, UI embedded, `-s -w -trimpath` | < 15 MB | linux/amd64 7.9 MB · linux/arm64 7.4 MB · darwin/arm64 7.6 MB · windows/amd64 8.2 MB |
| Docker image | < 15 MB compressed | 3.2 MB gzip · 11.1 MB as reported by `docker images` |
| Web UI bundle (embedded) | — | 78 KB gzip JavaScript |
| Idle memory | < 20 MB | ~5 MiB in the container (`docker stats`) · 11.8 MB RSS natively on macOS |
| Startup | < 100 ms | exec → first successful `/api/v1/health`: median 5.2 ms, max 44 ms (n=10; the max is the first, cold run). The banner's "Ready in" (time inside `main`) reports 0–4 ms |
| `docker stop` (SIGTERM → exit 0) | graceful | 0.26 s |

## Methodology

**Binary size.** Built as in the Dockerfile and release workflow:

```bash
npm run build -w @mailpeek/web
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o mailpeek ./cmd/mailpeek
ls -l mailpeek
```

**Docker image.** `docker build -t mailpeek .`, then `docker images mailpeek` (uncompressed, includes attestation metadata) and `docker save mailpeek | gzip -9 | wc -c` (what a registry transfers, roughly).

**Idle memory.** Start the container, send one email, wait a few seconds, then run `docker stats --no-stream`. For the native figure: `ps -o rss= -p <pid>` two seconds after startup.

**Startup.** A Python script starts the binary with `subprocess.Popen` and polls `GET /api/v1/health` every millisecond, measuring from just before `Popen` until the first 200. Repeated 10 times; median and max reported.

## Go benchmarks

`make bench` (`go test -run '^$' -bench . -benchmem ./internal/...`):

| Benchmark | Result |
| --- | --- |
| SMTP receive (one 3.6 KB message over a pipelined loopback connection) | ~35 µs/op, ~103 MB/s, 29 allocs |
| MIME parse (multipart/mixed with text, HTML with 100 links and a 13 KB base64 PDF, ~24 KB total) | ~84 µs/op, ~281 MB/s |
| Link extraction (HTML with 200 anchors) | ~65 µs/op |
| MemoryStore Save (store at capacity, evicting) | ~150 ns/op |
| MemoryStore List (100 messages, no filter) | ~0.9 µs/op |
| Filter (100 messages, `to` + `subject`) | ~4.5 µs/op |
| Body search (100 messages of ~200 KB text + HTML, `q` with no match) | ~18 ms/op, 0 allocs |
| Event dispatch (10 subscribers) | ~0.7 µs/op, 0 allocs |

Nothing has been optimized yet. These are baselines to compare future changes against.
