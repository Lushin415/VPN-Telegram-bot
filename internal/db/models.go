package db

type User struct {
	ID              uint   `gorm:"primaryKey"`
	TelegramID      string `gorm:"uniqueIndex"`
	CurrentDiscount int
}

type Server struct {
	ID       uint `gorm:"primaryKey"`
	Name     string
	IP       string
	Price1   int
	Price3   int
	Price6   int
	Price12  int
	IsActive bool
}

type VLESSKey struct {
	ID               uint `gorm:"primaryKey"`
	ServerID         uint
	Key              string
	IsUsed           bool
	ReservedUntil    *int64
	UserID           *uint
	AssignedAt       *int64
	NotifiedExpiring bool `gorm:"default:false"` // уведомление о скором окончании
}

type Payment struct {
	ID         uint `gorm:"primaryKey"`
	UserID     uint
	YooKassaID string
	Amount     int
	Status     string
	KeyID      *uint // если это продление, то ID ключа
	Months     *int  // срок продления
}

// Order представляет заказ пользователя
type Order struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Server    string
	Tariff    string
	PaymentID string
	Status    string
	CreatedAt int64
}

// Удалена структура Order (оставить только одну реализацию в db.go)
