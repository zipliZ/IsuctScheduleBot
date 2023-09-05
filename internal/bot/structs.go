package bot

import (
	"ScheduleBot/internal/repo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ScheduleBot struct {
	buttons tgbotapi.ReplyKeyboardMarkup
	bot     *tgbotapi.BotAPI
	db      *repo.BotRepo
}

type GroupExistRequest struct {
	LeftPart  string `json:"leftPart"`
	RightPart string `json:"rightPart"`
}

type GetScheduleRequest struct {
	Offset int `json:"offset"`
}

type GetScheduleResponse struct {
	Week     int `json:"week"`
	Weekday  int `json:"weekday"`
	Subjects []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Time struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"time"`
		Audience []struct {
			Name string `json:"name"`
		} `json:"audience"`
		Teachers []struct {
			Name string `json:"name"`
		} `json:"teachers"`
	} `json:"subjects"`
}
