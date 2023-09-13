package repo

import (
	"ScheduleBot/configs"
	"log"
	"time"

	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
)

type Repo interface {
	CreateUser(chatId int64, group string)
	UpdateUser(chatId int64, newGroup string)
	GetGroup(chatId int64) string
	GetGroupHistory(chatId int64) []string
	UpdateUserGroupHistory(chatId int64, newGroup string)
	UserExists(chatId int64) bool
	GetUsers(chatId int64)
}

type BotRepo struct {
	db *reindexer.Reindexer
}

func NewBotRepo(cfg configs.DbConfig) *BotRepo {
	database := reindexer.NewReindex("cproto://"+cfg.User+":"+cfg.Pass+"@"+cfg.Host+":"+cfg.Port+"/"+cfg.DbName, reindexer.WithCreateDBIfMissing())
	if err := database.Ping(); err != nil {
		panic(err)
	}
	err := database.OpenNamespace("users", reindexer.DefaultNamespaceOptions(), User{})
	if err != nil {
		panic(err)
	}

	return &BotRepo{db: database}
}

func (b *BotRepo) CreateUser(chatId int64, username string) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Println("Ошибка при установке часового пояса:", err)
		return
	}
	currentTime := time.Now().In(location).Format(time.DateTime)
	if _, err := b.db.Insert("users", &User{
		ChatId:       chatId,
		Username:     username,
		GroupHistory: make([]string, 4),
		CreateDate:   currentTime,
	}); err != nil {
		log.Println(err)
	}
}

func (b *BotRepo) UpdateUserGroup(chatId int64, newGroup string) {
	b.UpdateUserGroupHistory(chatId, newGroup)
	b.db.Query("users").Where("chatId", reindexer.EQ, chatId).Set("Group", newGroup).Update()
}

func (b *BotRepo) UpdateUserGroupHistory(chatId int64, newGroup string) {
	result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get()
	if !found {
		return
	}
	oldGroup := result.(*User).Group
	groupsArr := result.(*User).GroupHistory
	if newGroup == oldGroup {
		return
	}
	for i, group := range groupsArr {
		if group == newGroup {
			groupsArr = append(groupsArr[:i], groupsArr[i+1:]...)
		}
	}
	groupsArr = append([]string{oldGroup}, groupsArr[:3]...)

	b.db.Query("users").Where("chatId", reindexer.EQ, chatId).Set("GroupHistory", groupsArr).Update()
}

func (b *BotRepo) GetGroup(chatId int64) string {
	if result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get(); found {
		return result.(*User).Group
	}
	return ""
}

func (b *BotRepo) GetGroupHistory(chatId int64) []string {
	if result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get(); found {
		return result.(*User).GroupHistory
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
