FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /bin/trip-planner ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
WORKDIR /app
COPY --from=builder /bin/trip-planner /app/trip-planner
COPY migrations /app/migrations
COPY web /app/web
RUN mkdir -p /app/data
EXPOSE 8080
ENV APP_ADDR=:8080
ENV SQLITE_PATH=/app/data/trips.db
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
CMD ["/app/trip-planner"]
