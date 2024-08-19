FROM golang:1.22.6-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -mod=vendor -x -v

FROM alpine:latest

COPY --from=builder /app/kafka-change-retantion /usr/local/bin/kafka-change-retantion

RUN chmod +x /usr/local/bin/kafka-change-retantion && \
    addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -D appuser && \
    chown appuser:appgroup /usr/local/bin/kafka-change-retantion

USER appuser

CMD ["kafka-change-retantion"]
