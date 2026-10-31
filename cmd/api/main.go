// @title Notes API
// @version 1.0
// @description REST API for notes
// @BasePath /api/v1
package main

import (
	"context"
	"example.com/pr16/internal/httpapi"
	"example.com/pr16/internal/repo"
	"example.com/pr16/internal/service"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	repository, e := repo.Open(context.Background(), os.Getenv("DB_DSN"), 20)
	if e != nil {
		log.Fatal(e)
	}
	defer repository.DB.Close()
	h := &httpapi.Handler{Service: service.New(repository)}
	m := h.Router()
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Fatal((&http.Server{Addr: addr, Handler: m, ReadHeaderTimeout: 5 * time.Second}).ListenAndServe())
}
