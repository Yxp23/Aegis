package main

import (
	"log"
	"net/http"

	"github.com/Yxp23/aegis/internal/api"
	"github.com/Yxp23/aegis/internal/providers/mock"
	"github.com/Yxp23/aegis/internal/router"
)

func main() {
	mockProvider := &mock.Provider{}
	r := router.New(mockProvider)

	handler := api.NewHandler(r)

	log.Println("benchmark server listening on :9090")

	if err := http.ListenAndServe(":9090", handler); err != nil {
		log.Fatal(err)
	}
}
