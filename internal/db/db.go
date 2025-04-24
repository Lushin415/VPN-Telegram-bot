package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"

	"time"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	DB = db
	// Добавлено: миграция таблицы пользователей
	db.AutoMigrate(&User{}, &Server{}, &VLESSKey{}, &Payment{}, &Order{})
}

// CreateOrder сохраняет заказ в БД
func CreateOrder(userID uint, server, tariff, paymentID string) error {
	order := Order{
		UserID:    userID,
		Server:    server,
		Tariff:    tariff,
		PaymentID: paymentID,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}
	return DB.Create(&order).Error
}

// MarkOrderPaid меняет статус заказа на paid по paymentID
func MarkOrderPaid(paymentID string) error {
	return DB.Model(&Order{}).Where("payment_id = ?", paymentID).Update("status", "paid").Error
}

// StartExpiredKeyCleaner запускает фоновую задачу очистки просроченных резервов ключей
func StartExpiredKeyCleaner() {
	go func() {
		for {
			now := time.Now().Unix()
			// Ключ считается "зависшим", если is_used = false, reserved_until < now
			res := DB.Exec("UPDATE vless_keys SET user_id = NULL, reserved_until = NULL WHERE is_used = false AND reserved_until IS NOT NULL AND reserved_until < ?", now)
			if res.Error != nil {
				log.Println("Ошибка очистки просроченных резервов ключей:", res.Error)
			}
			time.Sleep(5 * time.Minute)
		}
	}()
}

// --- Админские методы для статистики, ключей, платежей, пользователей ---

func CountUsers() int {
	var count int64
	DB.Model(&User{}).Count(&count)
	return int(count)
}

func CountActiveSubscriptions() int {
	var count int64
	DB.Model(&VLESSKey{}).Where("is_used = true").Count(&count)
	return int(count)
}

func SumPayments(from, to time.Time) float64 {
	var sum int64
	DB.Model(&Payment{}).Where("status = ? AND created_at >= ? AND created_at <= ?", "succeeded", from.Unix(), to.Unix()).Select("sum(amount)").Scan(&sum)
	return float64(sum)
}

func GetKeys(offset, limit int, filter string) []VLESSKey {
	var keys []VLESSKey
	q := DB.Model(&VLESSKey{})
	switch filter {
	case "active":
		q = q.Where("is_used = true")
	case "free":
		q = q.Where("is_used = false")
	case "reserved":
		q = q.Where("reserved_until IS NOT NULL AND is_used = false")
	}
	q.Offset(offset).Limit(limit).Find(&keys)
	return keys
}

func GetPayments(from, to time.Time) []Payment {
	var pays []Payment
	DB.Model(&Payment{}).Where("created_at >= ? AND created_at <= ?", from.Unix(), to.Unix()).Find(&pays)
	return pays
}

func FindUser(idOrName string) (User, error) {
	var user User
	if err := DB.Where("telegram_id = ? OR id = ?", idOrName, idOrName).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func FindKey(idOrKey string) (VLESSKey, error) {
	var key VLESSKey
	if err := DB.Where("id = ? OR key = ?", idOrKey, idOrKey).First(&key).Error; err != nil {
		return key, err
	}
	return key, nil
}
