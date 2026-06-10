FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /out/facebook-service ./app/facebook-service/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/facebook-service /app/facebook-service
EXPOSE 8081
ENTRYPOINT ["/app/facebook-service"]
