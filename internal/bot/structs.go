package bot

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/service"
	"ScheduleBot/internal/store"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ScheduleBot struct {
	bot       *tgbotapi.BotAPI
	repo      repo.Repo
	service   service.Service
	store     *store.NotifierStore
	buttons   buttons
	endpoints configs.Endpoints
	location  *time.Location
}

type buttons struct {
	standard            tgbotapi.ReplyKeyboardMarkup
	inlineWeekDays      tgbotapi.InlineKeyboardMarkup
	inlineHolderHistory tgbotapi.InlineKeyboardMarkup
}

type GetScheduleResponse struct {
	Week    int      `json:"week"`
	Weekday int      `json:"weekday"`
	Lessons []Lesson `json:"lessons"`
}

type Teacher struct {
	Teacher string `json:"teacher"`
}

type Lesson struct {
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	Time     Time       `json:"time"`
	Audience []Audience `json:"audience"`
	Teachers []Teacher  `json:"teachers"`
}

type Time struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Audience struct {
	Audience string `json:"audience"`
}
