package repo

type User struct {
	ChatId int64  `reindex:"ChatId,,pk"`
	Group  string `reindex:"Group"`
}
