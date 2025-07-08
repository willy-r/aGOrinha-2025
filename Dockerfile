# Stage 1: Build
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gorinha-be ./cmd/server

# Stage 2: Final image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gorinha-be .

EXPOSE 9999

ENTRYPOINT ["./gorinha-be"]
