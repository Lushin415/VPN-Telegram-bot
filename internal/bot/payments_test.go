package bot

import (
	"VPN-Telegram-bot/internal/db"
	"testing"
)

func TestReserveRenewPaymentAndCreateYooKassa(t *testing.T) {
	// Тестируем бизнес-логику формирования платежа на реальных функциях
	user := db.User{ID: 1, TelegramID: "12345"}
	server := db.Server{ID: 2, Price1: 100, Price3: 250, Price6: 400, Price12: 700}
	pay := db.Payment{UserID: user.ID, Amount: 250, Status: "pending", KeyID: nil, Months: func() *int { v := 3; return &v }()}

	// Вызов реальной функции (ожидается, что все зависимости замоканы или тестовая БД инициализирована)
	paymentID, url, err := ReserveRenewPaymentAndCreateYooKassa(&pay, server, user)
	if err != nil {
		t.Errorf("Ошибка создания платежа: %v", err)
	}
	if paymentID == "" || url == "" {
		t.Errorf("Пустой paymentID или url: %v, %v", paymentID, url)
	}
}
