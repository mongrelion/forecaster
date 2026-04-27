# ---- Build ----
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

RUN go mod download && \
    go build -ldflags="-s -w" -o /forecaster ./cmd/server/

# ---- Runtime (scratch) ----
FROM scratch

WORKDIR /

COPY --from=builder /forecaster /forecaster

EXPOSE 8080

CMD ["/forecaster"]
