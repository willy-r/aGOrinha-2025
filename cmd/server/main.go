package main

import (
	"gorinha-2025/internal/api"
	"log"

	"github.com/valyala/fasthttp"
)

func main() {
	router := api.NewRouter()

	log.Println("Serve running on :9999")
	if err := fasthttp.ListenAndServe(":9999", router.HandleRequest); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
