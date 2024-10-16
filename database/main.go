package database

import (
	"database/sql"
	"embed"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations
var migrations embed.FS

type Database struct {
	DB *sqlx.DB
}

func Open(path string) (*sqlx.DB, error) {
	return sqlx.Open("sqlite3", path)
}

func Migrate(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		panic(err) // shouldn't fail
	}
	m, err := migrate.NewWithInstance(
		"iofs", source,
		"sqlite3", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil {
		return err
	}
	return nil
}

type Task struct {
	ID          int64
	Description string
	QuickTitle  string `db:"quick_title"`

	// Deadline is the time after which this task is useless to complete.
	// For example, studying for an exam after the exam itself is useless (for the purpose of scoring well on the exam).
	// In this case, the deadline would be the exam start time.
	// In the future, this may become a reference to another task, such that once that task is started, this task is useless to complete..
	Deadline *time.Time `db:"deadline"`
	Due      *time.Time `db:"due"`
}

func (t Task) GetID() int64 { return t.ID }

func (t Task) String() string {
	return t.QuickTitle
}

type Activity struct {
	ID        int64
	TaskID    int64 `db:"task_id"`
	Location  string
	TimeStart time.Time `db:"time_start"`
	TimeEnd   time.Time `db:"time_end"`
}

func (a Activity) GetID() int64 { return a.ID }
