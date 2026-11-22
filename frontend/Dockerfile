
# Этап 1. Сборка приложения
FROM node:22-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей
COPY package*.json ./

# Устанавливаем зависимости
RUN npm ci

# Копируем исходный код
COPY . .

# Собираем production-версию
RUN npm run build


# Этап 2. Запуск веб-сервера
FROM caddy:2-alpine

# Копируем собранный Frontend
COPY --from=builder /app/dist /usr/share/caddy

# Копируем конфигурацию Caddy
COPY Caddyfile /etc/caddy/Caddyfile