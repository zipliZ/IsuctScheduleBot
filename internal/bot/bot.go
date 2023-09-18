package bot

import (
	"ScheduleBot/internal/repo"
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

func NewScheduleBot(token string, db repo.Repo) *ScheduleBot {
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
		inlineGroupHistory: tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
				tgbotapi.NewInlineKeyboardButtonData("?", "?"),
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
		reGroup := regexp.MustCompile(`^\d-\d{1,3}$`)
		reDate := regexp.MustCompile(`^(0[1-9]|[12][0-9]|3[01]).(0[1-9]|1[0-2]).(\d{2}|\d{4})$`)

		if update.Message != nil {
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "")
			message := update.Message.Text
			chatId := update.Message.Chat.ID

			switch {
			case message != "/start" && !b.db.UserExists(chatId):
				msg.Text = "Вы не авторизированы, нужно прописать или нажать на /start"

			case reGroup.MatchString(message):
				if exist, err := checkGroupExist(message); exist {
					b.db.UpdateUserGroup(chatId, message)
					msg.Text = "Группа установленна"
					b.buttons.standard.Keyboard[1][1].Text = fmt.Sprintf("Сменить (%s)", message)
					msg.ReplyMarkup = b.buttons.standard
				} else if err != nil {
					msg.Text = formServerErr()
				} else {
					msg.Text = "Такой группы не существует"
				}

			case reDate.MatchString(message):
				var err error
				if msg.Text, err = b.getScheduleOnDate(chatId, message); err != nil {
					msg.Text = formServerErr()
				}
			default:
				var weakDay int
				var digit int
				message = strings.ToLower(message)

				switch {
				case message == "/help":
					msg.Text = formHelpMessage()

				case message == "/feedback":
					msg.Text = `Если ты придумал как можно улучшить нашего бота или нашел баг, то обязательно напиши @zipliZ`

				case message == "/toggle_notifier":
					if b.db.IsDailyNotifierOn(chatId) {
						b.db.UpdateNotificationStatus(chatId, false)
						msg.Text = "Получение ежедневного расписания выключено"
					} else {
						b.db.UpdateNotificationStatus(chatId, true)
						msg.Text = "Получение ежедневного расписания включено"
					}

				case message == "/start":
					b.db.CreateUser(chatId, update.Message.Chat.UserName)
					msg.Text = "Введите номер группы в форме \"4-185\""
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				case strings.Contains(message, "сменить"):
					msg.Text = "Введите номер группы в форме \"4-185\" \n\nПоследние группы:"
					groupsArr := b.db.GetGroupHistory(chatId)
					for i, group := range groupsArr {
						if group == "" {
							group = "-"
						}
						tempGroup := group
						b.buttons.inlineGroupHistory.InlineKeyboard[0][i].Text = group
						b.buttons.inlineGroupHistory.InlineKeyboard[0][i].CallbackData = &tempGroup
					}
					msg.ReplyMarkup = b.buttons.inlineGroupHistory

				case message == "сегодня":
					var err error
					if msg.Text, err = b.getDaySchedule(chatId, 0); err != nil {
						msg.Text = formServerErr()
					}

				case message == "завтра":
					var err error
					if msg.Text, err = b.getDaySchedule(chatId, 1); err != nil {
						msg.Text = formServerErr()
					}
				case isDigit(message, &digit):
					var err error
					if msg.Text, err = b.getDaySchedule(chatId, digit); err != nil {
						msg.Text = formServerErr()
					}

				case message == "неделя":
					msg.Text = "Выберите день недели"
					msg.ReplyMarkup = b.buttons.inlineWeekDays
				case message == "полное расписание":
					group := b.db.GetGroup(chatId)
					if group == "" {
						msg.Text = "У вас не установленна группа"
					} else {
						msg.Text = fmt.Sprintf("__*Ваше полное расписание:*__\nhttp://isuctschedule.ru/share/group/%s", group)
					}

				case checkWeekDay(message, &weakDay):
					var err error
					if msg.Text, err = b.getWeekSchedule(chatId, weakDay); err != nil {
						msg.Text = formServerErr()
					}
				case update.Message.Chat.UserName == "zipliZ" && update.Message.Poll != nil:
					users := b.db.GetUsers()
					for _, user := range users {
						pollForward := tgbotapi.NewForward(user, chatId, update.Message.MessageID)
						if _, err := b.bot.Send(pollForward); err != nil {
							log.Println(err, msg.ChatID)
						}
					}
					continue

				case update.Message.Chat.UserName == "zipliZ" && (update.Message.Command() == "notify_all" || update.Message.Command() == "notify_all_silent"):
					silent := strings.Contains(update.Message.Command(), "silent")

					msgArr := strings.Split(update.Message.Text, " ")
					msgText := strings.Join(msgArr[1:], " ")

					if len(msgArr) > 1 && msgText != "" {
						for _, user := range b.db.GetUsers() {
							msg = tgbotapi.NewMessage(user, msgText)
							msg.DisableNotification = silent

							msg.ParseMode = "Markdown"
							if _, err := b.bot.Send(msg); err != nil {
								log.Println(err, msg.ChatID)
							}
						}
					}
					continue
				default:
					msg.Text = "Вы ввели неправильные данные или неизвестную команду"
				}
			}
		} else if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			callbackData := update.CallbackQuery.Data
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
			msg = tgbotapi.NewMessage(chatID, "")

			var err error
			var weakDay int

			switch {
			case checkWeekDay(strings.ToLower(callbackData), &weakDay):
				if msg.Text, err = b.getWeekSchedule(chatID, weakDay); err != nil {
					msg.Text = formServerErr()
				}

			case reGroup.MatchString(callbackData):
				b.db.UpdateUserGroup(chatID, callbackData)

				deleteCfg := tgbotapi.NewDeleteMessage(chatID, update.CallbackQuery.Message.MessageID)
				_, deleteErr := b.bot.Request(deleteCfg)
				if deleteErr != nil {
					log.Println(deleteErr)
				}
				msg.Text = "Группа изменена"
				b.buttons.standard.Keyboard[1][1].Text = fmt.Sprintf("Сменить (%s)", callbackData)
				msg.ReplyMarkup = b.buttons.standard
			}

			_, err = b.bot.Request(callback)
			if err != nil {
				log.Println(err)
			}
		}
		if msg.Text == "" {
			continue
		}
		msg.Text = escapeSpecialChars(msg.Text)
		msg.ParseMode = "MarkdownV2"
		if _, err := b.bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}

func (b *ScheduleBot) NotifyUsers() {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Println("Ошибка при установке часового пояса:", err)
		return
	}

	for {
		currentTime := time.Now().In(location).Format("15:04:05")
		if currentTime == "4:20:00" {
			usersToNotify := b.db.GetNotificationOn()

			for _, user := range usersToNotify {
				msgText, err := b.getDaySchedule(user, 0)
				if err != nil {
					log.Println(err)
					continue
				}
				msgText = "_*Доброго утречка💟, высылаю тебе сегодняшний день😘*_\n\n" + msgText
				msg := tgbotapi.NewMessage(user, escapeSpecialChars(msgText))
				msg.ParseMode = "MarkdownV2"
				msg.DisableNotification = true
				_, err = b.bot.Send(msg)
				if err != nil {
					log.Println(err, user)
				}
			}
			time.Sleep(1 * time.Second)
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
	url := fmt.Sprintf("http://188.120.234.21:9818/api/group/%s/day?offset=%d", group, offset)
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", errors.New("неправильный статус ответа")
	}
	body, err := io.ReadAll(response.Body)
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
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Println("Ошибка при установке часового пояса:", err)
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
