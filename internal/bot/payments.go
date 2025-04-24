package bot

import (
	"VPN-Telegram-bot/config"
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/services"
	"errors"
	"github.com/google/uuid"
	"strconv"
	"time"
)

// ReserveVLESSKeyAndCreatePayment резервирует ключ, рассчитывает цену и создаёт платёж в YooKassa.
func ReserveVLESSKeyAndCreatePayment(userID uint, serverID uint, months int) (paymentURL string, err error) {
	// 1. Генерируем новый uuid для ключа
	uuidKey := uuid.New().String()

	// 2. Проверяем, есть ли у пользователя активный ключ
	var activeKey db.VLESSKey
	assignedAt := time.Now().Unix()
	err = db.DB.Where("user_id = ? AND is_used = true AND (reserved_until IS NULL OR reserved_until > ?)", userID, assignedAt).First(&activeKey).Error
	if err == nil {
		// Есть активный ключ — продлеваем срок действия
		var extendSeconds int64
		switch months {
		case 1:
			extendSeconds = 30 * 24 * 60 * 60
		case 3:
			extendSeconds = 90 * 24 * 60 * 60
		case 6:
			extendSeconds = 180 * 24 * 60 * 60
		case 12:
			extendSeconds = 365 * 24 * 60 * 60
		}
		if activeKey.ReservedUntil != nil && *activeKey.ReservedUntil > assignedAt {
			*activeKey.ReservedUntil += extendSeconds
		} else {
			val := assignedAt + extendSeconds
			activeKey.ReservedUntil = &val
		}
		db.DB.Save(&activeKey)
		return "Подписка продлена!", nil
	}

	// Новый ключ: определяем порядковый номер N для email
	var count int64
	db.DB.Model(&db.VLESSKey{}).Where("user_id = ?", userID).Count(&count)
	N := count + 1
	telegramID := ""
	var user db.User
	err = db.DB.First(&user, userID).Error
	if err == nil {
		telegramID = user.TelegramID
	}
	email := telegramID + "_" + strconv.FormatInt(N, 10)

	// 3. Сохраняем ключ в БД
	reservedUntil := assignedAt + 5*60 // 5 минут резерв
	vlessKey := db.VLESSKey{
		ServerID:      serverID,
		Key:           uuidKey,
		IsUsed:        true,
		ReservedUntil: &reservedUntil,
		UserID:        &userID,
		AssignedAt:    &assignedAt,
	}
	err = db.DB.Create(&vlessKey).Error
	if err != nil {
		return "", errors.New("Ошибка создания ключа: " + err.Error())
	}

	// 4. Добавляем ключ на сервер через SSH
	sshErr := services.AddClientToRemoteXrayConfig(
		"root", "150.241.85.73", "59421", "/root/.ssh/id_ed25519",
		uuidKey, email, "xtls-rprx-vision",
	)
	if sshErr != nil {
		return "", errors.New("Ошибка добавления ключа на сервер: " + sshErr.Error())
	}

	// 5. Рассчитать цену
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
	// 6. Учесть скидку пользователя (если есть)
	err = db.DB.Where("id = ?", userID).First(&user).Error
	if err == nil && user.CurrentDiscount > 0 {
		price = price * (100 - user.CurrentDiscount) / 100
	}
	// 7. Создать платёж в YooKassa
	paymentID, url, err := services.CreateYooKassaPayment(userID, price, config.AppCfg.YooKassaShopID, config.AppCfg.YooKassaSecret)
	if err != nil {
		return "", err
	}
	// 8. Сохранить платёж в базе
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
