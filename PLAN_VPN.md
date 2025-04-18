ПЛАН РАЗРАБОТКИ TELEGRAM-БОТА ДЛЯ УПРАВЛЕНИЯ ПОДПИСКАМИ VLESS

    ПОДГОТОВКА ОКРУЖЕНИЯ

        Создать файл .env с переменными:

    BOT_TOKEN=…
    ADMIN_TELEGRAM_ID=…
    YOOKASSA_SHOP_ID=…
    YOOKASSA_SECRET_KEY=…
    DATABASE_URL=postgres://user:pass@localhost:5432/dbname?sslmode=disable

    Убедиться, что все обязательные переменные заполнены ( ВАЖНО!).
    
    Убедись, что файл .env добавлен в .gitignore и не попадает в репозиторий.

УСТАНОВКА И ЗАПУСК POSTGRESQL

    ЛОКАЛЬНО или через Docker Compose создать пустую базу (например, vpn_bot_db).

    В .env прописать строку подключения к ней.

    GORM AutoMigrate создаст в ней таблицы при первом запуске.

ОПРЕДЕЛЕНИЕ МОДЕЛЕЙ БАЗЫ ДАННЫХ

    Таблицы: users, servers, vless_keys, payments

    Модели (internal/db/models.go) с полями:

        users: TelegramID, CurrentDiscount

        servers: Name, IP, Price1, Price3, Price6, Price12, IsActive

        vless_keys: ServerID, Key (vless://…), IsUsed, ReservedUntil, UserID, AssignedAt

        payments: UserID, YooKassaID, Amount, Status

ИНИЦИАЛИЗАЦИЯ БАЗЫ ДАННЫХ

    Файл internal/db/db.go:

    db := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), …)
    db.AutoMigrate(&User{}, &Server{}, &VLESSKey{}, &Payment{})

    ПРОВЕРИТЬ, что DATABASE_URL задана!
    
    Для продакшена использовать отдельные миграции (goose) для контроля изменений схемы данных.

ЗАГРУЗКА КОНФИГУРАЦИИ

    Файл config/config.go: чтение .env, валидация всех переменных, заполнение config.AppConfig.
    
    При отсутствии критичных переменных окружения бот должен завершать работу с ошибкой.

ТОЧКА ВХОДА**

    cmd/bot/main.go:

    config.LoadConfig()
    db.InitDB()
    go bot.StartWebhook()          // веб‑хуки YooKassa
    go bot.InitCronJobs()          // фоновые задачи
    bot.StartBot()                 // Telegram-бот

ИНИЦИАЛИЗАЦИЯ TELEGRAM-БОТА

    internal/bot/bot.go:

        Создание tgbotapi.NewBotAPI(config.AppConfig.BotToken)

        Получение обновлений и передача их в handlers.HandleUpdate

ОБРАБОТЧИКИ КОМАНД ДЛЯ ПОЛЬЗОВАТЕЛЕЙ

    internal/bot/handlers.go (package bot):

        /start, /support, /buy, /renew, /subscriptions

        Inline‑кнопки для выбора серверов и тарифов

        Функция HandleUpdate

ЛОГИКА РЕЗЕРВИРОВАНИЯ И ВЫДАЧИ КЛЮЧЕЙ

    internal/bot/payments.go:

        Поиск ключа: is_used = false AND (reserved_until IS NULL OR < NOW())

        ОБЯЗАТЕЛЬНО после поиска:

        db.Model(&VLESSKey{}).Where("id = ?", key.ID).Updates(map[string]interface{}{
          "is_used":        true,
          "reserved_until": time.Now().Add(5*time.Minute),
        })

        Расчёт цены с учётом скидок (5%/10%/15%)

        Вызов services.CreateYooKassaPayment
    
        Резервирование и выдача ключей должны выполняться в рамках одной транзакции для предотвращения гонок при одновременных покупках.

ИНТЕГРАЦИЯ С YOOKASSA

    internal/services/yookassa.go:

        CreateYooKassaPayment(userID, amount) (paymentID, paymentURL, error)
    
        Обработка webhook должна быть идемпотентной — повторные уведомления не должны приводить к ошибкам или двойной выдаче доступа.
        Рекомендуется проверять подписи или использовать секреты для валидации подлинности запросов от YooKassa.

