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
	scheduleBot := bot.NewScheduleBot(cfg.Bot.Token, db, cfg.Endpoints)
	log.Println("Bot started")
	go scheduleBot.NotifyUsers()
	scheduleBot.Listen()
}
