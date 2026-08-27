# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependency downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/server ./cmd/server

FROM alpine:3.20

# ca-certificates: needed for outbound HTTPS calls (Google OAuth, Midtrans, R2).
# tzdata: event/order timestamps are handled with real locations, not just UTC.
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/bin/server ./server

USER app

EXPOSE 8000

ENTRYPOINT ["./server"]
