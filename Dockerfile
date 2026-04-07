FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -tags netgo -ldflags '-s -w' -o executor ./cmd/executor-server

FROM eclipse-temurin:17-jdk-alpine

WORKDIR /app

COPY --from=builder /app/executor .

COPY internal/executor/worker ./internal/executor/worker/

EXPOSE 8081

CMD ["./executor"]