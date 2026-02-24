FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /mordecai ./cmd/mordecai

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /mordecai /usr/local/bin/mordecai

ENTRYPOINT ["mordecai"]
CMD ["--config", "/config/config.yaml"]
