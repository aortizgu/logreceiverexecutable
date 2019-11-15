package main

import (
	"encoding/json"
	"log"
	"logreceiver"
	"net/http"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
)

const (
	serviceName       string = "log"
	serviceSyslogType string = "_syslog._udp"
	serviceHTTPType   string = "_http._tcp"
	serviceDomain     string = "local."
	servicePort       int    = 514
	dbPath            string = "db/syslog.db"
	httpPort          int    = 8081
	cleanPeriodMs     int    = 1000 /*millis*/ * 60 /*seconds*/ * 1 /*minutes*/
	maxLogs           int    = 500                                  // MaxLogs : MaxLogs to store in db
)

//Web handlers:
func handleSearch(l *logreceiver.LogReceiver, w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params := r.URL.Query()
	device := params.Get("device")
	hostname := params.Get("hostname")
	severity, _ := strconv.ParseInt(params.Get("severity"), 10, 64)
	day := params.Get("day")
	maxEntries, _ := strconv.ParseInt(params.Get("max"), 10, 64)
	offsetEntries, _ := strconv.ParseInt(params.Get("offset"), 10, 64)
	logs := l.Search(device, hostname, day, severity, maxEntries, offsetEntries)
	logsJSON, err := json.Marshal(logs)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(logsJSON)
}

func handleInfo(l *logreceiver.LogReceiver, w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info := l.GetInfo()
	infoJSON, err := json.Marshal(info)
	if err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(infoJSON)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "static/indexhome.html")
}

func serveWs(logReceiver *logreceiver.LogReceiver, w http.ResponseWriter, r *http.Request) {
	conn, err := logreceiver.ClientUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	logreceiver.NewClient(logReceiver, conn)
}

// main
func main() {
	l := logreceiver.NewLogReceiver(serviceName, serviceSyslogType, serviceDomain, dbPath, servicePort, cleanPeriodMs, maxLogs)
	l.Start()
	_, err := zeroconf.Register(serviceName, serviceHTTPType, serviceDomain, httpPort, nil, nil)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(l, w, r)
	})
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		handleInfo(l, w, r)
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(l, w, r)
	})
	http.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("panic called")
	})
	http.HandleFunc("/nopanic", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				log.Println(x)
				log.Println("panic handled")
			}
		}()
		panic("panic called")
	})
	http.Handle("/", http.StripPrefix(strings.TrimRight("/", "/"), http.FileServer(http.Dir("./static"))))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(httpPort), nil))
}
