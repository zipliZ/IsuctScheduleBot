package bot

import (
	"ScheduleBot/internal/repo"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func escapeSpecialChars(input string) string {
	replacer := strings.NewReplacer(
		"-", "\\-",
		"|", "\\|",
		".", "\\.",
		"(", "\\(",
		")", "\\)",
	)
	return replacer.Replace(input)
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

// Проверяет строку на день недели, если день недели число дня записывается в weekDay
func checkWeekDay(message string, weakDay *int) bool {
	switch message {
	case "понедельник", "пн":
		*weakDay = 1
	case "вторник", "вт":
		*weakDay = 2
	case "среда", "ср":
		*weakDay = 3
	case "четверг", "чт":
		*weakDay = 4
	case "пятница", "пт":
		*weakDay = 5
	case "суббота", "сб":
		*weakDay = 6
	default:
		return false
	}
	return true
}

func NewMessage(chatId int64, text string, silent bool) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, text)
	msg.ParseMode = "MarkdownV2"
	msg.DisableNotification = silent
	return msg
}

func formMessage(schedule GetScheduleResponse) string {
	dateString := fmt.Sprintf("_ __*Расписание на %s, %s неделя*__ _\n\n", getWeekdayName(schedule.Weekday), getWeekName(schedule.Week))
	if len(schedule.Lessons) == 0 || schedule.Lessons[0].Name == "Научно-исследовательская работа" && len(schedule.Lessons) == 1 {
		return dateString + "_*Отдыхаем*_"
	}
	for _, subject := range schedule.Lessons {
		var audienceString string
		for _, audience := range subject.Audience {
			if audience.Audience == "—" {
				audience.Audience = ""
				break
			}
			audienceString += " " + "__*" + audience.Audience + "*__"
		}
		if subject.Type == "—" {
			subject.Type = ""
		}
		timeString := fmt.Sprintf("%s-%s |%s\n", subject.Time.Start[0:5], subject.Time.End[0:5], audienceString)
		var teacherString string
		for _, teacher := range subject.Teachers {
			if teacher.Teacher == "—" {
				teacherString = ""
				break
			}
			teacherString += teacher.Teacher + "\n"
		}
		var typeSymbol string
		switch subject.Type {
		case "лк.":
			typeSymbol = "🟩"
		case "пр.з.":
			typeSymbol = "🟧"
		case "лаб.":
			typeSymbol = "🟦"
		default:
			typeSymbol = "🤍"
		}
		subjectString := fmt.Sprintf("%s*%s |* *%s*\n%s*%s*\n", typeSymbol, subject.Name, subject.Type, timeString, teacherString)
		dateString += subjectString
	}

	return dateString
}

func checkHolderExistence(microUrl string, isStudent bool, holder string) (bool, error) {
	holderType := holderTypeTeacher
	if isStudent {
		holderType = holderTypeGroup
	}

	url := fmt.Sprintf("%s/api/check/%s/%s", microUrl, holderType, holder)
	response, err := http.Get(url) //nolint
	if err != nil {
		log.Println(err)
		return false, err
	}
	if response.StatusCode != http.StatusOK {
		return false, nil
	}
	return true, nil
}

func getCommonTeacherNames(ctx context.Context, microUrl, name string) ([]string, error) {
	url := fmt.Sprintf("%s/api/associatedWith/%s", microUrl, name)

	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, nil
	}

	var teachersNames []string
	if decodeErr := json.NewDecoder(response.Body).Decode(&teachersNames); decodeErr != nil {
		return nil, err
	}
	return teachersNames, nil
}

// Функция возвращает является ли строка чилом, если оно число значение присваивается в digit
func isDigit(message string, digit *int) bool {
	var err error
	*digit, err = strconv.Atoi(message)

	return err == nil
}

func formHelpMessage() string {
	text := `
__*Фукции бота:*__
• *Выдавать расписание по кнопкам*

• *Выдавать расписание по дате:*
    "08.01.2002" или "01.10.02"

• *Выдавать расписание по дню недели:*
    "Понедельник" или "Пн"

• *Быстро менять группу/преподавателя:*
	"4-185" или "Константинов Е.С."

• *Включение/выключение ежедневной рассылки расписания на текущий день*
   (используйте /toggle\_notifier)

• *Установка времени оповещения*
	"таймер 04:19"

• *Искать расписание преподавателя:*
    "Поиск Константинов"
    "Поиск Конст"

• *Быстро получать расписание по цифрам:*
    0 — получить сегодняшний день
    1 — получить завтрашний день
   -1 — получить вчерашний день

• *Получить список топ донатеров*
	(используйте /donate)


*Если у тебя есть вопросы или ты придумал как можно улучшить нашего бота или нашел баг, то обязательно напиши @zipliZ*`
	return text
}

func formServerErr() string {
	serverErrString := `
Проблемы на стороне сервера, ожидайте исправления

По вопросам к @anCreny, если не отвечает, то к @zipliZ`
	return serverErrString
}

func formDonatorsMessage(donators []repo.Donator) string {
	return fmt.Sprintf(`
*Топ любимых нами жорика-спасателей:*
	*1.__%s__ — %dр.*
	*2.__%s__ — %dр.*
	*3.__%s__ — %dр.*

*С каждым донатом вы сохраняете жизнь минимум одному жорику, задумайтесь.
Если вы тоже не любите есть жориков или хотели бы висеть сверху, пожертвовать можно:*
• По номеру телефона:
		__\+79807393606__
• По ссылке: 
		__https://www.tinkoff.ru/cf/9y6xKQyaGH3__

*жорик — 🪳*`,
		donators[0].Name, donators[0].AmountOfDonation,
		donators[1].Name, donators[1].AmountOfDonation,
		donators[2].Name, donators[2].AmountOfDonation,
	)
}

func formStartMessage() string {
	return `
*Если вы студент, введите номер группы*
 пример \"4-185\"
*Если вы учитель, то введите свое ФИО*
 пример \"Константинов Е.С.\"

Весь список команд \можно посмотреть командой */help*
`
}