ОБРАБОТКА ВЕБ‑ХУКОВ YOOKASSA

    internal/bot/webhook.go:

        handleYooKassaWebhook читает id/status

        Обновляет payments.status

        При succeeded → activateVLESSKey (оставить is_used=true, сбросить reserved_until)

        При другом статусе → releaseReservedKey (is_used=false, reserved_until=NULL)

ФОНОВАЯ ПРОВЕРКА ПЛАТЕЖЕЙ

    internal/services/payment_checker.go:

        Каждые 2 минуты (cron) проверять status = pending AND created_at < NOW() - 3min

        GET /v3/payments/{id} → обновить статус

НАПОМИНАНИЯ О ПОДПИСКЕ
фывыв
internal/services/subscription_notifier.go:

        Ежедневно в 10:00 (cron)

        Находит VLESSKey с is_used = true и AssignedAt + duration

        Отправляет за 7 и 3 дня до окончания

АДМИН‑КОМАНДЫ

    internal/handlers/admin.go (package handlers):

        /listservers – список серверов + подсчёт свободных/всех ключей

        /broadcast <msg> – рассылка всем пользователям

        Проверка chatID == config.AppConfig.AdminTelegramID

КЭШИРОВАНИЕ И МОНТОРИНГ (опционально)

    pkg/cache/cache.go – sync.Map для списка серверов

    internal/services/monitoring.go – ежедневный ping, деактивация is_active=false

РЕЗЕРВНОЕ КОПИРОВАНИЕ БАЗЫ

    external/cron_backup.sh – pg_dump + удаление старых бэкапов

    Запуск через crontab
    
    Для восстановления базы из бэкапа используйте команду:
    pg_restore -U user -d dbname /path/to/backup.dump

## ЛОГИРОВАНИЕ И МОНИТОРИНГ

Рекомендуется реализовать логирование ошибок и ключевых событий, а также мониторинг состояния бота и платежей для своевременного обнаружения и устранения проблем.

СТРУКТУРА ПРОЕКТА

/vpn-bot
├── cmd/bot/main.go
├── config/config.go
├── internal
│   ├── bot
│   │   ├── bot.go
│   │   ├── handlers.go
│   │   ├── payments.go
│   │   ├── webhook.go
│   │   └── scheduler.go
│   ├── handlers
│   │   └── admin.go
│   ├── db
│   │   ├── models.go
│   │   └── db.go
│   └── services
│       ├── yookassa.go
│       ├── payment_checker.go
│       └── subscription_notifier.go
├── pkg
│   ├── cache/cache.go
│   └── logger/logger.go
├── external/cron_backup.sh
├── .env
├── docker-compose.yml
├── go.mod
└── go.sum

    ВСЕ ВАЖНЫЕ МЕСТА (резервирование ключей, обновление статусов, .env) ОТМЕЧЕНЫ В ПЛАНЕ И КОДЕ.

ПРИМЕЧАНИЕ (КРАТКИЕ ИНСТРУКЦИИ ДЛЯ НЕЙРОСЕТИ)

    .env загружается первым: все секреты и URL БД читаются из config.LoadConfig().

    Каждый каталог — один пакет: файлы в internal/bot используют package bot; в internal/handlers — package handlers; в internal/db — package db; в internal/services — package services.

    InitDB → AutoMigrate: GORM автоматически создаёт таблицы при запуске.

    StartWebhook() и InitCronJobs() запускаются до StartBot(): веб‑хуки и cron‑задачи должны работать параллельно с ботом.

    HandleUpdate(bot, update) в internal/bot/handlers.go обрабатывает все пользовательские команды и callback.

    reserveKeyAndCreatePayment: после поиска ключа обнови его поля

    Updates(map[string]interface{}{
      "is_used":        true,
      "reserved_until": time.Now().Add(5*time.Minute),
    })

    Webhook‑handler в webhook.go читает id и status, обновляет payments.status, вызывает activateVLESSKey или releaseReservedKey.

    Cron‑задачи (scheduler.go):

        Каждые 2 мин → services.CheckPendingPayments()

        Каждый день в 10:00 → services.SendSubscriptionReminders()

    Админ‑команды (handlers/admin.go): проверка chatID == AdminTelegramID, команды /listservers и /broadcast.

    Резервное копирование: скрипт external/cron_backup.sh запускается из crontab.

    ВСЕ ключевые шаги и проверки обозначены.