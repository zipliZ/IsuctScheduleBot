package repo

import (
	"ScheduleBot/configs"
	"log"
	"log/slog"
	"time"

	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
)

type Repo interface {
	CreateUser(chatId int64, username string)
	UpdateUserHolder(chatId int64, isStudent bool, newHolder string)
	GetUserInfo(chatId int64) (bool, string)
	GetHistory(chatId int64) []string
	UpdateUserHistory(chatId int64, newGroup string)
	UserExists(chatId int64) bool
	GetUsers() []int64
	UpdateNotificationStatus(chatId int64, status bool)
	GetTop3Donators() []Donator
	IsDailyNotifierOn(chatId int64) bool
	GetUserTimer(chatId int64) string
	GetNotificationOn() []UsersToNotify
	UpdateUserTimer(chatId int64, timer string)
}

func New(cfg configs.DbConfig) *BotRepo {
	database := reindexer.NewReindex("cproto://"+cfg.User+":"+cfg.Pass+"@"+cfg.Host+":"+cfg.Port+"/"+cfg.DbName, reindexer.WithCreateDBIfMissing())
	if err := database.Ping(); err != nil {
		log.Panic(err)
	}
	err := database.OpenNamespace("users", reindexer.DefaultNamespaceOptions(), User{})
	if err != nil {
		log.Panic(err)
	}
	err = database.OpenNamespace("donators", reindexer.DefaultNamespaceOptions(), Donator{})
	if err != nil {
		log.Panic(err)
	}

	return &BotRepo{db: database}
}

func (b *BotRepo) CreateUser(chatId int64, username string) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		slog.Error("Ошибка при установке часового пояса:", err)
		return
	}
	currentTime := time.Now().In(location).Format(time.DateTime)
	if _, err := b.db.Insert("users", &User{
		ChatId:        chatId,
		Username:      username,
		History:       make([]string, 4),
		DailyNotifier: false,
		CreateDate:    currentTime,
	}); err != nil {
		slog.Error("creating user", err)
	}
}

func (b *BotRepo) UpdateUserHolder(chatId int64, isStudent bool, newHolder string) {
	b.UpdateUserHistory(chatId, newHolder)
	b.db.Query("users").Where("chatId", reindexer.EQ, chatId).Set("Holder", newHolder).Set("IsStudent", isStudent).Update()
}

func (b *BotRepo) UpdateUserHistory(chatId int64, newGroup string) {
	result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get()
	if !found {
		return
	}
	oldHolder := result.(*User).Holder
	historyArr := result.(*User).History
	if newGroup == oldHolder {
		return
	}
	for i, holder := range historyArr {
		if holder == newGroup {
			historyArr = append(historyArr[:i], historyArr[i+1:]...)
		}
	}
	historyArr = append([]string{oldHolder}, historyArr[:3]...)

	b.db.Query("users").Where("chatId", reindexer.EQ, chatId).Set("History", historyArr).Update()
}

func (b *BotRepo) GetUserInfo(chatId int64) (bool, string) {
	if result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get(); found {
		return result.(*User).IsStudent, result.(*User).Holder
	}
	return false, ""
}

func (b *BotRepo) GetHistory(chatId int64) []string {
	if result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get(); found {
		return result.(*User).History
	}
	return nil
}

func (b *BotRepo) GetUsers() []int64 {
	iterator := b.db.Query("users").Exec()
	defer iterator.Close()
	var users []int64
	for iterator.Next() {
		users = append(users, iterator.Object().(*User).ChatId)
	}
	return users
}

func (b *BotRepo) UserExists(chatId int64) bool {
	_, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get()
	return found
}

func (b *BotRepo) IsDailyNotifierOn(chatId int64) bool {
	_, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).And().Where("DailyNotifier", reindexer.EQ, true).Get()
	return found
}

func (b *BotRepo) GetUserTimer(chatId int64) string {
	if result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get(); found {
		return result.(*User).Timer
	}
	return ""
}

func (b *BotRepo) UpdateUserTimer(chatId int64, timer string) {
	b.db.Query("users").Where("chatId", reindexer.EQ, chatId).Set("Timer", timer).Update()
}

func (b *BotRepo) GetNotificationOn() []UsersToNotify {
	iterator := b.db.Query("users").Where("DailyNotifier", reindexer.EQ, true).Exec()
	defer iterator.Close()
	var users []UsersToNotify
	for iterator.Next() {
		user := iterator.Object().(*User)
		users = append(users, UsersToNotify{
			ChatId: user.ChatId,
			Time:   user.Timer,
		})
	}
	return users
}

func (b *BotRepo) UpdateNotificationStatus(chatId int64, status bool) {
	b.db.Query("users").Where("chatId", reindexer.EQ, chatId).Set("DailyNotifier", status).Update()
}

func (b *BotRepo) GetTop3Donators() []Donator {
	items, err := b.db.Query("donators").Sort("amountOfDonation", true).Limit(3).Exec().FetchAll()
	if err != nil {
		log.Println(err)
	}
	donators := make([]Donator, 0)
	for _, item := range items {
		donator := item.(*Donator)
		donators = append(donators, *donator)
	}
	return donators
}
