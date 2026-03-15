FROM golang:1.24-bullseye AS builder
WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN go build -o main ./cmd/server

FROM debian:bullseye-slim
RUN apt-get update && apt-get install -y \
    gcc g++ python3 default-jdk curl unzip \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*
RUN curl -sSL https://github.com/JetBrains/kotlin/releases/download/v1.9.0/kotlin-compiler-1.9.0.zip -o kotlin.zip \
    && unzip kotlin.zip -d /opt \
    && rm kotlin.zip \
    && ln -s /opt/kotlinc/bin/kotlinc /usr/local/bin/kotlinc \
    && ln -s /opt/kotlinc/bin/kotlin /usr/local/bin/kotlin
COPY --from=builder /app/main /app/main
EXPOSE 8080
CMD ["/app/main"]