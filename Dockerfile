# ---- Build ----
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN go mod download && \
    go build -ldflags="-s -w" -o /forecaster ./cmd/server/

# ---- Runtime (alpine) ----
FROM alpine:3

WORKDIR /app

COPY --from=builder /forecaster /app/forecaster

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

CMD ["./forecaster"]
