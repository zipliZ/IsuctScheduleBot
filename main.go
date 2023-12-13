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

	botRepo := repo.NewBotRepo(cfg.Db)
	notifierStore := store.NewNotifierStore()
	botService := service.NewBotService(botRepo, *notifierStore)
	scheduleBot := bot.NewScheduleBot(cfg.Bot.Token, botRepo, botService, notifierStore, cfg.Endpoints)
	botService.RestoreNotifications()
	slog.Info("Bot started")
	go scheduleBot.NotifyUsers()
	scheduleBot.Listen()
}
