package bot

import (
	"VPN-Telegram-bot/config"
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/services"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// ReserveVLESSKeyAndCreatePayment резервирует ключ, рассчитывает цену и создаёт платёж в YooKassa.
func ReserveVLESSKeyAndCreatePayment(userID uint, serverID uint, months int) (paymentURL string, err error) {
	var vlessKey db.VLESSKey
	// Транзакция для атомарности операций
	err = db.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Найти неиспользованный и не зарезервированный ключ для сервера
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("server_id = ? AND is_used = false AND (reserved_until IS NULL OR reserved_until < ?)", serverID, time.Now().Unix()).First(&vlessKey).Error
		if err != nil {
			return errors.New("Нет свободных ключей для выбранного сервера")
		}
		// 2. Зарезервировать ключ на 5 минут
		reservedUntil := time.Now().Add(5 * time.Minute).Unix()
		err = tx.Model(&vlessKey).Updates(map[string]interface{}{
			"is_used":        true,
			"reserved_until": reservedUntil,
			"user_id":        userID,
			"assigned_at":    time.Now().Unix(),
		}).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	// 3. Рассчитать цену
	var server db.Server
	err = db.DB.First(&server, serverID).Error
	if err != nil {
		return "", errors.New("Сервер не найден")
	}
	var price int
	switch months {
	case 1:
		price = server.Price1
	case 3:
		price = server.Price3
	case 6:
		price = server.Price6
	case 12:
		price = server.Price12
	default:
		return "", errors.New("Некорректный срок подписки")
	}
	// 4. Учесть скидку пользователя (если есть)
	var user db.User
	err = db.DB.Where("id = ?", userID).First(&user).Error
	if err == nil && user.CurrentDiscount > 0 {
		price = price * (100 - user.CurrentDiscount) / 100
	}
	// 5. Создать платёж в YooKassa
	paymentID, url, err := services.CreateYooKassaPayment(userID, price, config.AppCfg.YooKassaShopID, config.AppCfg.YooKassaSecret)
	if err != nil {
		return "", err
	}
	// 6. Сохранить платёж в базе
	pay := db.Payment{
		UserID:     userID,
		YooKassaID: paymentID,
		Amount:     price,
		Status:     "pending",
	}
	db.DB.Create(&pay)
	return url, nil
}

// ReserveRenewPaymentAndCreateYooKassa создаёт платёж для продления ключа и возвращает ссылку на оплату
func ReserveRenewPaymentAndCreateYooKassa(pay *db.Payment, server db.Server, user db.User) (paymentID, paymentURL string, err error) {
	// Создать платёж в YooKassa
	paymentID, url, err := services.CreateYooKassaPayment(user.ID, pay.Amount, config.AppCfg.YooKassaShopID, config.AppCfg.YooKassaSecret)
	if err != nil {
		return "", "", err
	}
	return paymentID, url, nil
}
