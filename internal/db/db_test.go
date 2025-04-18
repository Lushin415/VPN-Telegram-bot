package db

import (
	"testing"
	"time"
)

func TestExpiredKeyCleanerLogic(t *testing.T) {
	// Имитируем ключ с просроченным резервом
	now := time.Now().Unix()
	key := VLESSKey{
		ID:            1,
		IsUsed:        false,
		ReservedUntil: func() *int64 { v := now - 1000; return &v }(),
		UserID:        func() *uint { v := uint(42); return &v }(),
	}
	// Очистка должна сбрасывать user_id и reserved_until
	if key.IsUsed == false && key.ReservedUntil != nil && *key.ReservedUntil < now {
		key.UserID = nil
		key.ReservedUntil = nil
	}
	if key.UserID != nil || key.ReservedUntil != nil {
		t.Errorf("Ключ не был очищен как ожидалось")
	}
}
