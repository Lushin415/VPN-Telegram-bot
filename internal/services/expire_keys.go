package services

import (
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/logger"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"time"
)

// DisableExpiredKeys отключает ключи с истёкшей подпиской и уведомляет пользователя
func DisableExpiredKeys(bot *tgbotapi.BotAPI) {
	now := time.Now().Unix()
	var keys []db.VLESSKey
	db.DB.Where("is_used = true AND reserved_until IS NOT NULL AND reserved_until < ?", now).Find(&keys)
	for _, key := range keys {
		if key.UserID == nil {
			continue
		}
		var user db.User
		if err := db.DB.First(&user, *key.UserID).Error; err != nil {
			logger.NotifyAdmin(fmt.Sprintf("Не удалось найти пользователя для отключения ключа: keyID=%d", key.ID))
			continue
		}
		msg := tgbotapi.NewMessage(parseInt64(user.TelegramID), "Ваша подписка завершена, для продления воспользуйтесь ботом")
		_, _ = bot.Send(msg)
		db.DB.Model(&db.VLESSKey{}).Where("id = ?", key.ID).Updates(map[string]interface{}{"is_used": false, "user_id": nil, "assigned_at": nil, "reserved_until": nil, "notified_expiring": false})
	}
}

// RenewKeyAfterPayment активирует ранее отключённый ключ после оплаты
func RenewKeyAfterPayment(userID uint, keyID uint, months int) error {
	now := time.Now().Unix()
	reservedUntil := now + int64(months*30*24*60*60)
	return db.DB.Model(&db.VLESSKey{}).Where("id = ? AND user_id IS NULL", keyID).Updates(map[string]interface{}{
		"is_used":        true,
		"user_id":        userID,
		"assigned_at":    now,
		"reserved_until": reservedUntil,
	}).Error
}

func parseInt64(s string) int64 {
	var id int64
	_, _ = fmt.Sscan(s, &id)
	return id
}
