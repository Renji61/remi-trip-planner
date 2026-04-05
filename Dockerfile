FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /bin/trip-planner ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
RUN addgroup -S remi && adduser -S -G remi -h /app remi
WORKDIR /app
COPY --from=builder /bin/trip-planner /app/trip-planner
COPY migrations /app/migrations
COPY web /app/web
COPY CHANGELOG.md /app/CHANGELOG.md
RUN mkdir -p /app/data /app/web/static/uploads && chown -R remi:remi /app
USER remi
EXPOSE 8080
ENV APP_ADDR=:8080
ENV SQLITE_PATH=/app/data/trips.db
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
CMD ["/app/trip-planner"]
