package server

import (
	"encoding/json"
	"fmt"

	"github.com/bmizerany/pat"
	//"io/ioutil"
	//"log"
	//"mime"
	"net"
	"net/http"
	"strings"
	"time"
)

type RESTfulAPI struct {
	router *pat.PatternServeMux
}

// Constructor
func NewRESTfulAPI() *RESTfulAPI {
	return &RESTfulAPI{
		router: pat.New(),
	}
}

func (self *RESTfulAPI) Start(addr, staticDir string) {

	//self.mountResources()
	self.router.Get("/", self.indexHandler())
	self.router.Get("/static/", self.staticHandler(staticDir))

	// Mount router to server
	serverMux := http.NewServeMux()
	serverMux.Handle("/", self.router)

	s := &http.Server{
		Handler:        serverMux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("Starting RESTful server at http://%v\n", addr)

	err = s.Serve(ln)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func (self *RESTfulAPI) indexHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		b, _ := json.Marshal("Welcome to Cascades Server RESTful API")
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(b)
	}
}

func (self *RESTfulAPI) staticHandler(staticDir string) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		filePath := strings.Join(strings.Split(req.URL.Path, "/")[2:], "/")
		http.ServeFile(rw, req, staticDir+"/"+filePath)
	}
}
