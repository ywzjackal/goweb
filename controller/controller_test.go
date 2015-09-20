package controller

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

	"github.com/ywzjackal/goweb"
	ctx "github.com/ywzjackal/goweb/context"
	"github.com/ywzjackal/goweb/factory"
	"github.com/ywzjackal/goweb/router"
	"github.com/ywzjackal/goweb/session"
	"github.com/ywzjackal/goweb/storage"
)

var (
	once       sync.Once
	serverAddr string
	httpServer *httptest.Server
)

type FactoryCounter struct {
	factory.FactoryStandalone
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

func (f *ControllerCounter) ActionGet() {
	num := f.Count + f.Fc.Current()
	f.Context().ResponseWriter().Write([]byte(fmt.Sprintf("%d", num)))
}

func (f ControllerCounter) ActionPost() {

}

func (f ControllerCounter) ActionDelete() {

}

func startWsServer() {
	memStorage := storage.NewStorageMemory()
	factoryContainer := factory.NewFactoryContainer()
	controllerContainer := NewControllerContainer(factoryContainer)
	router := router.NewRouter(
		controllerContainer,
		func(res http.ResponseWriter, req *http.Request) goweb.Context {
			return ctx.NewContext(
				res,
				req,
				factoryContainer,
				session.NewSession(res, req, memStorage),
			)
		},
	)
	factoryContainer.Register(&FactoryCounter{
		count: 2,
	})
	router.ControllerContainer().Register("/counter", &ControllerCounter{})
	httpServer = httptest.NewServer(nil)
	serverAddr = httpServer.Listener.Addr().String()
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
