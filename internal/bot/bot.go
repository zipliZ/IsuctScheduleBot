package bot

import (
	"ScheduleBot/internal/repo"
	"bytes"
	"encoding/json"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"net/http"
	"regexp"
	"strings"
)

func NewScheduleBot(token string, db *repo.BotRepo) *ScheduleBot {
	bot, _ := tgbotapi.NewBotAPI(token)
	return &ScheduleBot{buttons: tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Today"),
			tgbotapi.NewKeyboardButton("Tomorrow"),
			tgbotapi.NewKeyboardButton("ChangeGroup"),
		)), bot: bot, db: db}
}

func (b *ScheduleBot) Listen() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {

			var msg tgbotapi.MessageConfig

			message := update.Message.Text

			re := regexp.MustCompile(`\d-\d{1,3}`)
			if re.MatchString(message) {

				arr := strings.Split(message, "-")
				course, number := arr[0], arr[1]

				url := "http://isuctschedule.ru/api"

				payload := GroupExistRequest{
					LeftPart:  course,
					RightPart: number,
				}

				payloadJSON, marshErr := json.Marshal(payload)
				if marshErr != nil {
					panic(marshErr)
				}

				if _, err := http.Post(url, "application/json", bytes.NewBuffer(payloadJSON)); err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Такой группы не существует")
				} else {
					b.db.UpdateUser(update.Message.Chat.ID, message)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Успешно")
					msg.ReplyMarkup = b.buttons
				}
			} else {

				switch message {
				case "/start":
					b.db.CreateUser(update.Message.Chat.ID)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер группы в форме \"3-185\"")
					//msg.ReplyMarkup = b.buttons
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				case "ChangeGroup":
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер группы в форме \"3-185\"")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

				default:
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
				}
			}
			if _, err := b.bot.Send(msg); err != nil {
				log.Panic(err)
			}
		}
	}
}
