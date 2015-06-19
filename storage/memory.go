package storage

import (
	"sync"
	"time"

	"github.com/ywzjackal/goweb"
)

var (
	StorageDefaultLife = time.Minute * 10
)

type storageValueWrap struct {
	mutex sync.Mutex
	Value interface{}
	timer *time.Timer
	life  time.Duration
}

func (s *storageValueWrap) Lock() {
	s.mutex.Lock()
}

func (s *storageValueWrap) Unlock() {
	s.mutex.Unlock()
}

type storageMemory struct {
	goweb.Storage
	storage map[string]*storageValueWrap
}

func NewStorageMemory() goweb.Storage {
	return &storageMemory{
		storage: make(map[string]*storageValueWrap),
	}
}

func (s *storageMemory) Get(key string) interface{} {
	storageValue, ok := s.storage[key]
	if !ok {
		return nil
	}
	storageValue.Lock()
	defer storageValue.Unlock()
	storageValue.timer.Reset(storageValue.life)
	return storageValue.Value
}

func (s *storageMemory) Set(key string, value interface{}) {
	if s.storage == nil {
		s.storage = make(map[string]*storageValueWrap)
	}
	s.SetWithLife(key, value, StorageDefaultLife)
}

func (s *storageMemory) SetWithLife(key string, value interface{}, life time.Duration) {
	if s.storage == nil {
		s.storage = make(map[string]*storageValueWrap)
	}
	_, ok := s.storage[key]
	if ok {
		s.Remove(key)
	}
	s.storage[key] = &storageValueWrap{
		Value: value,
		timer: time.AfterFunc(StorageDefaultLife, func() {
			s.Remove(key)
		}),
		life: life,
	}
}

func (s *storageMemory) Remove(key string) {
	if s.storage == nil {
		s.storage = make(map[string]*storageValueWrap)
	}
	storageValue, ok := s.storage[key]
	if ok {
		storageValue.Lock()
		defer storageValue.Unlock()
		storageValue.timer.Stop()
		delete(s.storage, key)
	}
}
