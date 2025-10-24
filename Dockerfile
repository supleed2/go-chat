FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o server ./server

FROM alpine:latest
RUN addgroup -g 1000 -S appgroup && adduser -u 1000 -S appuser -G appgroup
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
USER appuser
HEALTHCHECK CMD wget --spider http://127.0.0.1:8080/health || exit 1
CMD ["./server"]
