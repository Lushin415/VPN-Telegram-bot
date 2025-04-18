# VPN Telegram Bot

Production-ready Telegram-бот для продажи VPN-доступа с автоматизацией подписок, оплат, администрирования, резервного копирования и мониторинга.

---

## Быстрый старт (Docker)

1. Скопируйте `.env.example` в `.env` и заполните переменные:
   - `BOT_TOKEN` — токен Telegram-бота
   - `ADMIN_TELEGRAM_ID` — Telegram ID администратора
   - `YOOKASSA_SHOP_ID` — Shop ID YooKassa
   - `YOOKASSA_SECRET_KEY` — секретный ключ YooKassa
   - `DATABASE_URL` — строка подключения к PostgreSQL (пример для docker-compose: `postgres://vpnuser:vpnpassword@db:5432/vpn?sslmode=disable`)

2. Запустите проект:
   ```sh
   docker-compose up --build
   ```

3. Для остановки:
   ```sh
   docker-compose down
   ```

---

## Структура проекта
- `cmd/bot/main.go` — точка входа
- `internal/` — бизнес-логика, сервисы, админ-функции
- `config/` — конфиги
- `backups/` — резервные копии базы
- `USER_GUIDE.md` — инструкция для пользователя
- `ADMIN_GUIDE.md` — инструкция для администратора

---

## Полезные команды
- `/start`, `/buy`, `/subscriptions`, `/getkey`, `/support` — для пользователя
- `/admin_stats`, `/admin_addserver`, `/admin_backup` и др. — для администратора

---

## Документация
- [USER_GUIDE.md](./USER_GUIDE.md) — инструкция пользователя
- [ADMIN_GUIDE.md](./ADMIN_GUIDE.md) — инструкция администратора

---

## Примечания
- Все переменные должны быть заданы через `.env` или переменные окружения.
- Для production рекомендуется использовать Docker.
- Бэкапы и логи хранятся на хосте.

---

Если нужна помощь — см. инструкции или пиши в поддержку!

