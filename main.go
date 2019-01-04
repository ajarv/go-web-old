package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func logRequest(req *http.Request) {
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("--Request :\n%v\n------------\n", string(requestDump))
}

func getDebugData(req *http.Request) map[string]interface{} {
	var hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown host"
	}

	getenvironment := func(data []string, getkeyval func(item string) (key, val string)) map[string]string {
		items := make(map[string]string)
		for _, item := range data {
			key, val := getkeyval(item)
			items[key] = val
		}
		return items
	}

	data := map[string]interface{}{
		"Host":       hostname,
		"ApiVersion": appVersion,
		"AppName":    appName,
		"Request":    map[string]interface{}{"Headers": req.Header},
	}
	if viewenv := req.URL.Query().Get("showenv"); viewenv != "" {
		environment := getenvironment(os.Environ(), func(item string) (key, val string) {
			splits := strings.Split(item, "=")
			key = splits[0]
			val = splits[1]
			return
		})
		data["Environment"] = environment
	}

	return data
}

func writeData(w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(&data)
	defer w.Write(b)
	if err != nil {
		b = []byte(`{"result":"Error"}`)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	data := getDebugData(r)
	writeData(w, r, data)
}

var appVersion = getEnv("APP_VERSION", "1.0.0")
var appName = getEnv("APP_NAME", "GO_WEB")

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	var host string
	var dir string
	var port string

	flag.StringVar(&dir, "dir", ".", "the directory to serve files from. Defaults to the current dir")
	flag.StringVar(&host, "host", "0.0.0.0", "listen host")
	flag.StringVar(&port, "port", "8080", "listen port")
	flag.Parse()

	r := mux.NewRouter()
	// This will serve files under http://localhost:8000/static/<filename>
	r.PathPrefix("/static/").Handler(http.FileServer(http.Dir(dir)))
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(dir))))
	r.HandleFunc("/", indexHandler)

	srv := &http.Server{
		Handler: r,
		Addr:    host + ":" + port,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	fmt.Fprintf(os.Stdout, "Server listening %s:%s\n", host, port)
	log.Fatal(srv.ListenAndServe())
}
