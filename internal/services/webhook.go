package services

import (
	"VPN-Telegram-bot/config"
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/logger"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
	"net/http"
	"strings"
)

// Проверка HMAC подписи webhook YooKassa (Authorization или Content-Yoomoney-Signature)
func checkYooKassaSignature(secret string, body []byte, authHeader, yoomoneyHeader string) bool {
	var signatures []string
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "HMAC ") || strings.HasPrefix(authHeader, "HMAC-SHA256 ") {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				signatures = append(signatures, parts[1])
			}
		}
	}
	if yoomoneyHeader != "" {
		signatures = append(signatures, yoomoneyHeader)
	}
	if len(signatures) == 0 {
		return false
	}
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	calc := hex.EncodeToString(h.Sum(nil))
	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(calc)) {
			return true
		}
	}
	return false
}

// WebhookHandler обрабатывает уведомления от YooKassa
func WebhookHandler(bot *tgbotapi.BotAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer logger.NotifyOnPanic("WebhookHandler")
		if r.Method != http.MethodPost {
			logger.NotifyAdmin("Webhook вызван с неверным методом: " + r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.NotifyAdmin("Ошибка чтения тела webhook: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		r.Body.Close()
		authHeader := r.Header.Get("Authorization")
		yoomoneyHeader := r.Header.Get("Content-Yoomoney-Signature")
		if !checkYooKassaSignature(config.AppCfg.YooKassaSecret, body, authHeader, yoomoneyHeader) {
			logger.NotifyAdmin("Недействительная подпись webhook")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid signature"))
			return
		}
		var event struct {
			Object struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"object"`
		}
		err = json.Unmarshal(body, &event)
		if err != nil {
			logger.NotifyAdmin("Ошибка парсинга webhook: " + err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if event.Object.Status != "succeeded" {
			logger.NotifyAdmin("Платеж не прошёл: статус = " + event.Object.Status)
			w.WriteHeader(http.StatusOK)
			return // Обрабатываем только успешные платежи
		}
		// --- Продление старого ключа, если есть ---
		var pay db.Payment
		err = db.DB.Where("yoo_kassa_id = ?", event.Object.ID).First(&pay).Error
		if err != nil {
			logger.NotifyAdmin("Ошибка поиска платежа по YooKassaID: " + err.Error())
			w.WriteHeader(http.StatusOK)
			return
		}
		if pay.KeyID != nil && pay.Months != nil {
			err := RenewKeyAfterPayment(pay.UserID, *pay.KeyID, *pay.Months)
			if err != nil {
				logger.NotifyAdmin("Ошибка продления ключа после оплаты: " + err.Error())
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		// Обновить статус платежа
		db.DB.Model(&pay).Update("status", "succeeded")

		// Обычная покупка — найти пользователя и ключ, выдать ключ
		var user db.User
		db.DB.First(&user, pay.UserID)
		var key db.VLESSKey
		db.DB.Where("user_id = ?", user.ID).Order("assigned_at desc").First(&key)
		if key.Key != "" {
			msg := tgbotapi.NewMessage(parseInt64(user.TelegramID), "Ваш VPN-ключ: "+key.Key)
			bot.Send(msg)
		}
		w.WriteHeader(http.StatusOK)
	}
}
