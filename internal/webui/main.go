package webui

import (
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/viper"

	"github.com/joyrex2001/nightshift/internal/webui/backend"
)

type WebUI struct {
	m    sync.Mutex
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
	enabled := viper.GetBool("web.enable")
	if !enabled {
		return
	}
	go func() {
		addr := viper.GetString("web.listen-addr")
		hndlr := backend.NewHandler()
		srv := http.Server{
			Addr:         addr,
			Handler:      backend.HTTPLogger(hndlr, []string{"/healthz"}),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		}

		cert := viper.GetString("web.cert-file")
		key := viper.GetString("web.key-file")
		tls := viper.GetBool("web.enable-tls")

		glog.Infof("Starting webui on %s...", addr)
		if tls {
			glog.Fatal(srv.ListenAndServeTLS(cert, key))
		} else {
			glog.Fatal(srv.ListenAndServe())
		}
	}()
}

// Stop will stop the webserver.
func (a *WebUI) Stop() {
	a.m.Lock()
	defer a.m.Unlock()
	a.done <- true
}
