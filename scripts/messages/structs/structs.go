package structs

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

type MsgLog struct {
	Id          int64     `reindex:"Id,,pk"`
	MessageText string    `reindex:"Message"`
	Messages    []Message `reindex:"Messages"`
	SendErrs    []SendErr `reindex:"SendErrs"`
	SentTime    int64     `reindex:"SentTime"`
}

type Message struct {
	ChatId    int64 `reindex:"ChatId"`
	MessageId int   `reindex:"MessageId"`
}
type SendErr struct {
	ChatId int64  `reindex:"ChatId"`
	Error  string `reindex:"Error"`
}
