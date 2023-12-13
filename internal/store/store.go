package store

import "sync"

func NewNotifierStore() *NotifierStore {
	return &NotifierStore{data: make(map[string][]int64), mutex: &sync.RWMutex{}}
}

func (n *NotifierStore) Get(key string) ([]int64, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	value, exist := n.data[key]
	if !exist {
		return nil, false
	}
	return value, true
}

func (n *NotifierStore) StoreUser(key string, user int64) {
	users, exist := n.Get(key)

	n.mutex.Lock()
	defer n.mutex.Unlock()

	if !exist {
		n.data[key] = []int64{user}
	}
	n.data[key] = append(users, user)
}

func (n *NotifierStore) StoreDeleteUser(key string, user int64) {
	users, exist := n.Get(key)
	if !exist {
		return
	}

	n.mutex.Lock()
	defer n.mutex.Unlock()

	for i, val := range users {
		if val == user {
			n.data[key] = append(users[:i], users[i+1:]...)
		}
	}

	if len(n.data[key]) == 0 {
		delete(n.data, key)
	}
}

func (n *NotifierStore) StoreUpdateUser(oldKey, newKey string, user int64) {
	n.StoreDeleteUser(oldKey, user)
	n.StoreUser(newKey, user)
}
