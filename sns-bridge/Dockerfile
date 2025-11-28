FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod tidy

COPY . .
RUN go build -o sns-bridge main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/sns-bridge .

CMD ["./sns-bridge"]
