# GOWEB

"GOWEB" is a light but powerful web framework for golang developer

# Feature
### 1)MVC design

    http request ->  http handle  -> goweb.Router -> goweb.Controller ->    Action  -> http response 
                                                        [M]                   [C]         [V]
                  (golang builtin)  (goweb buildin)  (goweb buildin)    (user define)  
                  
### 2)Injectable

GOWEB support inject factory(like EJB in J2EE) to controller, or inject factory to factory.

    http request ->  http handle  -> goweb.Router -> goweb.Controller ->    Action  -> http response 
                                                        [M]                   [C]         [V]
                                                         ^
                                                         ^ (Injection) 
                                          (Injection)    ^
                          goweb.Factory(1)   > >    goweb.Factory(2)
                                                        [F]

### 3)Stateful, Stateless or Standalone controller and factory design

* Stateful Factory and Controller instance are created at first use in session, and destroy with session destroy.
* Stateless Factory and Controller instance are created by request every time, and destroy after response.
* Standalone Factory and Controller instance are created at application startup, and destroy with application destroy.

# Hello Word for Stateful, Stateless and Standalone

* Build and run code:
```
    package main
    import (
    	"github.com/ywzjackal/goweb"
    	"github.com/ywzjackal/goweb/context"
    	"github.com/ywzjackal/goweb/controller"
    	"github.com/ywzjackal/goweb/factory"
    	"github.com/ywzjackal/goweb/router"
    	"github.com/ywzjackal/goweb/session"
    	"github.com/ywzjackal/goweb/storage"
    	"net/http"
    	"fmt"
    )
    //...Setup basic configuration begin...
    var (
    	FactoryContainer    = factory.NewContainer()
    	ControllerContainer = controller.NewContainer(FactoryContainer)
    	// root router of webservice
    	Router = router.NewRouter(
    		ControllerContainer,
    		contextGenerator,
    	)
    )
    // ContextGenerator is an function that return an instance of goweb.Context
    func contextGenerator(res http.ResponseWriter, req *http.Request) goweb.Context {
    	return context.NewContext(
    		res,
    		req,
    		FactoryContainer,
    		session.NewSession(
    			res,
    			req,
    			storage.NewStorageMemory()),
    	)
    }
    //```Setup basic configuration finish```
    //...Setup Hello Word Controller begin...
    type index struct {
    	controller.Controller
    	controller.Standalone
    	Message string
    	Counter int
    }
    func (i *index) Action() {
    	msg := fmt.Sprintf("message:%s, counter:%d", i.Message, i.Counter)
    	i.Context().ResponseWriter().Write([]byte(msg))
    	i.Counter++
    }
    func init() {
    	ControllerContainer.Register("/", &index{})
    }
    //```Setup Hello Word Controller finish```
    func main() {
    	http.Handle("/", Router)
    	er := http.ListenAndServe(":8080", nil)
    	if er != nil {
    		panic(er)
    	}
    }
```
    
* Open browser and type url "localhost:8080/?message=hello_word_from_goweb",will return: "message:hello_word_from_goweb,counter:0"
and counter will increase when you refresh browser.
* Restart browser and retype url above to see what happened with counter. yes, its standalone counter.
* Change index type to Stateful Controller, rebuild and run :
```
    ...
    type index struct {
    	controller.Controller
    	controller.Stateful // changed this line
    	Message string
    	Counter int
    }
    ...
```
* Refresh browser and the counter will increase too. but if you restart browser and reopen url, you will see counter increase from `0`
 not like Standalone Controller, a new Stateful Controller will be created when session begin.
* Change index type to Stateless Controller, rebuild and run :
```
    ...
    type index struct {
    	controller.Controller
    	controller.Stateless // changed this line
    	Message string
    	Counter int
    }
    ...
``` 

* the counter will always be `0`, not increase. because Stateless Controller will always be created when request arrived.

#####Note: 

* Basic configuration only need setup once in whole application, it told Router which ControllerContainer will be used and 
how to generate an context.
* See more example from <https://github.com/ywzjackal/goweb_example> /src/goweb/example/helloword/controller/..

#Hello Word for Injection
