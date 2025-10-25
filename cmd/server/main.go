package main

import (
	"log"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
)

func main() {
	server := &http.Server{
		Addr:    ":8000",
		Handler: api.GetDefaultRoute(),
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
