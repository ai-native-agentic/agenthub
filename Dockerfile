FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o agenthub-server ./cmd/agenthub-server

FROM alpine:3.20
RUN apk add --no-cache git
WORKDIR /app
COPY --from=builder /app/agenthub-server .
EXPOSE 8080
CMD ["./agenthub-server"]
