package bot

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/service"
	"ScheduleBot/internal/store"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
	_ "time/tzdata"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func NewScheduleBot(token string, repo repo.Repo, service service.Service, store *store.NotifierStore, endpoints configs.Endpoints) *ScheduleBot {
	bot, _ := tgbotapi.NewBotAPI(token)
	return &ScheduleBot{buttons: buttons{
		standard: tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Сегодня"),
				tgbotapi.NewKeyboardButton("Завтра"),
				tgbotapi.NewKeyboardButton("Неделя"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Полное расписание"),
				tgbotapi.NewKeyboardButton("Сменить (3-185)"),
			),
		),
		inlineWeekDays: tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Пн", "Понедельник"),
				tgbotapi.NewInlineKeyboardButtonData("Вт", "Вторник"),
				tgbotapi.NewInlineKeyboardButtonData("Ср", "Среда"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Чт", "Четверг"),
				tgbotapi.NewInlineKeyboardButtonData("Пт", "Пятница"),
				tgbotapi.NewInlineKeyboardButtonData("Сб", "Суббота"),
			),
		),
		inlineHolderHistory: tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
			),
		),
	}, bot: bot, repo: repo, service: service, store: store, endpoints: endpoints}
}

func (b *ScheduleBot) Listen() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		go func(update tgbotapi.Update) {
			var msg tgbotapi.MessageConfig
			var err error

			if update.Message != nil {
				msg, err = b.handleMessage(update.Message)
				if err != nil {
					slog.Error("handling message", err, "chat_id", update.Message.Chat.ID, "message", update.Message.Text)
				}
			} else if update.CallbackQuery != nil {
				msg, err = b.handleCallback(update.CallbackQuery)
				if err != nil {
					slog.Error("handling callback", err, "chat_id", update.CallbackQuery.Message.Chat.ID, "data", update.CallbackQuery.Data)
				}
			}

			if msg.Text == "" {
				return
			}
			msg.Text = escapeSpecialChars(msg.Text)
			msg.ParseMode = "MarkdownV2"

			if _, err := b.bot.Send(msg); err != nil {
				slog.Error("sending message :", err, "chat_id:", msg.ChatID)
			}
		}(update)
	}
}

func (b *ScheduleBot) NotifyUsers() {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Error("Ошибка при установке часового пояса:", err)
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		currentTime := time.Now().In(location).Format("15:04")
		if usersToNotify, exist := b.store.Get(currentTime); exist {
			go b.alertUsers(usersToNotify)
		}
	}
}

func (b *ScheduleBot) alertUsers(usersToNotify []int64) {
	sendTicker := time.NewTicker(time.Second / 30)
	for _, user := range usersToNotify {
		go b.sendUserAlert(user)
		<-sendTicker.C
	}
}

func (b *ScheduleBot) sendUserAlert(chatId int64) {
	msgText, err := b.getDaySchedule(chatId, 0)
	if err != nil {
		slog.Error("getting schedule: ", err)
		return
	}
	msgText = "_*Высылаю тебе сегодняшний день😘*_\n\n" + msgText
	msg := tgbotapi.NewMessage(chatId, escapeSpecialChars(msgText))
	msg.ParseMode = "MarkdownV2"
	msg.DisableNotification = true
	_, err = b.bot.Send(msg)
	if err != nil {
		slog.Error("sending notification: ", err, "chat_id", chatId)
	}
}

func (b *ScheduleBot) getWeekSchedule(chatId int64, dayOfWeekReq int) (string, error) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return "", err
	}
	currentDay := time.Now().In(location)

	weekNumber := int(currentDay.Weekday())
	var diff int
	if dayOfWeekReq-weekNumber >= 0 {
		diff = dayOfWeekReq - weekNumber
	} else {
		diff = dayOfWeekReq - weekNumber + 7
	}
	return b.getDaySchedule(chatId, diff)
}

func (b *ScheduleBot) getDaySchedule(chatID int64, offset int) (string, error) {
	isStudent, holder := b.repo.GetUserInfo(chatID)
	if holder == "" {
		return "", errors.New("холдер пустой")
	}

	holderType := "teacher"
	if isStudent {
		holderType = "group"
	}

	url := fmt.Sprintf("%s/api/%s/%s/day?offset=%d", b.endpoints.Microservice, holderType, holder, offset)
	response, err := http.Get(url) //nolint
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.New("неправильный статус ответа")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var result GetScheduleResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return formMessage(result), nil
}

func (b *ScheduleBot) getScheduleOnDate(chatId int64, date string) (string, error) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return "", err
	}

	currentTime := time.Now().In(location)
	currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	var reqDate time.Time
	switch len(date) {
	case 8:
		reqDate, err = time.Parse("02.01.06", date)
	case 10:
		reqDate, err = time.Parse("02.01.2006", date)
	}

	if err != nil {
		return "", err
	}

	if reqDate.IsZero() {
		return "", errors.New("не существующая дата")
	}

	offset := int(reqDate.Sub(currentTime).Hours() / 24)

	return b.getDaySchedule(chatId, offset)
}

func (b *ScheduleBot) getTeacherButtons(names []string) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	for _, name := range names {
		url := fmt.Sprintf("%s/share/teacher/%s", b.endpoints.Frontend, name)
		button := tgbotapi.NewInlineKeyboardButtonURL(name, url)
		row := tgbotapi.NewInlineKeyboardRow(button)
		rows = append(rows, row)
	}

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return inlineKeyboard
}
