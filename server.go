package main

import(
	"net/http"
	"fmt"
	"os"
	"code.google.com/p/go-uuid/uuid"
    "github.com/jessevdk/go-flags"
)

func channels(w http.ResponseWriter, r *http.Request){
	fmt.Println(fmt.Sprintf("%s @ %s", r.Method, r.URL.Path))
	if r.URL.Path == "/v1/channels" {
		id := uuid.New()
	  	w.Header().Set("Content-Type", "application/json")
		// w.Header().Set("Connection", "keep-alive") // because node adds it too
		fmt.Fprintf(w, fmt.Sprintf("{\"id\":\"%s\"}", id))
	} else if r.Method == "PUT" {
		fmt.Fprintf(w, "")
	} else {
		fmt.Println(r.URL.Path)
		fmt.Fprintf(w, "[{\"id\":\"abc\",\"name\":\"def\"}]")
	}
	fmt.Println("Served")
}

var port string

type Options struct {
    Port string `short:"p" long:"port" description:"Port the server should listen on"`
}

func init() {
	var options Options
	var parser = flags.NewParser(&options, flags.Default)
    if _, err := parser.Parse(); err != nil {
    	fmt.Println(fmt.Sprintf("Error when parsing arguments: %v", err))
        os.Exit(1)
    }
    port = options.Port
    if port == "" {
    	fmt.Println("Missing port declaration. Should run with -p <port>.")
    	os.Exit(1)
    }
}

func main() {
	http.HandleFunc("/v1/channels", channels)
	http.HandleFunc("/v1/channels/", channels)
	http.ListenAndServe(":"+port, nil)
}