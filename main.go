package main

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/bot"
	"ScheduleBot/internal/repo"
	"log"
)

func main() {
	cfg := configs.DecodeConfig()

	db := repo.NewBotRepo(cfg.Db)
	scheduleBot := bot.NewScheduleBot(cfg.Bot.Token, db)
	log.Println("Bot started")
	scheduleBot.Listen()
}
