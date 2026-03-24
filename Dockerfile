FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /bin/trip-planner ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /bin/trip-planner /app/trip-planner
COPY migrations /app/migrations
COPY web /app/web
RUN mkdir -p /app/data
EXPOSE 8080
ENV APP_ADDR=:8080
ENV SQLITE_PATH=/app/data/trips.db
CMD ["/app/trip-planner"]
