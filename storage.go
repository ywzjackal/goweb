package goweb

import (
	"sync"
	"time"
)

var (
	StorageDefaultLife = time.Minute * 10
)

type Storage interface {
	Get(string) interface{}
	Set(string, interface{})
	SetWithLife(string, interface{}, time.Duration)
	Remove(string)
}

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

type StorageMemory struct {
	Storage
	storage map[string]*storageValueWrap
}

func (s *StorageMemory) Get(key string) interface{} {
	storageValue, ok := s.storage[key]
	if !ok {
		return nil
	}
	storageValue.Lock()
	defer storageValue.Unlock()
	storageValue.timer.Reset(storageValue.life)
	return storageValue.Value
}

func (s *StorageMemory) Set(key string, value interface{}) {
	if s.storage == nil {
		s.storage = make(map[string]*storageValueWrap)
	}
	s.SetWithLife(key, value, StorageDefaultLife)
}

func (s *StorageMemory) SetWithLife(key string, value interface{}, life time.Duration) {
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

func (s *StorageMemory) Remove(key string) {
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
