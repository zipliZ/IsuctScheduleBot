package bot

import (
	"ScheduleBot/internal/repo"
	"bytes"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func NewScheduleBot(token string, db *repo.BotRepo) *ScheduleBot {
	bot, _ := tgbotapi.NewBotAPI(token)
	return &ScheduleBot{buttons: tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Сегодня"),
			tgbotapi.NewKeyboardButton("Завтра"),
			tgbotapi.NewKeyboardButton("Смена Группы"),
		)), bot: bot, db: db}
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

			case message != "/start" && b.db.UserExists(update.Message.Chat.ID) == false:
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы не авторизированы, нужно прописать или нажать на /start")

			case reGroup.MatchString(message):
				if b.checkGroupExist(message) {
					b.db.UpdateUser(update.Message.Chat.ID, message)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Группа установленна")
					msg.ReplyMarkup = b.buttons
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Такой группы не существует")
				}

			case reDate.MatchString(message):
				msgText := b.getScheduleOnDate(update.Message.Chat.ID, message)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

			default:
				switch strings.ToLower(message) {

				case "/start":
					b.db.CreateUser(update.Message.Chat.ID, update.Message.Chat.UserName)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер группы в форме \"4-185\"")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				case "смена группы":
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер группы в форме \"4-185\"")
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

				case "сегодня":
					msgText := b.getDaySchedule(update.Message.Chat.ID, 0)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "завтра":
					msgText := b.getDaySchedule(update.Message.Chat.ID, 1)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "понедельник", "пн":
					msgText := b.getWeekSchedule(update.Message.Chat.ID, 1)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "вторник", "вт":
					msgText := b.getWeekSchedule(update.Message.Chat.ID, 2)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "среда", "ср":
					msgText := b.getWeekSchedule(update.Message.Chat.ID, 3)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "четверг", "чт":
					msgText := b.getWeekSchedule(update.Message.Chat.ID, 4)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "пятница", "пт":
					msgText := b.getWeekSchedule(update.Message.Chat.ID, 5)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				case "суббота", "сб":
					msgText := b.getWeekSchedule(update.Message.Chat.ID, 6)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

				default:
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы ввели неправильные данные или неизвестную команду")
				}
			}
		}

		if _, err := b.bot.Send(msg); err != nil {
			log.Println(err)
		}
	}
}

func (b *ScheduleBot) checkGroupExist(group string) bool {
	arr := strings.Split(group, "-")
	course, number := arr[0], arr[1]

	url := "http://188.120.234.21/api"

	payload := GroupExistRequest{
		LeftPart:  course,
		RightPart: number,
	}

	payloadJSON, marshErr := json.Marshal(payload)
	if marshErr != nil {
		log.Println(marshErr)
	}

	if _, err := http.Post(url, "application/json", bytes.NewBuffer(payloadJSON)); err != nil {
		return false
	} else {
		return true
	}
}
func (b *ScheduleBot) getWeekSchedule(chatId int64, dayOfWeekReq int) string {
	currentDay := time.Now()
	weekNumber := int(currentDay.Weekday())
	diff := dayOfWeekReq - weekNumber
	return b.getDaySchedule(chatId, diff)
}

func (b *ScheduleBot) getDaySchedule(chatId int64, offset int) string {
	client := &http.Client{}

	if group := b.db.GetGroup(chatId); group != "" {

		payload := GetScheduleRequest{
			Offset: offset,
		}

		payloadJSON, marshErr := json.Marshal(payload)
		if marshErr != nil {
			log.Println(marshErr)
			return ""
		}

		req, err := http.NewRequest("POST", "http://188.120.234.21/today/api", bytes.NewBuffer(payloadJSON))
		if err != nil {
			log.Println(err)
			return ""
		}
		req.Header.Set("Content-Type", "application/json")

		cookie := &http.Cookie{
			Name:  "value",
			Value: group,
		}
		req.AddCookie(cookie)

		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return ""
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
			return ""
		}
		var result GetScheduleResponse
		if err := json.Unmarshal(body, &result); err != nil {
			log.Println(err)
			return ""
		}
		return b.formMessage(result)
	}
	return ""
}

func (b *ScheduleBot) getScheduleOnDate(chatId int64, date string) string {
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
		return "Неправильно введена дата"
	}
	if reqDate.IsZero() {
		return "Несуществующая дата"
	} else {
		offset := int(reqDate.Sub(currentTime).Hours() / 24)
		return b.getDaySchedule(chatId, offset)
	}
}

func (b *ScheduleBot) formMessage(schedule GetScheduleResponse) string {
	dateString := fmt.Sprintf("Расписание на %s, %s неделя \n\n", getWeekdayName(schedule.Weekday), getWeekName(schedule.Week))

	for _, subject := range schedule.Subjects {
		timeString := fmt.Sprintf("%s-%s |%s\n", subject.Time.Start, subject.Time.End, subject.Type)
		audienceString := subject.Audience[0].Name
		var teacherString string
		for _, teacher := range subject.Teachers {
			teacherString += teacher.Name + "\n"
		}

		subjectString := fmt.Sprintf("%s | %s\n%s%s\n", subject.Name, audienceString, timeString, teacherString)
		dateString += subjectString
	}
	return dateString
}

func getWeekName(weekNumber int) string {
	if weekNumber%2 == 0 {
		return "Вторая"
	}
	return "Первая"
}

func getWeekdayName(weekday int) string {
	weekdays := []string{"Воскресенье", "Понедельник", "Вторник", "Среду", "Четверг", "Пятницу", "Субботу"}
	if weekday == -1 { // вопросы к создателю api
		return weekdays[0]
	}
	return weekdays[weekday]
}
