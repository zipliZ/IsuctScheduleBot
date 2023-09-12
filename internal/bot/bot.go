package bot

import (
	"ScheduleBot/internal/repo"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func NewScheduleBot(token string, db *repo.BotRepo) *ScheduleBot {
	bot, _ := tgbotapi.NewBotAPI(token)
	return &ScheduleBot{buttons: buttons{
		standard: tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Сегодня"),
				tgbotapi.NewKeyboardButton("Завтра"),
				tgbotapi.NewKeyboardButton("День недели"),
				tgbotapi.NewKeyboardButton("Смена Группы"),
			),
		),
		inline: tgbotapi.NewInlineKeyboardMarkup(
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
	}, bot: bot, db: db}
}

func (b *ScheduleBot) Listen() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		var msg tgbotapi.MessageConfig

		if update.Message != nil {
			message := update.Message.Text

			reGroup := regexp.MustCompile(`^\d-\d{1,3}$`)
			reDate := regexp.MustCompile(`^(0[1-9]|[12][0-9]|3[01]).(0[1-9]|1[0-2]).(\d{2}|\d{4})$`)

			switch {
			case message != "/start" && !b.db.UserExists(update.Message.Chat.ID):
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы не авторизированы, нужно прописать или нажать на /start")

			case reGroup.MatchString(message):
				if exist, err := checkGroupExist(message); exist {
					b.db.UpdateUser(update.Message.Chat.ID, message)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Группа установленна")
					b.buttons.standard.Keyboard[0][3].Text = fmt.Sprintf("Сменить (%s)", message)
					msg.ReplyMarkup = b.buttons.standard
				} else if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, formServerErr())
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Такой группы не существует")
				}

			case reDate.MatchString(message):
				if msgText, err := b.getScheduleOnDate(update.Message.Chat.ID, message); err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, formServerErr())
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
				}
			default:
				var weakDay int
				message = strings.ToLower(message)

				switch {
				case message == "/help":
					helpText := formHelpMessage()
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, helpText)

				case message == "/start":
					b.db.CreateUser(update.Message.Chat.ID, update.Message.Chat.UserName)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер группы в форме \"4-185\"")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				case message == "смена группы", strings.Contains(message, "сменить"):
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер группы в форме \"4-185\"")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

				case message == "сегодня":
					if msgText, err := b.getDaySchedule(update.Message.Chat.ID, 0); err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, formServerErr())
					} else {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					}

				case message == "завтра":
					if msgText, err := b.getDaySchedule(update.Message.Chat.ID, 1); err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, formServerErr())
					} else {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					}

				case message == "день недели":
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите день недели")
					msg.ReplyMarkup = b.buttons.inline

				case checkWeekDay(message, &weakDay):
					if msgText, err := b.getWeekSchedule(update.Message.Chat.ID, weakDay); err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, formServerErr())
					} else {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
					}

				case update.Message.IsCommand() && update.Message.Command() == "notify_all" && update.Message.Chat.UserName == "zipliZ":
					msgText := strings.Split(update.Message.Text, "/notify_all ")[1]
					if msgText != "" {
						for _, user := range b.db.GetUsers() {
							msg = tgbotapi.NewMessage(user, msgText)
							if _, err := b.bot.Send(msg); err != nil {
								log.Println(err)
							}
						}
					}
					continue
				default:
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы ввели неправильные данные или неизвестную команду")
				}
			}
		} else if update.CallbackQuery != nil {
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			if _, err := b.bot.Request(callback); err != nil {
				log.Println(err)
			}
			var weakDay int
			if checkWeekDay(strings.ToLower(update.CallbackQuery.Data), &weakDay) {
				if msgText, err := b.getWeekSchedule(update.CallbackQuery.Message.Chat.ID, weakDay); err != nil {
					msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, formServerErr())
				} else {
					msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, msgText)
				}
			}
		}
		msg.Text = escapeSpecialChars(msg.Text)
		msg.ParseMode = "MarkdownV2"
		if _, err := b.bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}

func (b *ScheduleBot) getWeekSchedule(chatId int64, dayOfWeekReq int) (string, error) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Println("Ошибка при установке часового пояса:", err)
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
	group := b.db.GetGroup(chatID)
	if group == "" {
		return "", errors.New("группа не найдена")
	}

	payload := GetScheduleRequest{
		Offset: offset,
	}
	payloadJSON, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://188.120.234.21/today/api", bytes.NewBuffer(payloadJSON))
	if err != nil {
		log.Println(err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	cookie := &http.Cookie{
		Name:  "value",
		Value: group,
	}
	req.AddCookie(cookie)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка HTTP: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", err
	}

	var result GetScheduleResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Println(err)
		return "", err
	}

	return formMessage(result), nil
}

func (b *ScheduleBot) getScheduleOnDate(chatId int64, date string) (string, error) {
	currentTime := time.Now()
	currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	var reqDate time.Time
	var err error
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
