package bot

import (
	"VPN-Telegram-bot/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

func StartBot() {
	bot, err := tgbotapi.NewBotAPI(config.AppCfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		HandleUpdate(bot, update)
	}
}

// StartBotWithInstance запускает Telegram-бота с переданным экземпляром
func StartBotWithInstance(bot *tgbotapi.BotAPI) {
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		HandleUpdate(bot, update)
	}
}
