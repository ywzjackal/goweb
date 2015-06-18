package goweb

import (
	"sync"
	"time"
)

var (
	StorageDefaultLife = time.Minute * 10
)

type Storage interface {
	// Init() called by framework before used.
	Init() WebError
	// Get() return the element(interface{}) find by key,
	// return nil if not found with the key
	Get(string) interface{}
	// Set() element(interface{}) with it's key,
	// and data will removed after the default duration from last query
	Set(string, interface{})
	// Set() element(interface{}) with life.
	// data will be removed after the duration from last query
	SetWithLife(string, interface{}, time.Duration)
	// Remove() element before deadline.
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

type storageMemory struct {
	Storage
	storage map[string]*storageValueWrap
}

func NewStorageMemory() Storage {
	return &storageMemory{
		storage: make(map[string]*storageValueWrap),
	}
}

func (s *storageMemory) Init() WebError {
	s.storage = make(map[string]*storageValueWrap)
	return nil
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
