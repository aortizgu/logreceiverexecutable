package main

import (
	"encoding/json"
	"fmt"
	"log"
	"logreceiver"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

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

func getAdapterList() (*syscall.IpAdapterInfo, error) {
	b := make([]byte, 1000)
	l := uint32(len(b))
	a := (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
	// TODO(mikio): GetAdaptersInfo returns IP_ADAPTER_INFO that
	// contains IPv4 address list only. We should use another API
	// for fetching IPv6 stuff from the kernel.
	err := syscall.GetAdaptersInfo(a, &l)
	if err == syscall.ERROR_BUFFER_OVERFLOW {
		b = make([]byte, l)
		a = (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
		err = syscall.GetAdaptersInfo(a, &l)
	}
	if err != nil {
		return nil, os.NewSyscallError("GetAdaptersInfo", err)
	}
	return a, nil
}

func localAddresses() ([]net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	aList, err := getAdapterList()
	if err != nil {
		return nil, err
	}

	for _, ifi := range ifaces {
		for ai := aList; ai != nil; ai = ai.Next {
			index := ai.Index

			if ifi.Index == int(index) {
				ipl := &ai.IpAddressList
				for ; ipl != nil; ipl = ipl.Next {

					fmt.Printf("id[%d]: name[%s] addres[%s] mask[%s]\n", ifi.Index, ifi.Name, ipl.IpAddress, ipl.IpMask)
				}
			}
		}
	}
	return ifaces, err
}

// main
func main() {
	ifaceAddress, _ := localAddresses()
	log.Println("insert id of interface")
	var i int
	fmt.Scanf("%d", &i)
	var ok bool = false
	var iface net.Interface
	for _, ifi := range ifaceAddress {
		if ifi.Index == i {
			ok = true
			iface = ifi
			break
		}
	}
	if ok {
		log.Println("selected " + iface.Name)
		l := logreceiver.NewLogReceiver(serviceName, serviceType, serviceDomain, dbPath, servicePort, cleanPeriodMs, maxLogs, iface)
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
		log.Println("open http://localhost:" + httpPort)
		log.Fatal(http.ListenAndServe("0.0.0.0:"+httpPort, nil))
	} else {
		log.Fatal("wrong interface")
	}
}
