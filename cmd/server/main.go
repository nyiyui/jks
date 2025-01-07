package main

import (
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/rdf"
	"nyiyui.ca/jks/server"
)

func main() {
	var dbPath string
	var bindAddress string
	var baseURI string
	var seekbackServerBaseURI string
	var seekbackServerToken string
	var seekbackServerEnabled bool
	var customLogUser string
	flag.StringVar(&dbPath, "db-path", "db.sqlite3", "path to database")
	flag.StringVar(&bindAddress, "bind", "127.0.0.1:8080", "bind address")
	flag.StringVar(&baseURI, "base-uri", "http://127.0.0.1/", "base URI for RDF")
	flag.StringVar(&seekbackServerBaseURI, "seekback-server-base-uri", "", "base URI for seekback-server")
	flag.StringVar(&seekbackServerToken, "seekback-server-token", "", "token for seekback-server")
	flag.BoolVar(&seekbackServerEnabled, "seekback-server-enabled", true, "enable seekback-server")
	flag.StringVar(&customLogUser, "custom-log-user", "", "custom log user")
	flag.Parse()

	if seekbackServerEnabled && seekbackServerBaseURI == "" {
		log.Fatalf("seekback-server-base-uri is required")
	}
	if seekbackServerEnabled && seekbackServerToken == "" {
		log.Fatalf("seekback-server-token is required")
	}

	serializer := rdf.NewSerializer(baseURI)

	log.Printf("opening database...")
	db, err := database.Open(dbPath)
	if err != nil {
		panic(err)
	}
	log.Printf("migrating database...")
	err = database.Migrate(db.DB)
	if err != nil && err != migrate.ErrNoChange {
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
	}, store, "nyiyui", serializer, customLogUser)
	if err != nil {
		panic(err)
	}
	if seekbackServerEnabled {
		s.SetupSeekbackServer(seekbackServerBaseURI, seekbackServerToken)
	}
	panic(http.ListenAndServe(bindAddress, s))
}
