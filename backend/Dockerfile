FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /analytics-server ./cmd/server

FROM alpine:3.19

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /analytics-server /app/analytics-server

ENV PORT=8080 \
    DB_PATH=/data/analytics.db \
    SYNC_INTERVAL_MINUTES=60 \
    CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000

VOLUME ["/data"]

EXPOSE 8080

CMD ["/app/analytics-server"]
