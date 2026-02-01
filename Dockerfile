FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /aptos-guardian ./cmd/aptos-guardian

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /aptos-guardian /app/aptos-guardian
COPY --from=builder /app/configs/example.yaml /app/configs/example.yaml
COPY --from=builder /app/web /app/web
EXPOSE 8080
ENTRYPOINT ["/app/aptos-guardian"]
CMD ["-config", "/app/configs/example.yaml"]
