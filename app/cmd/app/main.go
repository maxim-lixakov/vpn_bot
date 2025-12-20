package main

import (
	"context"
	"log"
	"net/http"
	"time"
	"vpn-app/internal/handlers"

	"vpn-app/internal/config"
	"vpn-app/internal/db"
	"vpn-app/internal/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pg, err := db.Open(cfg.PG)
	if err != nil {
		log.Fatal("db open: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrations.Up(ctx, pg); err != nil {
		log.Fatal("migrations: ", err)
	}

	srv := handlers.New(cfg, pg)

	log.Printf("app listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
