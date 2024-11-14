package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

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

	store := sessions.NewFilesystemStore(".filesystem-store")
	s, err := server.New(&database.Database{DB: db}, &oauth2.Config{
		ClientID:     os.Getenv("JKS_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("JKS_OAUTH_CLIENT_SECRET"),
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
	}, store, "nyiyui")
	if err != nil {
		panic(err)
	}
	panic(http.ListenAndServe("127.0.0.1:8080", s))
}
