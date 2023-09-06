package repo

import (
	"ScheduleBot/configs"
	"github.com/restream/reindexer/v3"
	_ "github.com/restream/reindexer/v3/bindings/cproto"
)

type Repo interface {
	CreateUser(chatId int64, group string)
	UpdateUser(chatId int64, newGroup string)
	GetGroup(chatId int64) string
	UserExists(chatId int64) bool
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

func (b *BotRepo) CreateUser(chatId int64) {
	if _, err := b.db.Insert("users", &User{
		ChatId: chatId,
	}); err != nil {
		panic(err)
	}
}

func (b *BotRepo) UpdateUser(chatId int64, newGroup string) {
	if _, err := b.db.Update("users", &User{
		ChatId: chatId,
		Group:  newGroup,
	}); err != nil {
		panic(err)
	}
}

func (b *BotRepo) GetGroup(chatId int64) string {
	if result, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get(); found {
		return result.(*User).Group
	}
	return ""
}

func (b *BotRepo) UserExists(chatId int64) bool {
	_, found := b.db.Query("users").Where("ChatId", reindexer.EQ, chatId).Get()
	return found
}
