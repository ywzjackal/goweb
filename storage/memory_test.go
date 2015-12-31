package storage

import (
	"fmt"
	"testing"
	"time"
	"sync"
)

func Test_StorageMemory(t *testing.T) {
	var wg sync.WaitGroup
	StorageDefaultLife = time.Second * 1
	storage := NewStorageMemory()
	m1 := sync.Mutex{}
	m1.Lock()
	wg.Add(100)
	for i := 0; i < 100; i++ {
		m1.Unlock()
		go func() {
			defer wg.Done()
			m1.Lock()
			key := fmt.Sprintf("string_%d", i)
			m1.Unlock()
			storage.Set(key, "haha")
			if storage.Get(key) != "haha" {
				t.Error("Fail to restore values!")
			}
		}()
		m1.Lock()
	}
	m1.Unlock()
	m2 := sync.Mutex{}
	m2.Lock()
	wg.Add(100)
	time.Sleep(2 * time.Second)
	for i := 0; i < 100; i++ {
		m2.Unlock()
		go func() {
			defer wg.Done()
			m2.Lock()
			key := fmt.Sprintf("string_%d", i)
			m2.Unlock()
			if storage.Get(key) != nil {
				t.Error("values doesn't remove when deadline is arrived!")
			}
		}()
		m2.Lock()
	}
	m2.Unlock()
	wg.Wait()
}
