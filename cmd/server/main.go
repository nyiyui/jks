package main

import (
	"encoding/hex"
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

func main() {
	var dbPath string
	var bindAddress string
	flag.StringVar(&dbPath, "db-path", "db.sqlite3", "path to database")
	flag.StringVar(&bindAddress, "bind", "127.0.0.1:8080", "bind address")
	flag.Parse()

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

	authKey, err := hex.DecodeString(os.Getenv("JKS_STORE_AUTH_KEY"))
	if err != nil {
		panic(err)
	}
	store := sessions.NewFilesystemStore("", authKey)
	s, err := server.New(&database.Database{DB: db}, &oauth2.Config{
		ClientID:     os.Getenv("JKS_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("JKS_OAUTH_CLIENT_SECRET"),
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
		RedirectURL:  os.Getenv("JKS_OAUTH_REDIRECT_URI"),
	}, store, "nyiyui")
	if err != nil {
		panic(err)
	}
	panic(http.ListenAndServe(bindAddress, s))
}
