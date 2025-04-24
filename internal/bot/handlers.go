package bot

import (
	"VPN-Telegram-bot/config"
	"VPN-Telegram-bot/internal/admin"
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/services"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
	"time"
)

var rateLimiter = NewRateLimiter()

func HandleUpdate(botapi *tgbotapi.BotAPI, update tgbotapi.Update) {
	// Проверяем и добавляем/обновляем пользователя в БД при любом апдейте
	if update.Message != nil && update.Message.From != nil {
		telegramID := strconv.FormatInt(update.Message.From.ID, 10)
		var user db.User
		err := db.DB.Where("telegram_id = ?", telegramID).First(&user).Error
		if err != nil {
			// Пользователь не найден — создаём
			user = db.User{TelegramID: telegramID}
			db.DB.Create(&user)
		} else {
			// Пользователь найден — обновляем TelegramID на случай, если что-то изменилось
			if user.TelegramID != telegramID {
				db.DB.Model(&user).Update("telegram_id", telegramID)
			}
		}
	}

	if update.CallbackQuery != nil {
		log.Printf("Received callback_query: %+v", update.CallbackQuery)
		// Обработка inline-кнопок
		data := update.CallbackQuery.Data
		if strings.HasPrefix(data, "buy_server_") {
			serverIDstr := strings.TrimPrefix(data, "buy_server_")
			_, err := strconv.ParseInt(serverIDstr, 10, 64)
			if err != nil {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка выбора сервера"))
				return
			}
			// Кнопки выбора тарифа (1, 3, 6, 12 месяцев)
			var rows [][]tgbotapi.InlineKeyboardButton
			for _, m := range []int{1, 3, 6, 12} {
				label := strconv.Itoa(m) + " мес."
				row := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(label, "buy_tariff_"+serverIDstr+"_"+strconv.Itoa(m)),
				)
				rows = append(rows, row)
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Выберите срок подписки:")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
			botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сервер выбран"))
			return
		}
		if strings.HasPrefix(data, "buy_tariff_") {
			parts := strings.Split(data, "_")
			if len(parts) != 4 {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка выбора тарифа"))
				return
			}
			serverID, err := strconv.ParseInt(parts[2], 10, 64)
			if err != nil {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка выбора тарифа"))
				return
			}
			months, _ := strconv.Atoi(parts[3])
			userTGID := update.CallbackQuery.From.ID
			// Найти/создать пользователя по TelegramID
			var user db.User
			db.DB.FirstOrCreate(&user, db.User{TelegramID: strconv.FormatInt(userTGID, 10)})
			// Проверить, что сервер существует и активен
			var server db.Server
			err = db.DB.First(&server, serverID).Error
			if err != nil || !server.IsActive {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сервер не найден или неактивен"))
				return
			}
			// Запустить оплату с выбранным сервером
			url, err := ReserveVLESSKeyAndCreatePayment(user.ID, uint(serverID), months)
			if err != nil {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка: "+err.Error()))
				return
			}
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ссылка на оплату: "+url)
			botapi.Send(msg)
			botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Платёж создан"))
			return
		}
		if strings.HasPrefix(data, "renew_tariff_") {
			parts := strings.Split(data, "_")
			if len(parts) != 4 {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка выбора тарифа продления"))
				return
			}
			keyID, _ := strconv.ParseInt(parts[2], 10, 64)
			months, _ := strconv.Atoi(parts[3])
			// Найти ключ
			var key db.VLESSKey
			db.DB.First(&key, keyID)
			if key.ID == 0 {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Подписка не найдена"))
				return
			}
			// Найти пользователя
			var user db.User
			db.DB.Where("telegram_id = ?", strconv.FormatInt(update.CallbackQuery.From.ID, 10)).First(&user)
			if key.UserID == nil || *key.UserID != user.ID {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Это не ваша подписка"))
				return
			}
			// Найти сервер
			var server db.Server
			db.DB.First(&server, key.ServerID)
			// Рассчитать цену
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
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Некорректный срок продления"))
				return
			}
			// Учесть скидку пользователя
			if user.CurrentDiscount > 0 {
				price = price * (100 - user.CurrentDiscount) / 100
			}
			// Создать платёж для продления
			pay := db.Payment{
				UserID: user.ID,
				Amount: price,
				Status: "pending",
				KeyID:  &key.ID,
				Months: &months,
			}
			db.DB.Create(&pay)
			// Получить ссылку на оплату
			paymentID, url, err := ReserveRenewPaymentAndCreateYooKassa(&pay, server, user)
			if err != nil {
				botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ошибка: "+err.Error()))
				return
			}
			db.DB.Model(&pay).Update("yoo_kassa_id", paymentID)
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Ссылка на оплату продления: "+url)
			botapi.Send(msg)
			botapi.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Платёж на продление создан"))
			return
		}
		return
	}

	if update.Message == nil {
		return
	}
	userID := update.Message.From.ID
	cmd := strings.Fields(update.Message.Text)[0]
	if !admin.IsAdmin(userID) && rateLimiter.IsLimited(userID, cmd) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, не так быстро! Подождите пару секунд...")
		msg.ReplyMarkup = GetReplyKeyboard(userID)
		botapi.Send(msg)
		return
	}
	// Динамическая клавиатура с учётом админа
	keyboard := GetReplyKeyboard(userID)
	// Вызов обработчика админ-команд
	if admin.IsAdmin(userID) && strings.HasPrefix(update.Message.Text, "/admin_") {
		admin.HandleAdminCommand(botapi, &update)
		return
	}
	if update.Message != nil {
		switch {
		case strings.HasPrefix(update.Message.Text, "/start"):
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Добро пожаловать! Для покупки VPN используйте /buy")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
		case strings.HasPrefix(update.Message.Text, "/buy"):
			// 1. Получить список серверов
			var servers []db.Server
			db.DB.Where("is_active = true").Find(&servers)
			if len(servers) == 0 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Извините, сейчас нет доступных серверов для покупки. Попробуйте позже или напишите /support.")
				msg.ReplyMarkup = keyboard
				botapi.Send(msg)
				return
			}
			// 2. Сформировать inline-кнопки по серверам
			var rows [][]tgbotapi.InlineKeyboardButton
			for _, s := range servers {
				row := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(s.Name, "buy_server_"+strconv.FormatInt(int64(s.ID), 10)),
				)
				rows = append(rows, row)
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите сервер для покупки:")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
		case strings.HasPrefix(update.Message.Text, "/support"):
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Поддержка: напишите вашему администратору.")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
		case strings.HasPrefix(update.Message.Text, "/subscriptions"):
			userTGID := strconv.FormatInt(update.Message.From.ID, 10)
			var user db.User
			db.DB.Where("telegram_id = ?", userTGID).First(&user)
			var keys []db.VLESSKey
			db.DB.Where("user_id = ? AND is_used = true", user.ID).Find(&keys)
			if len(keys) == 0 {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет активных подписок. Для покупки используйте /buy.")
				msg.ReplyMarkup = keyboard
				botapi.Send(msg)
				return
			}
			var text strings.Builder
			text.WriteString("Ваши активные подписки:\n")
			for _, k := range keys {
				text.WriteString("Ключ: " + k.Key + "\n")
				if k.ReservedUntil != nil {
					exp := time.Unix(*k.ReservedUntil, 0)
					text.WriteString("Действует до: " + exp.Format("2006-01-02 15:04:05") + "\n")
				}
				text.WriteString("/renew_" + strconv.FormatInt(int64(k.ID), 10) + " — продлить\n\n")
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text.String()+"Спасибо, что пользуетесь нашим VPN!")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
			return
		case strings.HasPrefix(update.Message.Text, "/getkey"):
			userTGID := strconv.FormatInt(update.Message.From.ID, 10)
			var user db.User
			db.DB.Where("telegram_id = ?", userTGID).First(&user)
			var key db.VLESSKey
			db.DB.Where("user_id = ? AND is_used = true", user.ID).Order("assigned_at desc").First(&key)
			if key.Key == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "У вас нет активных оплаченных подписок. Для покупки используйте /buy.")
				msg.ReplyMarkup = keyboard
				botapi.Send(msg)
				return
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ваш VPN-ключ: "+key.Key+"\nСпасибо, что выбрали наш сервис!")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
			return
		case strings.HasPrefix(update.Message.Text, "/renew_"):
			parts := strings.Split(update.Message.Text, "_")
			if len(parts) != 2 {
				botapi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Некорректная команда продления. Используйте /help для справки."))
				return
			}
			keyID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				botapi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Некорректный ID подписки. Используйте /help для справки."))
				return
			}
			var key db.VLESSKey
			db.DB.First(&key, keyID)
			if key.ID == 0 {
				botapi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Подписка не найдена. Обратитесь в /support."))
				return
			}
			// Найти пользователя
			var user db.User
			db.DB.Where("telegram_id = ?", strconv.FormatInt(update.Message.From.ID, 10)).First(&user)
			if key.UserID == nil || *key.UserID != user.ID {
				botapi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Это не ваша подписка. Обратитесь в /support."))
				return
			}
			// Предложить выбрать тариф продления (1, 3, 6, 12 мес)
			var rows [][]tgbotapi.InlineKeyboardButton
			for _, m := range []int{1, 3, 6, 12} {
				label := strconv.Itoa(m) + " мес."
				row := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(label, "renew_tariff_"+strconv.FormatInt(keyID, 10)+"_"+strconv.Itoa(m)),
				)
				rows = append(rows, row)
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите срок продления:")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
		case strings.HasPrefix(update.Message.Text, "/help"):
			helpText := `Доступные команды:
/buy — Купить VPN
/subscriptions — Мои подписки
/renew_<id> — Продлить подписку
/getkey — Повторно получить ключ
/support — Связаться с поддержкой
/help — Показать эту справку

Покупка: /buy → выберите сервер и срок → получите ссылку для оплаты.
Продление: /subscriptions → /renew_<id> → выберите срок → оплатите.
После оплаты бот автоматически выдаст или продлит ваш ключ.`
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, helpText)
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
			return
		case strings.HasPrefix(update.Message.Text, "/sync_xray_keys"):
			adminID, err := strconv.ParseInt(config.AppCfg.AdminTelegramID, 10, 64)
			if err == nil && update.Message != nil && update.Message.From != nil && update.Message.From.ID == adminID {
				if update.Message.Text == "/sync_xray_keys" {
					go func() {
						uuids, err := services.GetAllXrayUUIDsFromRemote("root", "150.241.85.73", "59421", "/root/.ssh/id_ed25519")
						if err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка получения uuid из config.json: "+err.Error())
							botapi.Send(msg)
							return
						}
						err = services.CleanDBKeysNotInXray(uuids)
						if err != nil {
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка очистки БД: "+err.Error())
							botapi.Send(msg)
							return
						}
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Синхронизация завершена. Удалены все ключи, которых нет в config.json.")
						botapi.Send(msg)
					}()
				}
			}
		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда. Используйте /help для списка всех возможностей.")
			msg.ReplyMarkup = keyboard
			botapi.Send(msg)
		}
	}

}
