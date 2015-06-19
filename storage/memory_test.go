package storage

import (
	"fmt"
	"testing"
	"time"
)

func Test_StorageMemory(t *testing.T) {
	StorageDefaultLife = time.Second * 1
	storage := &storageMemory{}
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("string_%d", i)
		storage.Set(key, "haha")
		if storage.Get(key) != "haha" {
			t.Error("Fail to restore values!")
		}
	}
	time.Sleep(1 * time.Second)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("string_%d", i)
		if storage.Get(key) != nil {
			t.Error("values doesn't remove when deadline is arrived!")
		}
	}
}
