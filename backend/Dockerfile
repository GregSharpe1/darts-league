FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/darts-league-api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache tzdata && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/darts-league-api /usr/local/bin/darts-league-api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/darts-league-api"]
