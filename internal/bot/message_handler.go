package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

func (b *ScheduleBot) handleMessage(message *tgbotapi.Message) (tgbotapi.MessageConfig, error) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "")
	msgText := strings.ToLower(message.Text)
	chatId := message.Chat.ID

	reGroup := regexp.MustCompile(`^\d-\d{1,3}$`)
	reTeacher := regexp.MustCompile("^[А-ЯЁ][а-яё]+\\s[А-ЯЁ]\\.[А-ЯЁ]\\.$")
	reDate := regexp.MustCompile(`^(0[1-9]|[12][0-9]|3[01]).(0[1-9]|1[0-2]).(\d{2}|\d{4})$`)

	var weakDay int
	var digit int
	var err error

	switch {
	case msgText != "/start" && !b.repo.UserExists(chatId):
		msg.Text = "Вы не авторизированы, нужно прописать или нажать на /start"

	case reGroup.MatchString(msgText):
		if exist, checkErr := checkHolderExist(b.endpoints.Microservice, true, msgText); exist {
			b.repo.UpdateUserHolder(chatId, true, msgText)
			msg.Text = "Группа установлена"
			b.buttons.standard.Keyboard[1][1].Text = fmt.Sprintf("Сменить (%s)", msgText)
			msg.ReplyMarkup = b.buttons.standard
		} else if checkErr != nil {
			err = checkErr
			msg.Text = formServerErr()
		} else {
			msg.Text = "Такой группы не существует"
		}

	case reTeacher.MatchString(message.Text):
		if exist, checkErr := checkHolderExist(b.endpoints.Microservice, false, message.Text); exist {
			b.repo.UpdateUserHolder(chatId, false, message.Text)
			msg.Text = "Преподаватель выбран"
			b.buttons.standard.Keyboard[1][1].Text = fmt.Sprintf("Сменить (%s)", message.Text)
			msg.ReplyMarkup = b.buttons.standard
		} else if checkErr != nil {
			err = checkErr
			msg.Text = formServerErr()
		} else {
			msg.Text = "Такого преподавателя не существует"
		}

	case reDate.MatchString(msgText):
		if msg.Text, err = b.getScheduleOnDate(chatId, msgText); err != nil {
			msg.Text = formServerErr()
		}

	case msgText == "/help":
		msg.Text = formHelpMessage()

	case msgText == "/feedback":
		msg.Text = `Если ты придумал как можно улучшить нашего бота или нашел баг, то обязательно напиши @zipliZ`

	case msgText == "/donate":
		donators := b.repo.GetTop3Donators()
		msg.Text = formDonatorsMessage(donators)

	case msgText == "/toggle_notifier":
		msg.Text = b.service.ToggleNotification(chatId)

	case strings.HasPrefix(msgText, "таймер"):
		msgArr := strings.Split(msgText, " ")
		reqTime := strings.Join(msgArr[1:], " ")

		reTime := regexp.MustCompile("^([0-1][0-9]|2[0-3]):([0-5][0-9])$")

		if reqTime == "" {
			msg.Text = "*Вы забыли указать время* \n пример - таймер 04:20"
		} else if reTime.MatchString(reqTime) {
			msg.Text = b.service.UpdateTimer(chatId, reqTime)
		} else {
			msg.Text = "*Время не соответствует формату* \n пример - таймер 04:19"
		}

	case msgText == "/start":
		b.repo.CreateUser(chatId, message.Chat.UserName)
		msg.Text = formStartMessage()

		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

	case strings.Contains(msgText, "сменить"):
		msg.Text = `
*Введите номер группы или ФИО преподавателя*
Пример \"*4-185*\" , \"*Константинов Е.С.*\" 

История использования:`
		historyArr := b.repo.GetHistory(chatId)
		for i, holder := range historyArr {
			if holder == "" {
				holder = "-"
			}
			tempGroup := holder
			b.buttons.inlineHolderHistory.InlineKeyboard[i][0].Text = holder
			b.buttons.inlineHolderHistory.InlineKeyboard[i][0].CallbackData = &tempGroup
		}
		msg.ReplyMarkup = b.buttons.inlineHolderHistory

	case msgText == "сегодня":
		if msg.Text, err = b.getDaySchedule(chatId, 0); err != nil {
			msg.Text = formServerErr()
		}

	case msgText == "завтра":
		if msg.Text, err = b.getDaySchedule(chatId, 1); err != nil {
			msg.Text = formServerErr()
		}

	case isDigit(msgText, &digit):
		if msg.Text, err = b.getDaySchedule(chatId, digit); err != nil {
			msg.Text = formServerErr()
		}

	case msgText == "неделя":
		msg.Text = "Выберите день недели"
		msg.ReplyMarkup = b.buttons.inlineWeekDays

	case msgText == "полное расписание":
		isStudent, holder := b.repo.GetUserInfo(chatId)
		if holder == "" {
			msg.Text = "У вас не установлена группа"
		} else {
			holderType := "group"
			if !isStudent {
				holderType = "teacher"
				holder = strings.ReplaceAll(holder, " ", "-")
			}
			msg.Text = fmt.Sprintf("__*Ваше полное расписание:*__\n[%s/share/%s/%s]", b.endpoints.Frontend, holderType, holder)
		}

	case checkWeekDay(msgText, &weakDay):
		if msg.Text, err = b.getWeekSchedule(chatId, weakDay); err != nil {
			msg.Text = formServerErr()
		}

	case strings.HasPrefix(msgText, "поиск"):
		msgArr := strings.Split(msgText, " ")
		searchText := strings.Join(msgArr[1:], " ")

		if searchText == "" {
			msg.Text = "Вы забыли ввести фамилию"

		} else if namesArr, getErr := getCommonTeacherNames(b.endpoints.Microservice, searchText); getErr != nil {
			err = getErr
			msg.Text = formServerErr()

		} else if len(namesArr) == 0 {
			msg.Text = "Такого преподавателя не существует"

		} else {
			msg.Text = "Выберите нужного вам преподавателя"
			msg.ReplyMarkup = b.getTeacherButtons(namesArr)
		}

	case message.Chat.UserName == "zipliZ" && message.Poll != nil:
		users := b.repo.GetUsers()
		ticker := time.NewTicker(time.Second / 30)
		for _, user := range users {
			select {
			case <-ticker.C:
				go func(user int64) {
					pollForward := tgbotapi.NewForward(user, chatId, message.MessageID)
					if _, err := b.bot.Send(pollForward); err != nil {
						slog.Error("sending pull", err, "chat_id:", msg.ChatID)
					}
				}(user)
			}
		}
		msg.Text = "Голосование разосланно"

	default:
		msg.Text = "Вы ввели неправильные данные или неизвестную команду"
	}

	return msg, err
}
