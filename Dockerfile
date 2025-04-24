# syntax=docker/dockerfile:1
FROM golang:1.24.2-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/bot ./bot
COPY ./config ./config
COPY ./backups ./backups
COPY ./USER_GUIDE.md ./USER_GUIDE.md
COPY ./ADMIN_GUIDE.md ./ADMIN_GUIDE.md
COPY .env.example ./.env.example
RUN apk add --no-cache bash
RUN apk add --no-cache openssh
ENV TZ=Asia/Shanghai
EXPOSE 8080
CMD ["./bot"]
