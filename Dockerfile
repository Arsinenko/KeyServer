# --- Stage 1: build ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Кэш зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем код
COPY . .

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server

# --- Stage 2: runtime ---
FROM alpine:3.20

WORKDIR /app

# Добавим CA сертификаты
RUN apk add --no-cache ca-certificates

# Копируем бинарник
COPY --from=builder /app/server .

# Копируем TLS сертификаты
COPY cert.pem key.pem ./

# Открываем порт
EXPOSE 8443

# Запуск
CMD ["./server"]
    