package main

import (
	"ScheduleBot/configs"
	"ScheduleBot/internal/bot"
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/service"
	"ScheduleBot/internal/store"
	"log/slog"
)

func main() {
	cfg := configs.DecodeConfig("./config.yaml")

	botRepo := repo.New(cfg.Db)
	notifierStore := store.New()
	botService := service.Init(botRepo, *notifierStore)
	scheduleBot := bot.Init(cfg.Bot.Token, botRepo, botService, notifierStore, cfg.Endpoints)
	slog.Info("Bot started")
	scheduleBot.Listen()
}
