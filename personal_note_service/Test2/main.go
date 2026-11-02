package main

import (
	"log"
	"net/http"
)

func runServer(addr string, handler http.Handler) error {
	return http.ListenAndServe(addr, handler)
}

func main() {
	mux := http.NewServeMux()
	err := runServer(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
