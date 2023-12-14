package bot

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/service"
	"ScheduleBot/internal/store"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"
	_ "time/tzdata"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	holderTypeGroup   = "group"
	holderTypeTeacher = "teacher"
)

func Init(token string, repo repo.Repo, service service.Service, store *store.NotifierStore, endpoints configs.Endpoints) *ScheduleBot {
	bot, _ := tgbotapi.NewBotAPI(token)

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Panic(err)
	}

	scheduleBot := ScheduleBot{buttons: buttons{
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
	}, bot: bot, repo: repo, service: service, store: store, endpoints: endpoints, location: location}

	// Запуск горутины отвечающей за рассылку ежедневного расписания
	go scheduleBot.InitUsersNotification()

	return &scheduleBot
}

func (b *ScheduleBot) Listen() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		go b.processUpdate(update)
	}
}

func (b *ScheduleBot) processUpdate(update tgbotapi.Update) {
	var (
		msg tgbotapi.MessageConfig
		err error
	)

	ctx := context.Background()

	switch {
	case update.Message != nil:
		msg, err = b.handleMessage(ctx, update.Message)
		if err != nil {
			slog.Error("handling message", err, "chat_id", update.Message.Chat.ID, "message", update.Message.Text)
		}
	case update.CallbackQuery != nil:
		msg, err = b.handleCallback(ctx, update.CallbackQuery)
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
}

func (b *ScheduleBot) InitUsersNotification() {
	ctx := context.Background()

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		currentTime := time.Now().In(b.location).Format("15:04")
		if usersToNotify, exist := b.store.Get(store.TargetTime(currentTime)); exist {
			go b.alertUsers(ctx, usersToNotify)
		}
	}
}

func (b *ScheduleBot) alertUsers(ctx context.Context, usersToNotify []int64) {
	sendTicker := time.NewTicker(time.Second / 30)
	for _, user := range usersToNotify {
		go b.sendUserAlert(ctx, user)
		<-sendTicker.C
	}
}

func (b *ScheduleBot) sendUserAlert(ctx context.Context, chatId int64) {
	msgText, err := b.getDaySchedule(ctx, chatId, 0)
	if err != nil {
		slog.Error("getting schedule: ", err)
		return
	}
	msgText = "_*Высылаю тебе сегодняшний день😘*_\n\n" + msgText
	msg := NewMessage(chatId, escapeSpecialChars(msgText), true)
	_, err = b.bot.Send(msg)
	if err != nil {
		slog.Error("sending notification: ", err, "chat_id", chatId)
	}
}

func (b *ScheduleBot) getScheduleByWeekDay(ctx context.Context, chatId int64, dayOfWeekReq int) (string, error) {
	currentDay := time.Now().In(b.location)

	weekNumber := int(currentDay.Weekday())
	var diff int
	if dayOfWeekReq-weekNumber >= 0 {
		diff = dayOfWeekReq - weekNumber
	} else {
		diff = dayOfWeekReq - weekNumber + 7
	}
	return b.getDaySchedule(ctx, chatId, diff)
}

func (b *ScheduleBot) getDaySchedule(ctx context.Context, chatID int64, offset int) (string, error) {
	isStudent, holder := b.repo.GetUserInfo(chatID)
	if holder == "" {
		return "", errors.New("холдер пустой")
	}

	holderType := holderTypeTeacher
	if isStudent {
		holderType = holderTypeGroup
	}

	url := fmt.Sprintf("%s/api/%s/%s/day?offset=%d", b.endpoints.Microservice, holderType, holder, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(req)
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

func (b *ScheduleBot) getScheduleOnDate(ctx context.Context, chatId int64, date string) (string, error) {
	currentTime := time.Now().In(b.location)
	currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	var (
		reqDate time.Time
		err     error
	)

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

	return b.getDaySchedule(ctx, chatId, offset)
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
