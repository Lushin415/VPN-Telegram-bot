package services

import (
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/logger"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"time"
)

// NotifyExpiringSubscriptions отправляет уведомления пользователям о скором окончании подписки
func NotifyExpiringSubscriptions(bot *tgbotapi.BotAPI, daysBefore int) {
	var keys []db.VLESSKey
	now := time.Now().Unix()
	soon := now + int64(daysBefore*24*60*60)
	db.DB.Where("is_used = true AND reserved_until IS NOT NULL AND reserved_until <= ? AND reserved_until > ? AND notified_expiring = false", soon, now).Find(&keys)
	for _, key := range keys {
		if key.UserID == nil {
			continue
		}
		var user db.User
		if err := db.DB.First(&user, *key.UserID).Error; err != nil {
			logger.NotifyAdmin(fmt.Sprintf("Не удалось найти пользователя для уведомления о скором окончании: keyID=%d", key.ID))
			continue
		}
		msg := tgbotapi.NewMessage(parseInt64(user.TelegramID), "Ваша подписка истекает через 3 дня. Продлить: /subscriptions")
		if _, err := bot.Send(msg); err != nil {
			logger.NotifyAdmin(fmt.Sprintf("Ошибка отправки уведомления пользователю %s: %v", user.TelegramID, err))
			continue
		}
		db.DB.Model(&db.VLESSKey{}).Where("id = ?", key.ID).Update("notified_expiring", true)
	}
}
