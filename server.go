package main

import(
	"net/http"
	"fmt"
	"code.google.com/p/go-uuid/uuid"
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

func main() {
	http.HandleFunc("/v1/channels", channels)
	http.HandleFunc("/v1/channels/", channels)
	http.ListenAndServe(":8080", nil)
}