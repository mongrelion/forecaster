# ---- Build ----
FROM golang:1.26-alpine AS builder

ARG PORT=8080

WORKDIR /app
COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN go mod download && \
    go build -ldflags="-s -w" -o /forecaster ./cmd/server/

# ---- Runtime (alpine) ----
FROM alpine:3

ARG PORT=8080

ENV PORT=$PORT HOST=0.0.0.0 PUBLIC_DIR=/app/public

WORKDIR /app

COPY --from=builder /app/public /app/public
COPY --from=builder /forecaster /app/forecaster

# EXPOSE is metadata-only and does not interpolate variables.
# The actual port is controlled by the PORT env var (default 8080).
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:${PORT}/healthz || exit 1

CMD ["./forecaster"]
