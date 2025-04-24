package main

import (
	"VPN-Telegram-bot/config"
	"VPN-Telegram-bot/internal/admin"
	"VPN-Telegram-bot/internal/bot"
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/logger"
	"VPN-Telegram-bot/internal/services"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"io"
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
	// --- Логирование в файл и консоль ---
	logFile, err := os.OpenFile("bot.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Не удалось открыть файл логов: %v", err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
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
	// --- POLLING MODE: Telegram webhook server is DISABLED for testing ---
	// TODO: Enable for webhook production mode:
	// go func() {
	// 	http.HandleFunc("/telegram/webhook", services.TelegramWebhookHandler(botapi))
	// 	log.Println("Запуск Telegram webhook-сервера на :443 (VPS)")
	// 	if err := http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil); err != nil {
	// 		log.Fatalf("Telegram webhook server error: %v", err)
	// 	}
	// }()
	// -------------------------------------------------------------
	// Запуск Telegram-бота (polling)
	bot.StartBotWithInstance(botapi)
}
