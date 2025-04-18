package main

import (
	"VPN-Telegram-bot/config"
	"VPN-Telegram-bot/internal/admin"
	"VPN-Telegram-bot/internal/bot"
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/inter
al/logger"
	"VPN-Telegram-bot/inter
	"VPN-Telegram-bot/internal/logger"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"os"
)

func main() {
	config.LoadConfig()
	db.InitDB()
	db.StartExpiredKeyCleaner() // запуск фоновой очистки резервов
	botapi, err := tgbotapi.NewBotAPI(config.AppCfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	logger.InitNotifier(botapi, admin.AdminTelegramID)
	// Автоматическое обновление статуса серверов
	c := cron.New()
	c.AddFunc("@every 1m", services.UpdateAllServerStatuses)
	// Автоматический бэкап БД раз в сутки
	c.AddFunc("0 3 * * *", func() {
		dsn := os.Getenv("DATABASE_URL")
		admin.AutoBackupDatabase(botapi, admin.AdminTelegramID, dsn)
	})
	// Уведомления о скором окончании подписки (раз в сутки в 10:00)
	c.AddFunc("0 10 * * *", func() {
		services.NotifyExpiringSubscriptions(botapi, 3)
	})
	// Отключение просроченных ключей и уведомление пользователей (каждый день в 03:30)
	c.AddFunc("30 3 * * *", func() {
		services.DisableExpiredKeys(botapi)
	})
	c.Start()
	// Запуск webhook-сервера для YooKassa (VPS)
	go func() {
		http.HandleFunc("/yookassa/webhook", services.WebhookHandler(botapi))
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		log.Println("Запуск webhook-сервера на :8080 (VPS)")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Webhook server error: %v", err)
		}
	}()
	// Запуск Telegram-бота (VPS)
	bot.StartBotWithInstance(botapi)
}
