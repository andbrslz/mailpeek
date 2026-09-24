# syntax=docker/dockerfile:1

# 1. Web UI (React + Vite)
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src
COPY package.json package-lock.json tsconfig.base.json ./
COPY web/package.json web/
COPY packages/client/package.json packages/client/
COPY packages/playwright/package.json packages/playwright/
COPY e2e/package.json e2e/
COPY examples/playwright/package.json examples/playwright/
RUN npm ci --workspace @mailpeek/web --include-workspace-root --no-audit --no-fund
COPY web/ web/
RUN npm run build -w @mailpeek/web

# 2. Go binary with the UI embedded (cross-compiled for the target platform)
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /src/web/dist web/dist
ARG VERSION=dev
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /mailpeek ./cmd/mailpeek

# 3. Runtime: just the binary
FROM scratch
COPY --from=build /mailpeek /mailpeek
USER 65534:65534
EXPOSE 1026 8026
HEALTHCHECK --interval=10s --timeout=3s --start-period=2s --retries=3 CMD ["/mailpeek", "healthcheck"]
ENTRYPOINT ["/mailpeek"]
