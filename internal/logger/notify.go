package logger

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sync"
)

var (
	botInstance *tgbotapi.BotAPI
	adminID     int64
	once        sync.Once
)

// InitNotifier инициализирует Telegram-уведомления об ошибках
func InitNotifier(bot *tgbotapi.BotAPI, admin int64) {
	once.Do(func() {
		botInstance = bot
		adminID = admin
	})
}

// NotifyAdmin отправляет критическое уведомление админу
func NotifyAdmin(msg string) {
	if botInstance == nil || adminID == 0 {
		return
	}
	botInstance.Send(tgbotapi.NewMessage(adminID, "[ALERT] "+msg))
}

// NotifyOnPanic ловит панику, логирует и уведомляет
func NotifyOnPanic(context string) {
	if r := recover(); r != nil {
		NotifyAdmin("Panic in " + context + ": " + toString(r))
	}
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "panic: unknown error"
}
