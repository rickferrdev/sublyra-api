FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/bin/api ./cmd/api

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/api ./api

EXPOSE 8080

CMD ["./api"]
