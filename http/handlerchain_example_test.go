package http

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func Example() {
	timeMux := http.NewServeMux()
	timeMux.Handle("/time/RFC822", tellTime("RFC822"))
	timeMux.Handle("/time/RFC822Z", tellTime("RFC822Z"))

	helloMux := http.NewServeMux()
	helloMux.Handle("/hello", NewHandlerChain(http.HandlerFunc(postOnly), http.HandlerFunc(sayHello)))

	NewHandlerChain(http.HandlerFunc(logger), timeMux, helloMux, http.HandlerFunc(http.NotFound))
	http.ListenAndServe(":8000", NewHandlerChain(http.HandlerFunc(logger), timeMux, helloMux, http.HandlerFunc(http.NotFound)))
}

func tellTime(format string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := time.Now().UTC().Format(format)
		fmt.Fprintln(w, "Hello! The current time is:", t)
	}
}

func logger(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for:", r.URL.String())
}

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello there!!")
}

func postOnly(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only POST here", http.StatusMethodNotAllowed)
	}
}
