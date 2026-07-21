FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /out/shopee-service ./app/shopee-service/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/shopee-service /app/shopee-service
EXPOSE 8085
ENTRYPOINT ["/app/shopee-service"]
