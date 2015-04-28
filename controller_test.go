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
	*Factory
	count int
}

func (f *FactoryCounter) Current() int {
	return f.count
}

type ControllerCounter struct {
	*controller
}

type Params struct {
	parameters
	Count int
}

func (f *ControllerCounter) ActionDefault(
	params *Params, fa *FactoryCounter, res http.ResponseWriter) {
	num := fa.Current() + params.Count
	res.Write([]byte(fmt.Sprintf("%d", num)))
}

func startWsServer() {
	router := &router{}
	router.Init()
	router.RegisterController("", &ControllerCounter{})
	router.RegisterFactory(&FactoryCounter{})
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
	for i := 0; i < 1000; i++ {
		url := "http://" + serverAddr + "/counter?Count=" + fmt.Sprintf("%d", num)
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
		if n != num {
			t.Error(fmt.Errorf("Content Error!%d", n))
		} else {
			num += 1
		}
	}
}
