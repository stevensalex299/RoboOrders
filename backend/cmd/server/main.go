package main

import (
	"log"
	"net/http"
	"os"

	"github.com/stevensalex299/RoboOrders/backend/internal/api"
	"github.com/stevensalex299/RoboOrders/backend/internal/db"
	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

func main() {
	addr := envOr("ROBO_ORDERS_ADDR", ":8080")

	sqlDB, err := db.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	srv := &api.Server{Store: store.New(sqlDB)}
	handler := api.CORS(srv.Handler())

	log.Printf("RoboOrders API listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
