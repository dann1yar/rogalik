# syntax=docker/dockerfile:1

# ---- Stage 1: build ---------------------------------------------------
# Only cmd/server is built here. cmd/game (the Ebiten client) needs cgo +
# platform GUI/graphics headers and is intentionally built natively per-OS
# by the GitHub Actions matrix instead (see .github/workflows/release.yml).
FROM golang:1.23-bookworm AS builder

WORKDIR /src

# Network in this environment can't reach proxy.golang.org, so route
# golang.org/x/* and google.golang.org/protobuf through their GitHub
# mirrors via the replace directives already committed in go.mod.
ENV GOPROXY=direct
ENV GOSUMDB=off
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/pixelkeep-server ./cmd/server

# ---- Stage 2: minimal runtime ------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /out/pixelkeep-server /pixelkeep-server

ENV PIXELKEEP_ADDR=:8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/pixelkeep-server"]
