package main

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/bot"
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/service"
	"ScheduleBot/internal/store"
	"log"
	"log/slog"
)

func main() {
	cfg := configs.DecodeConfig("./config.yaml")

	botRepo, err := repo.New(cfg.Db)
	if err != nil {
		log.Panic(err)
	}
	notifierStore := store.New()
	botService := service.Init(botRepo, *notifierStore)
	scheduleBot := bot.Init(cfg.BotToken, botRepo, botService, notifierStore, cfg.Endpoints)
	slog.Info("Bot started")
	scheduleBot.Listen()
}
