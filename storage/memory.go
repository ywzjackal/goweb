package storage

import (
	"time"

	"github.com/ywzjackal/goweb"
	"sync"
)

var (
	StorageDefaultLife = time.Minute * 30
)

type storageValueWrap struct {
	value interface{}
	last  time.Time
	life  time.Duration
}

func (s *storageValueWrap)IsExpired() bool {
	return s.life < time.Since(s.last)
}

func (s *storageValueWrap)Refresh() {
	s.last = time.Now()
}

type storageMemory struct {
	goweb.Storage
	sync.Mutex
	storage map[string]storageValueWrap
}

func NewStorageMemory() goweb.Storage {
	return &storageMemory{
		storage: make(map[string]storageValueWrap),
	}
}

func (s *storageMemory) Get(key string) interface{} {
	s.Lock()
	defer s.Unlock()
	storageValue, ok := s.storage[key]
	if !ok {
		return nil
	}
	if(storageValue.IsExpired()){
		s.Remove(key)
		return nil
	}
	storageValue.Refresh()
	return storageValue.value
}

func (s *storageMemory) Set(key string, value interface{}) {
	s.SetWithLife(key, value, StorageDefaultLife)
}

func (s *storageMemory) SetWithLife(key string, value interface{}, life time.Duration) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.storage[key]
	if ok {
		s.Remove(key)
	}
	s.storage[key] = storageValueWrap{
		value: value,
		last:  time.Now(),
		life:  life,
	}
}

func (s *storageMemory) Remove(key string) {
	_, ok := s.storage[key]
	if ok {
		delete(s.storage, key)
	}
}
