FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/platform cmd/server/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder /build/platform .
COPY --from=builder /build/api/ ./api/

USER appuser

EXPOSE 8080

CMD ["./platform"]
