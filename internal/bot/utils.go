package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	if len(schedule.Subjects) == 0 || schedule.Subjects[0].Name == "Научно-исследовательская работа" && len(schedule.Subjects) == 1 {
		return dateString + "_*Отдыхаем*_"
	}
	for _, subject := range schedule.Subjects {
		if subject.Audience[0].Name == "—" {
			subject.Audience[0].Name = ""
		}
		if subject.Type == "—" {
			subject.Type = ""
		}
		timeString := fmt.Sprintf("%s-%s | __*%s*__\n", subject.Time.Start[0:5], subject.Time.End[0:5], subject.Audience[0].Name)
		var teacherString string
		for _, teacher := range subject.Teachers {
			if teacher.Name == "—" {
				teacherString = ""
				break
			}
			teacherString += teacher.Name + "\n"
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

func checkGroupExist(group string) bool {
	arr := strings.Split(group, "-")
	course, number := arr[0], arr[1]

	url := "http://188.120.234.21/api"

	payload := GroupExistRequest{
		LeftPart:  course,
		RightPart: number,
	}
	payloadJSON, _ := json.Marshal(payload)

	client := http.Client{}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return false
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return true
}
