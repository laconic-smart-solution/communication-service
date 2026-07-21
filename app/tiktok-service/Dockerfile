FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /out/tiktok-service ./app/tiktok-service/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/tiktok-service /app/tiktok-service
EXPOSE 8086
ENTRYPOINT ["/app/tiktok-service"]
