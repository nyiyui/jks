package main

import (
	"context"
	"encoding"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/teambition/rrule-go"
	"nyiyui.ca/jks/database"
	"nyiyui.ca/jks/storage"
)

type Cache struct {
	GenerateFrom time.Time
}

type RRules struct {
	GenerateInterval int
	Tasks            map[string]Task
}

type Task struct {
	RRuleSet *RRuleSet
	Task     storage.Task
	DryRun   bool
}

type RRuleSet rrule.Set

func (r *RRuleSet) UnmarshalText(text []byte) error {
	rs, err := rrule.StrToRRuleSet(string(text))
	if err != nil {
		return fmt.Errorf("parse RRULE set: %s", err)
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
	defer cacheRaw.Close()
	var cache Cache
	err = json.NewDecoder(cacheRaw).Decode(&cache)
	if err != nil {
		panic(err)
	}

	rrulesRaw, err := os.ReadFile(rrulesPath)
	if err != nil {
		panic(err)
	}
	var cfg RRules
	err = toml.Unmarshal(rrulesRaw, &cfg)
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

	generateTo := time.Now().Add(time.Duration(cfg.GenerateInterval) * time.Hour)
	for name := range cfg.Tasks {
		err := createForTask(name, &database.Database{db}, cfg, cache, cache.GenerateFrom, generateTo)
		if err != nil {
			panic(err)
		}
	}
	cache.GenerateFrom = generateTo
	log.Printf("writing to cache...")
	err = cacheRaw.Truncate(0)
	if err != nil {
		panic(err)
	}
	_, err = cacheRaw.Seek(0, 0)
	if err != nil {
		panic(err)
	}
	err = json.NewEncoder(cacheRaw).Encode(cache)
	if err != nil {
		panic(err)
	}
	log.Printf("wrote to cache.")
}

func createForTask(name string, st storage.Storage, cfg RRules, cache Cache, generateFrom, generateTo time.Time) error {
	log.Printf("createForTask %s - %s → %s", name,
		generateFrom, generateTo)
	taskCfg := cfg.Tasks[name]
	set := (*rrule.Set)(taskCfg.RRuleSet)
	times := set.Between(generateFrom, generateTo, true)
	for i, t := range times {
		if taskCfg.DryRun {
			log.Printf("[%s.%d] dry run for task at %s.", name, i, t)
			continue
		}
		log.Printf("[%s.%d] generated task at %s.", name, i, t)
		task := taskCfg.Task
		task.Due = &t
		task.Deadline = &t
		_, err := st.TaskAdd(task, context.Background())
		if err != nil {
			return fmt.Errorf("add for %s: %w", t, err)
		}
	}
	return nil
}
