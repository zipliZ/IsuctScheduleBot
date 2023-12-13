package store

import "sync"

type NotifierStore struct {
	mutex *sync.RWMutex
	data  map[string][]int64
}
