package main

import (
	"kotoba-loop/backend/internal/database"
	"kotoba-loop/backend/internal/httpapi"
	"log"
	"net/http"
	"os"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func main() {
	db, err := database.Open(env("MYSQL_DSN", "kotoba:kotoba_dev@tcp(localhost:3306)/kotoba_loop?parseTime=true"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	api := &httpapi.API{DB: db, Secret: env("JWT_SECRET", "local-development-secret"), Origin: env("WEB_ORIGIN", "http://localhost:5173")}
	addr := ":" + env("PORT", "8080")
	log.Printf("Kotoba Loop API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, api.Handler()))
}
