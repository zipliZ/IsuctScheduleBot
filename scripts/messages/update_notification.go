package main

import (
	"ScheduleBot/configs"
	"ScheduleBot/scripts/messages/structs"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
	"log"
	"log/slog"
	"sync"
	"time"
)

func main() {
	var LogMsgId int64 = 12
	UpdatedText := `
Обновленный текст

`
	cfg := configs.DecodeConfig("./../../config.yaml")

	db := reindexer.NewReindex("cproto://"+cfg.Db.User+":"+cfg.Db.Pass+"@"+cfg.Db.Host+":"+cfg.Db.Port+"/"+cfg.Db.DbName, reindexer.WithCreateDBIfMissing())
	if err := db.Ping(); err != nil {
		panic(err)
	}
	err := db.OpenNamespace("sent_msg_logs", reindexer.DefaultNamespaceOptions(), structs.MsgLog{})
	if err != nil {
		panic(err)
	}
	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		log.Panic(err)
	}

	item, found := db.Query("sent_msg_logs").Where("id", reindexer.EQ, LogMsgId).Get()
	if !found {
		fmt.Println("Не найдено сообщение с Id: ", LogMsgId)
		return
	}

	messages := item.(*structs.MsgLog).Messages
	sendErrors := item.(*structs.MsgLog).SendErrs
	msgLog := structs.MsgLog{Id: LogMsgId, MessageText: UpdatedText}
	ticker := time.NewTicker(time.Second / 30)
	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	for _, message := range messages {
		wg.Add(1)
		select {
		case <-ticker.C:
			go func(message structs.Message, sendErrors *[]structs.SendErr) {
				defer wg.Done()
				msg := tgbotapi.NewEditMessageText(message.ChatId, message.MessageId, UpdatedText)
				msg.ParseMode = "MarkdownV2"
				_, err := bot.Send(msg)
				if err != nil {
					mutex.Lock()
					*sendErrors = append(*sendErrors, structs.SendErr{
						ChatId: message.ChatId,
						Error:  err.Error(),
					})
					mutex.Unlock()
					slog.Error("sending err:", err, "chat_id", message.ChatId)
					return
				}
			}(message, &sendErrors)
		}
	}
	wg.Wait()
	msgLog.Messages = messages
	msgLog.SendErrs = sendErrors
	msgLog.SentTime = time.Now().Unix()

	_, insertErr := db.Update("sent_msg_logs", msgLog)
	if insertErr != nil {
		panic(insertErr)
	}
	fmt.Println("Сообщения обновленны")
}
