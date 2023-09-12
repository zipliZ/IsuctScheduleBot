package repo

type User struct {
	ChatId     int64  `reindex:"ChatId,,pk"`
	Username   string `reindex:"Username"`
	Group      string `reindex:"Group"`
	CreateDate string `reindex:"CreateDate"`
}
