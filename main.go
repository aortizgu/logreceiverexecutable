package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/aortizgu/logreceiver"

	"github.com/grandcat/zeroconf"
)

const (
	serviceName     string = "log"
	serviceType     string = "_syslog._udp"
	serviceHTTPType string = "_http._tcp"
	serviceDomain   string = "local."
	servicePort     int    = 514
	httpPort        int    = 8081
	cleanPeriodMs   int    = 1000 /*millis*/ * 60 /*seconds*/ * 1 /*minutes*/
)

//Web handlers:
func handleSearch(l *logreceiver.LogReceiver, w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	params := r.URL.Query()
	app := params.Get("app")
	hostname := params.Get("hostname")
	severity, err := strconv.ParseInt(params.Get("severity"), 10, 64)
	if err != nil {
		severity = 8
	}
	from, _ := strconv.ParseInt(params.Get("from"), 10, 64)
	to, _ := strconv.ParseInt(params.Get("to"), 10, 64)
	maxEntries, _ := strconv.ParseInt(params.Get("max"), 10, 64)
	offsetEntries, _ := strconv.ParseInt(params.Get("offset"), 10, 64)
	logs := l.Search(app, hostname, from, to, severity, maxEntries, offsetEntries)
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
	interfaceName := flag.String("iface", "en0", "interface name")
	dbPath := flag.String("db", "/tmp/syslog.db", "db filepath")
	maxLogs := flag.Int("maxlogs", 500000, "max log entries")
	flag.Parse()
	var iface, err = net.InterfaceByName(*interfaceName)
	if err == nil {
		log.Println("selected " + iface.Name)
		l := logreceiver.NewLogReceiver(serviceName, serviceType, serviceDomain, *dbPath, servicePort, cleanPeriodMs, *maxLogs, *iface)
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
		log.Println("open http://localhost:" + strconv.Itoa(httpPort))
		log.Fatal(http.ListenAndServe("0.0.0.0:"+strconv.Itoa(httpPort), nil))
	} else {
		log.Fatal("wrong interface")
	}
}
