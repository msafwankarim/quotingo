FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download || true
RUN CGO_ENABLED=0 GOOS=linux go build -o /quotingo ./cmd/main.go

FROM alpine:3.18
RUN addgroup -S app && adduser -S app -G app
COPY --from=builder /quotingo /usr/local/bin/quotingo
COPY --from=builder /app/templates /app/templates
WORKDIR /app
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/quotingo"]
