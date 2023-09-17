package bot

import (
	"ScheduleBot/internal/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ScheduleBot struct {
	bot     *tgbotapi.BotAPI
	db      repo.Repo
	buttons buttons
}

type buttons struct {
	standard           tgbotapi.ReplyKeyboardMarkup
	inlineWeekDays     tgbotapi.InlineKeyboardMarkup
	inlineGroupHistory tgbotapi.InlineKeyboardMarkup
}

type GroupExistRequest struct {
	LeftPart  string `json:"leftPart"`
	RightPart string `json:"rightPart"`
}

type GetScheduleRequest struct {
	Offset int `json:"offset"`
}

type GetScheduleResponse struct {
	Week    int `json:"week"`
	Weekday int `json:"weekday"`
	Lessons []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Time struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"time"`
		Audience []struct {
			Audience string `json:"audience"`
		} `json:"audience"`
		Teachers []struct {
			Teacher string `json:"teacher"`
		} `json:"teachers"`
	} `json:"lessons"`
}
