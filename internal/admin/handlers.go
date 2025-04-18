package admin

import (
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/logger"
	"VPN-Telegram-bot/internal/services"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func IsAdmin(userID int64) bool {
	return userID == AdminTelegramID
}

func HandleAdminCommand(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	if update.Message == nil || update.Message.From.ID != AdminTelegramID {
		return
	}
	cmd := update.Message.Command()
	switch cmd {
	case "admin_stats":
		handleStats(bot, update)
	case "admin_keys":
		handleKeys(bot, update)
	case "admin_logs":
		handleLogs(bot, update)
	case "admin_servers":
		handleServers(bot, update)
	case "admin_reload":
		handleReload(bot, update)
	case "admin_payments":
		handlePayments(bot, update)
	case "admin_broadcast":
		handleBroadcast(bot, update)
	case "admin_user":
		handleUser(bot, update)
	case "admin_key":
		handleKey(bot, update)
	case "admin_release":
		handleRelease(bot, update)
	case "admin_backup":
		handleBackup(bot, update)
	case "admin_restore":
		handleRestore(bot, update)
	case "admin_addserver":
		handleAddServer(bot, update)
	}
	logger.LogAdminAction(AdminTelegramID, cmd, update.Message.Text)
}

func handleStats(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	users := db.CountUsers()
	activeSubs := db.CountActiveSubscriptions()
	todayPayments := db.SumPayments(time.Now().Truncate(24*time.Hour), time.Now())
	monthPayments := db.SumPayments(time.Now().AddDate(0, 0, -30), time.Now())
	allPayments := db.SumPayments(time.Time{}, time.Now())
	msg := fmt.Sprintf(
		"Пользователей: %d\nАктивных подписок: %d\nПлатежи: сегодня: %.2f₽, месяц: %.2f₽, всего: %.2f₽",
		users, activeSubs, todayPayments, monthPayments, allPayments)
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
}

func handleKeys(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	// Пагинация и фильтрация реализуются через inline-кнопки (см. ниже)
	// Здесь пример первой страницы всех ключей
	keys := db.GetKeys(0, 20, "all")
	var sb strings.Builder
	sb.WriteString("Список ключей (первые 20):\n")
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("ID: %d, Key: %s, User: %v, Статус: %v\n", k.ID, k.Key, k.UserID, k.IsUsed))
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, sb.String()))
}

func handleLogs(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Функция логов не реализована"))
}

func handleServers(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	statuses := services.GetServerStatuses()
	msg := "Статус серверов:\n"
	for _, s := range statuses {
		msg += fmt.Sprintf("%s (%s): %s, нагрузка: %d, последний пинг: %s\n",
			s.Name, s.IP, s.Status, s.Load, s.LastChecked.Format("02.01 15:04"))
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
}

func handleReload(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	err := services.ReloadServers()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка перезагрузки: "+err.Error()))
		return
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Конфиг серверов успешно перезагружен."))
}

func handlePayments(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	// Пример: /admin_payments 2024-01-01 2024-01-31
	args := strings.Fields(update.Message.CommandArguments())
	var from, to time.Time
	var err error
	if len(args) == 2 {
		from, err = time.Parse("2006-01-02", args[0])
		if err != nil {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат даты (from)"))
			return
		}
		to, err = time.Parse("2006-01-02", args[1])
		if err != nil {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат даты (to)"))
			return
		}
	} else {
		from = time.Now().AddDate(0, 0, -30)
		to = time.Now()
	}
	payments := db.GetPayments(from, to)
	var sb strings.Builder
	for _, p := range payments {
		sb.WriteString(fmt.Sprintf("ID: %d, User: %v, Amount: %.2f, Status: %s\n", p.ID, p.UserID, p.Amount, p.Status))
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, sb.String()))
}

func handleBroadcast(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	// Подтверждение через inline-кнопки (реализация в основном обработчике)
	// После подтверждения — рассылка всем пользователям
}

func handleUser(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите userID или username"))
		return
	}
	user, err := db.FindUser(args[0])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден"))
		return
	}
	msg := fmt.Sprintf("User: %v", user)
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
}

func handleKey(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите ID или ключ"))
		return
	}
	key, err := db.FindKey(args[0])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ключ не найден"))
		return
	}
	msg := fmt.Sprintf("Key: %v\nПользователь: %v\nСтатус: %v", key, key.UserID, key.IsUsed)
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
}

func handleRelease(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите ID ключа"))
		return
	}
	// Подтверждение через inline-кнопки (реализация в основном обработчике)
}

func handleBackup(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	backupDir := "backups"
	os.MkdirAll(backupDir, 0o755)
	filename := filepath.Join(backupDir, "backup_"+time.Now().Format("20060102_150405")+".dump")
	dsn := os.Getenv("DATABASE_URL")
	err := BackupDatabase(filename, dsn)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка резервного копирования: "+err.Error()))
		return
	}
	// Отправить файл админу
	file := tgbotapi.NewDocument(update.Message.Chat.ID, tgbotapi.FilePath(filename))
	file.Caption = "Резервная копия БД успешно создана"
	bot.Send(file)
	// (Опционально) удалить файл после отправки
	_ = os.Remove(filename)
}

func handleRestore(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	backupDir := "backups"
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) < 1 {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите имя файла для восстановления"))
		return
	}
	filename := filepath.Join(backupDir, args[0])
	dsn := os.Getenv("DATABASE_URL")
	err := RestoreDatabase(filename, dsn)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка восстановления: "+err.Error()))
		return
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Восстановление успешно завершено из файла: "+args[0]))
}

func handleAddServer(bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) < 7 {
		msg := "Использование: /admin_addserver <Name> <IP> <Price1> <Price3> <Price6> <Price12> <is_active(0/1)>"
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		return
	}
	name := args[0]
	ip := args[1]
	price1, err1 := strconv.Atoi(args[2])
	price3, err3 := strconv.Atoi(args[3])
	price6, err6 := strconv.Atoi(args[4])
	price12, err12 := strconv.Atoi(args[5])
	isActive := args[6] == "1"
	if err1 != nil || err3 != nil || err6 != nil || err12 != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка: цены должны быть целыми числами"))
		return
	}
	server := db.Server{
		Name:     name,
		IP:       ip,
		Price1:   price1,
		Price3:   price3,
		Price6:   price6,
		Price12:  price12,
		IsActive: isActive,
	}
	err := db.DB.Create(&server).Error
	if err != nil {
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка добавления сервера: "+err.Error()))
		return
	}
	bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Сервер успешно добавлен: "+name+" ("+ip+")"))
}
