FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Копировать всё сразу (включая vendor)
COPY . .

# Билд с vendor (без go mod download)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /build/platform cmd/server/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /build/platform .
COPY --from=builder /build/api/ ./api/

USER appuser

EXPOSE 8080

CMD ["./platform"]