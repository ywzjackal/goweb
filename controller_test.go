package goweb

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof"
	"strconv"
	"sync"
	"testing"
)

var (
	once       sync.Once
	serverAddr string
	httpserver *httptest.Server
)

type FactoryCounter struct {
	FactoryStandalone
	count int
}

func (f *FactoryCounter) Current() int {
	return f.count
}

type ControllerCounter struct {
	ControllerStandalone
	Fc    *FactoryCounter
	Count int
}

func (f *ControllerCounter) ActionGet() string {
	num := f.Count + f.Fc.Current()
	f.Context().ResponseWriter().Write([]byte(fmt.Sprintf("%d", num)))
	return ""
}

func startWsServer() {
	memStorage := NewStorageMemory()
	factoryContainer := NewFactoryContainer()
	controllerContainer := NewControllerContainer(factoryContainer)
	router := NewRouter(controllerContainer, factoryContainer, memStorage)
	router.FactoryContainer().Register(&FactoryCounter{
		count: 2,
	})
	router.ControllerContainer().Register("/counter", &ControllerCounter{})
	httpserver = httptest.NewServer(nil)
	serverAddr = httpserver.Listener.Addr().String()
	log.Println("goweb server listen on", serverAddr)
	http.Handle("/", router)
	//	serverAddr = "localhost:8080"
	//	go func() {
	//		err := http.ListenAndServe("localhost:8080", nil)
	//		if err != nil {
	//			Log.Fatal(err)
	//		}
	//	}()
}

func Test_Controller(t *testing.T) {
	var (
		num (int64) = 1
	)
	once.Do(startWsServer)
	for i := 0; i < 3; i++ {
		url := "http://" + serverAddr + "/counter?count=" + fmt.Sprintf("%d", num)
		t.Log("Get:", url)
		res, err := http.Get(url)
		if err != nil {
			t.Error(err)
			return
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Error(err)
			return
		}
		n, err := strconv.ParseInt(string(content), 10, 0)
		if err != nil {
			t.Error(err)
			return
		}
		if n != num+2 {
			t.Error(fmt.Errorf("Content Error!%d", n))
		} else {
			num += 1
		}
	}
}
