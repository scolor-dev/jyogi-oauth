FROM golang:1-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o auth-server ./cmd/server

FROM alpine:3.20
COPY --from=builder /app/auth-server /usr/local/bin/
EXPOSE 8080
CMD ["auth-server"]
