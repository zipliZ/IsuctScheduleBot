package service

import (
	r "ScheduleBot/internal/repo"
	s "ScheduleBot/internal/store"
	"fmt"
	"log/slog"
)

type Service interface {
	ToggleNotification(chatId int64) string
	UpdateTimer(chatId int64, newTime string) string
	RestoreNotifications()
}

func NewBotService(repo r.Repo, store s.NotifierStore) *BotService {
	return &BotService{repo: repo, store: store}
}

func (s *BotService) ToggleNotification(chatId int64) string {
	userTimer := s.repo.GetUserTimer(chatId)
	if s.repo.IsDailyNotifierOn(chatId) {
		s.repo.UpdateNotificationStatus(chatId, false)
		s.store.StoreDeleteUser(userTimer, chatId)
		return "Получение ежедневного расписания выключено"
	} else {
		if userTimer == "" {
			userTimer = "04:20"
			s.repo.UpdateUserTimer(chatId, userTimer)
			s.store.StoreUser(userTimer, chatId)
		} else {
			s.store.StoreUser(userTimer, chatId)
		}
		s.repo.UpdateNotificationStatus(chatId, true)
		return fmt.Sprintf("Ежедневное расписание будет приходить в %s", userTimer)
	}
}

func (s *BotService) UpdateTimer(chatId int64, newTime string) string {
	oldTimer := s.repo.GetUserTimer(chatId)
	s.repo.UpdateNotificationStatus(chatId, true)
	s.repo.UpdateUserTimer(chatId, newTime)
	s.store.StoreUpdateUser(oldTimer, newTime, chatId)
	return fmt.Sprintf("Время оповещения установленно на %s", newTime)
}

func (s *BotService) RestoreNotifications() {
	users := s.repo.GetNotificationOn()
	for _, user := range users {
		s.store.StoreUser(user.Time, user.ChatId)
	}
	slog.Info("Уведомления востановленны")
}
