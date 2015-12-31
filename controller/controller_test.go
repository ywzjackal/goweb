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
	"math/rand"
	"time"
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
	Controller
	Standalone
	Fc    *FactoryCounter
	Count int
}

func (f *ControllerCounter) ActionGet() {
	num := f.Count + f.Fc.Current()
	f.Context().ResponseWriter().Write([]byte(fmt.Sprintf("%d", num)))
}

type ControllerCounterWithPreActionAndPostAction struct {
	Controller
	Standalone
	goweb.ActionPreprocessor
	goweb.ActionPostprocessor
	Fc         *FactoryCounter
	Count      int
	beforeIsOk bool
	afterIsOk  bool
}

func (f *ControllerCounterWithPreActionAndPostAction) BeforeAction() bool {
	f.beforeIsOk = true
	f.Count++
	return true
}

func (f *ControllerCounterWithPreActionAndPostAction) ActionGet() {
	num := f.Count + f.Fc.Current()
	f.Context().ResponseWriter().Write([]byte(fmt.Sprintf("%d", num)))
}

func (f *ControllerCounterWithPreActionAndPostAction) AfterAction() {
	f.afterIsOk = true
	f.Count++
}

func startWsServer() {
	memStorage := storage.NewStorageMemory()
	factoryContainer := factory.NewContainer()
	controllerContainer := NewContainer(factoryContainer)
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
	}, "counter")
	router.ControllerContainer().Register("/counter", &ControllerCounter{})
	router.ControllerContainer().Register("/counterprepost", &ControllerCounterWithPreActionAndPostAction{})
	httpServer = httptest.NewServer(nil)
	serverAddr = httpServer.Listener.Addr().String()
	log.Println("goweb server listen on", serverAddr)
	http.Handle("/", router)
}

func Test_Controller(t *testing.T) {
	var (
		num (int64) = 1
	)
	once.Do(startWsServer)
	for i := 0; i < 10; i++ {
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

func Test_ControllerPrePost(t *testing.T) {
	var (
		num (int64) = 1
	)
	once.Do(startWsServer)
	for i := 0; i < 30; i++ {
		url := "http://" + serverAddr + "/counterprepost"
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
			num += 2
		}
	}
}

func Test_ControllerDataRace(t *testing.T) {
	once.Do(startWsServer)
	wg := sync.WaitGroup{}
	wg.Add(10000)
	begin := time.Now()
	for i := 0; i < 10000; i++ {
		go func() {
			defer wg.Done()
			num := int64(rand.Int())
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
			if n != num + 2 {
				t.Error(fmt.Errorf("Content Error!%d", n))
			} else {
				num += 1
			}
		}()
	}
	wg.Wait()

	esplace := time.Since(begin)
	t.Logf("race useage %s", esplace)
}
