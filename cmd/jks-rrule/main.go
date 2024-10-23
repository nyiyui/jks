package main

import (
	"encoding"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pelletier/go-toml/v2"
	"github.com/teambition/rrule-go"
	"nyiyui.ca/jks/database"
)

type Cache struct {
	GenerateFrom time.Time
}

type RRules struct {
	GenerateInterval time.Duration
	Tasks            map[string]Task
}

type Task struct {
	RRuleSet *RRuleSet
	Task     database.Task
}

type RRuleSet rrule.Set

func (r *RRuleSet) UnmarshalText(text []byte) error {
	rs, err := rrule.StrToRRuleSet(string(text))
	if err != nil {
		return err
	}
	*r = RRuleSet(*rs)
	return nil
}

func (r *RRuleSet) MarshalText() ([]byte, error) {
	return []byte((*rrule.Set)(r).String()), nil
}

var _ encoding.TextMarshaler = (*RRuleSet)(nil)
var _ encoding.TextUnmarshaler = (*RRuleSet)(nil)

func getCacheDir() string {
	cacheDir, ok := os.LookupEnv("XDG_CACHE_HOME")
	if !ok {
		home, err := os.UserHomeDir()
		if err != nil {
			cacheDir = "./.cache"
		} else {
			cacheDir = filepath.Join(home, ".cache")
		}
	}
	return cacheDir
}

func main() {
	var dbPath string
	var rrulesPath string
	var cachePath string
	flag.StringVar(&dbPath, "db-path", "db.sqlite3", "path to database")
	flag.StringVar(&rrulesPath, "rrules-path", "jks-rrules.toml", "path to rrules")
	flag.StringVar(&cachePath, "cache-path", filepath.Join(getCacheDir(), "jks-rrule-cache.json"), "path to cache")
	flag.Parse()

	cacheRaw, err := os.OpenFile(cachePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	var cache Cache
	err = json.NewDecoder(cacheRaw).Decode(&cache)
	if err != nil {
		panic(err)
	}

	rrulesRaw, err := os.ReadFile(rrulesPath)
	if err != nil {
		panic(err)
	}
	var rrules RRules
	err = toml.Unmarshal(rrulesRaw, &rrules)
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
}

func createTasks(set *rrule.Set, db *sqlx.DB, cfg RRules, cache Cache) {
	times := set.Between(cache.GenerateFrom, time.Now().Add(cfg.GenerateInterval), true)
	for i, t := range times {
		log.Printf("[%d] generated task at %s.", i, t)
		db.Exec(`INSERT INTO tasks (id, description, quick_title, deadline, due) 
	}
}
