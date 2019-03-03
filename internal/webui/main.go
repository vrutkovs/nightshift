package webui

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/joyrex2001/nightshift/internal/webui/backend"
)

type WebUI struct {
	Addr string
	TLS  bool
	Cert string
	Key  string

	m    sync.Mutex
	srv  *http.Server
	done chan bool
}

var instance *WebUI
var once sync.Once

// New will instantiate a new WebUI object.
func New() *WebUI {
	once.Do(func() {
		instance = &WebUI{
			done: make(chan bool),
		}
	})
	return instance
}

// Start will start the webserver.
func (a *WebUI) Start() {
	go func() {
		hndlr := backend.NewHandler()
		a.srv = &http.Server{
			Addr:         a.Addr,
			Handler:      backend.HTTPLogger(hndlr, []string{"/healthz"}),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		}
		glog.Infof("Starting webui on %s...", a.Addr)
		if a.TLS {
			glog.Fatal(a.srv.ListenAndServeTLS(a.Cert, a.Key))
		} else {
			glog.Fatal(a.srv.ListenAndServe())
		}
	}()
}

// Stop will stop the webserver.
func (a *WebUI) Stop() error {
	a.m.Lock()
	defer a.m.Unlock()

	if err := a.srv.Shutdown(context.TODO()); err != nil {
		return err
	}

	a.done <- true
	return nil
}
