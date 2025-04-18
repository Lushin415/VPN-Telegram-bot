package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type AppConfig struct {
	BotToken        string
	AdminTelegramID string
	YooKassaShopID  string
	YooKassaSecret  string
	DatabaseURL     string
}

var AppCfg AppConfig

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found, relying on environment variables")
	}

	AppCfg.BotToken = os.Getenv("BOT_TOKEN")
	AppCfg.AdminTelegramID = os.Getenv("ADMIN_TELEGRAM_ID")
	AppCfg.YooKassaShopID = os.Getenv("YOOKASSA_SHOP_ID")
	AppCfg.YooKassaSecret = os.Getenv("YOOKASSA_SECRET_KEY")
	AppCfg.DatabaseURL = os.Getenv("DATABASE_URL")

	if AppCfg.BotToken == "" || AppCfg.AdminTelegramID == "" || AppCfg.YooKassaShopID == "" || AppCfg.YooKassaSecret == "" || AppCfg.DatabaseURL == "" {
		log.Fatal("Critical environment variables are missing. Bot will exit.")
	}
}
