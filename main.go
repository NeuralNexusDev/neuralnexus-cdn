package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	ip := os.Getenv("IP_ADDRESS")
	if ip == "" {
		ip = "0.0.0.0"
	}
	port := os.Getenv("REST_PORT")
	if port == "" {
		port = "3004"
	}

	router := http.NewServeMux()

	router.Handle("/", http.FileServer(http.Dir("./static")))

	server := http.Server{
		Addr:    ip + ":" + port,
		Handler: router,
	}
	log.Fatal(server.ListenAndServe())
}
