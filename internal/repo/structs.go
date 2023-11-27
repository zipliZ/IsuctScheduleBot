package repo

type User struct {
	ChatId        int64    `reindex:"ChatId,,pk"`
	Username      string   `reindex:"Username"`
	Group         string   `reindex:"Group"`
	GroupHistory  []string `reindex:"GroupHistory"`
	DailyNotifier bool     `reindex:"DailyNotifier"`
	CreateDate    string   `reindex:"CreateDate"`
}

type Donator struct {
	Id               int    `reindex:"id,,pk"`
	Name             string `reindex:"name"`
	AmountOfDonation int    `reindex:"amountOfDonation"`
}
