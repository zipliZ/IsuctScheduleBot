package bot

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *ScheduleBot) handleCallback(ctx context.Context, callbackQuery *tgbotapi.CallbackQuery) (tgbotapi.MessageConfig, error) {
	chatID := callbackQuery.Message.Chat.ID
	callbackData := callbackQuery.Data

	callback := tgbotapi.NewCallback(callbackQuery.ID, callbackQuery.Data)
	msg := NewMessage(chatID, "", false)

	reGroup := regexp.MustCompile(`^\d-\d{1,3}$`)
	reTeacher := regexp.MustCompile(`^[А-ЯЁ][а-яё]+\s[А-ЯЁ]\.[А-ЯЁ]\.$`)

	var err error
	var weakDay int

	switch {
	case checkWeekDay(strings.ToLower(callbackData), &weakDay):
		if msg.Text, err = b.getScheduleByWeekDay(ctx, chatID, weakDay); err != nil {
			msg.Text = formServerErr()
		}

	case reGroup.MatchString(callbackData):
		b.repo.UpdateUserHolder(chatID, true, callbackData)

		deleteCfg := tgbotapi.NewDeleteMessage(chatID, callbackQuery.Message.MessageID)
		_, err = b.bot.Request(deleteCfg)

		msg.Text = "Измененно"
		b.buttons.standard.Keyboard[1][1].Text = fmt.Sprintf("Сменить (%s)", callbackData)
		msg.ReplyMarkup = b.buttons.standard

	case reTeacher.MatchString(callbackData):
		b.repo.UpdateUserHolder(chatID, false, callbackData)

		deleteCfg := tgbotapi.NewDeleteMessage(chatID, callbackQuery.Message.MessageID)
		_, err = b.bot.Request(deleteCfg)

		msg.Text = "Измененно"
		b.buttons.standard.Keyboard[1][1].Text = fmt.Sprintf("Сменить (%s)", callbackData)
		msg.ReplyMarkup = b.buttons.standard
	}

	go func() {
		_, err := b.bot.Request(callback)
		if err != nil {
			slog.Error("callback request err:", err)
		}
	}()

	return msg, err
}
