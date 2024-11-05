package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/server"
)

type Config struct{}

func main() {
	var dbPath string
	var configPath string
	flag.StringVar(&dbPath, "db-path", "db.sqlite3", "path to database")
	flag.StringVar(&configPath, "config-path", "jks-server-config.json", "path to config")
	flag.Parse()

	configRaw, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	var config Config
	err = json.Unmarshal(configRaw, &config)
	if err != nil {
		panic(err)
	}

	log.Printf("opening database...")
	db, err := database.Open(dbPath)
	if err != nil {
		panic(err)
	}
	log.Printf("migrating database...")
	database.Migrate(db.DB)
	if err != nil {
		panic(err)
	}
	log.Printf("database ready.")

	s, err := server.New(&database.Database{db})
	if err != nil {
		panic(err)
	}
	panic(http.ListenAndServe("127.0.0.1:8080", s))
}
