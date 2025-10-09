

FROM golang:1.24.0-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/app

FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/main .

RUN mkdir -p /app/storage/original /app/storage/processed /app/storage/metadata

RUN mkdir -p /app/internal/web/templates

EXPOSE 8080

CMD ["./main"]