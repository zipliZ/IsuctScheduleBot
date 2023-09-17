package bot

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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

func formHelpMessage() string {
	text := `
Фукции бота:
• Выдавать расписание по кнопкам
• Выдавать расписание по дате:
    сообщение формата "08.01.2002" или "01.10.02"
• Выдавать расписание по дню недели:
    сообщение формата "Понедельник" или "Пн"
• Быстро менять группу:
    сообщение типа "4-185"

По всем вопросам: @zipliZ`
	return text
}

func formFeedbackMessage() string {
	text := `
Если ты придумал как можно улучшить нашего бота или нашел баг, то обязательно напиши @zipliZ`
	return text
}

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

func formMessage(schedule GetScheduleResponse) string {
	dateString := fmt.Sprintf("_ __*Расписание на %s, %s неделя*__ _\n\n", getWeekdayName(schedule.Weekday), getWeekName(schedule.Week))
	if len(schedule.Lessons) == 0 || schedule.Lessons[0].Name == "Научно-исследовательская работа" && len(schedule.Lessons) == 1 {
		return dateString + "_*Отдыхаем*_"
	}
	for _, subject := range schedule.Lessons {
		if subject.Audience[0].Audience == "—" {
			subject.Audience[0].Audience = ""
		}
		if subject.Type == "—" {
			subject.Type = ""
		}
		timeString := fmt.Sprintf("%s-%s | __*%s*__\n", subject.Time.Start[0:5], subject.Time.End[0:5], subject.Audience[0].Audience)
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

func checkGroupExist(group string) (bool, error) {
	url := fmt.Sprintf("http://188.120.234.21:9818/api/check/%s", group)
	request, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return false, err
	}
	if request.StatusCode != 200 {
		return false, nil
	}
	return true, nil
}
func isDigit(message string, digit *int) bool {
	var err error
	*digit, err = strconv.Atoi(message)
	if err != nil {
		return false
	}
	return true

}

func formServerErr() string {
	serverErrString := `Проблемы на стороне сервера, ожидайте исправления

По вопросам к @anCreny, если не отвечает, то к @zipliZ`
	return serverErrString
}
