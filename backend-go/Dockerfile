FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o seller-trust-api ./cmd/api

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/seller-trust-api ./seller-trust-api

EXPOSE 8080

CMD ["./seller-trust-api"]
