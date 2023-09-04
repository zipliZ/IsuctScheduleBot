package main

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/bot"
	"ScheduleBot/internal/repo"
)

func main() {
	cfg := configs.DecodeConfig()

	db := repo.NewBotRepo(cfg.Db)
	scheduleBot := bot.NewScheduleBot(cfg.Bot.Token, db)
	scheduleBot.Listen()
}
