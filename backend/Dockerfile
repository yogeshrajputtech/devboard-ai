FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o devboard-ai-backend .

FROM alpine:3.20

WORKDIR /app

# ca-certificates: the backend calls out to api.anthropic.com over HTTPS.
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 appuser

COPY --from=builder /app/devboard-ai-backend ./devboard-ai-backend
RUN chmod +x ./devboard-ai-backend

USER appuser

EXPOSE 8080

ENTRYPOINT ["./devboard-ai-backend"]
