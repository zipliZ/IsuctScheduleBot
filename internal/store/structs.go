package store

import "sync"

type TargetTime string

type NotifierStore struct {
	mutex *sync.RWMutex
	// мапа с данными для оповещения пользователей
	data map[TargetTime][]int64
}
