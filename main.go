package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"
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

var tmpl = template.Must(template.ParseFiles("templates/layout.html"))

func writeData(w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	if strings.Contains(r.Header["Accept"][0], "html") {
		w.Header().Set("Content-Type", "text/html")
		err := tmpl.Execute(w, data)
		if err != nil {
			w.Write([]byte(`{"result":"Error"}`))
		}
		return
	}

	if strings.Contains(r.Header["Accept"][0], "json") {
		w.Header().Set("Content-Type", "application/json")
		b, err := json.Marshal(&data)
		if err != nil {
			w.Write([]byte(`{"result":"Error"}`))
			return
		}
		w.Write(b)
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	b, err := yaml.Marshal(&data)
	if err != nil {
		w.Write([]byte(`{"result":"Error"}`))
		return
	}
	w.Write(b)

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
