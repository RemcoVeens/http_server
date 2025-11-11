package main

import "net/http"

func main() {
	servemux := http.NewServeMux()
	servemux.Handle("/", http.FileServer(http.Dir(".")))
	server := http.Server{
		Handler: servemux,
		Addr:    ":8080",
	}
	server.ListenAndServe()
}
