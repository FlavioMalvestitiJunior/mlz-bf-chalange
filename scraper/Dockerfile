FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod tidy

COPY . .
RUN go build -o scraper main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/scraper .

CMD ["./scraper"]
