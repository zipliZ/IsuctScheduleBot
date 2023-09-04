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
