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
	text := `

Текст сообщения

`
	silent := true

	cfg := configs.DecodeConfig("./../../config.yaml")
	db := reindexer.NewReindex("cproto://"+cfg.Db.User+":"+cfg.Db.Pass+"@"+cfg.Db.Host+":"+cfg.Db.Port+"/"+cfg.Db.DbName, reindexer.WithCreateDBIfMissing())
	if err := db.Ping(); err != nil {
		panic(err)
	}
	err := db.OpenNamespace("users", reindexer.DefaultNamespaceOptions(), structs.User{})
	if err != nil {
		panic(err)
	}
	err = db.OpenNamespace("sent_msg_logs", reindexer.DefaultNamespaceOptions(), structs.MsgLog{})
	if err != nil {
		panic(err)
	}
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic(err)
	}

	iterator := db.Query("users").Exec()
	defer iterator.Close()

	var users []int64

	for iterator.Next() {
		user := iterator.Object().(*structs.User).ChatId
		users = append(users, user)
	}

	msgLog := structs.MsgLog{MessageText: text}
	messages := make([]structs.Message, 0)
	sendErrors := make([]structs.SendErr, 0)
	ticker := time.NewTicker(time.Second / 30)
	wg := sync.WaitGroup{}
	mutex := sync.Mutex{}

	for _, user := range users {
		wg.Add(1)
		select {
		case <-ticker.C:
			go func(user int64, messages *[]structs.Message, sendErrors *[]structs.SendErr) {
				defer wg.Done()
				msg := tgbotapi.NewMessage(user, text)
				msg.DisableNotification = silent
				msg.ParseMode = "MarkdownV2"

				messageId, err := bot.Send(msg)
				mutex.Lock()
				defer mutex.Unlock()
				if err != nil {
					*sendErrors = append(*sendErrors, structs.SendErr{
						ChatId: user,
						Error:  err.Error(),
					})
					slog.Error("sending err:", err, "chat_id", user)
					return
				}
				*messages = append(*messages, structs.Message{
					ChatId:    user,
					MessageId: messageId.MessageID,
				})
			}(user, &messages, &sendErrors)
		}
	}

	wg.Wait()
	msgLog.Messages = messages
	msgLog.SendErrs = sendErrors
	msgLog.SentTime = time.Now().Unix()

	_, insertErr := db.Insert("sent_msg_logs", msgLog, "id=serial()")
	if insertErr != nil {
		panic(insertErr)
	}
	fmt.Println("Сообщения отправленны")

}
