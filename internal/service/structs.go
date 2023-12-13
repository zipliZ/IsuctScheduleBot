package service

import (
	"ScheduleBot/internal/repo"
	"ScheduleBot/internal/store"
)

type BotService struct {
	repo  repo.Repo
	store store.NotifierStore
}
