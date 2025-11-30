# Первая стадия (builder)
FROM golang:1.24.0-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/app

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

#RUN mkdir -p internal/web/templates internal/web/static config

COPY --from=builder /app/main .
COPY --from=builder /app/internal/web/templates ./internal/web/templates
COPY --from=builder /app/internal/web/static ./internal/web/static
COPY --from=builder /app/config ./config



EXPOSE 8080

CMD ["./main"]



