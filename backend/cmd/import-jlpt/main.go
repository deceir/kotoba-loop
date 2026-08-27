package main

import (
	"context"
	"log"
	"os"
	"time"

	"kotoba-loop/backend/internal/database"
	"kotoba-loop/backend/internal/jlpt"
)

func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "kotoba:kotoba_dev@tcp(localhost:3306)/kotoba_loop?parseTime=true"
	}
	db, err := database.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	report, err := (&jlpt.Importer{DB: db}).Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("JLPT import complete: %d levels, %d source rows, %d new words, %d new deck links", report.Levels, report.Rows, report.Words, report.Links)
}
