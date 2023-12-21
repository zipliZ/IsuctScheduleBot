package repo

import (
	"github.com/restream/reindexer/v3"
)

type BotRepo struct {
	db *reindexer.Reindexer
}

type User struct {
	ChatId        int64    `reindex:"ChatId,,pk"`
	Username      string   `reindex:"Username"`
	IsStudent     bool     `reindex:"IsStudent"`
	Holder        string   `reindex:"Holder"`
	History       []string `reindex:"History"`
	DailyNotifier bool     `reindex:"DailyNotifier"`
	Timer         string   `reindex:"Timer"`
	CreateDate    string   `reindex:"CreateDate"`
}

type Donator struct {
	Id               int    `reindex:"id,,pk"`
	Name             string `reindex:"name"`
	AmountOfDonation int    `reindex:"amountOfDonation"`
}

type UsersToNotify struct {
	ChatId int64
	Time   string
}
