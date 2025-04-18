package bot

import (
	"VPN-Telegram-bot/internal/admin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetReplyKeyboard(userID int64) tgbotapi.ReplyKeyboardMarkup {
	if admin.IsAdmin(userID) {
		return tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("/admin_stats"),
				tgbotapi.NewKeyboardButton("/admin_keys"),
				tgbotapi.NewKeyboardButton("/admin_logs"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("/admin_servers"),
				tgbotapi.NewKeyboardButton("/admin_reload"),
				tgbotapi.NewKeyboardButton("/admin_payments"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("/admin_broadcast"),
				tgbotapi.NewKeyboardButton("/admin_backup"),
				tgbotapi.NewKeyboardButton("/admin_addserver"),
			),
		)
	}
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("/buy"),
			tgbotapi.NewKeyboardButton("/subscriptions"),
		),
	)
}
