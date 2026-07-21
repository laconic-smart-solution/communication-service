FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /out/telegram-service ./app/telegram-service/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/telegram-service /app/telegram-service
EXPOSE 8087
ENTRYPOINT ["/app/telegram-service"]
