package bot

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/service"
	"ScheduleBot/internal/store"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ScheduleBot struct {
	bot       *tgbotapi.BotAPI
	repo      repo.Repo
	service   service.Service
	store     *store.NotifierStore
	buttons   buttons
	endpoints configs.Endpoints
}

type buttons struct {
	standard            tgbotapi.ReplyKeyboardMarkup
	inlineWeekDays      tgbotapi.InlineKeyboardMarkup
	inlineHolderHistory tgbotapi.InlineKeyboardMarkup
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
