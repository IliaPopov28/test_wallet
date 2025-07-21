FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /wallet-app ./cmd/server

FROM alpine:latest
COPY --from=builder /wallet-app /wallet-app
COPY config.env /config.env
EXPOSE 8080
CMD ["/wallet-app"] 