package sqlitestore

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	_ "embed" // for side effect

	_ "modernc.org/sqlite" // for side effect

	"github.com/jmoiron/sqlx"
)

// errors form database
var (
	// ErrNoRowsAffected by the operation.
	ErrNoRowsAffected = errors.New("no rows affected by operation")

	// ErrUniqueConstraintViolation indicates the primary key, or a secondary index, had a collision
	ErrUniqueConstraintViolation = errors.New("unique constraint violation")

	// ErrDBDoesNotExist means that the database did not exist
	ErrDBDoesNotExist = errors.New("database does not exist")

	// ErrDBAlreadyClosed is returned if you call Close and the database is either already closed or it was
	// never opened in the first place.
	ErrDBAlreadyClosed = errors.New("database already closed")
)

type SqliteStore struct {
	dbSpec string
	mu     sync.RWMutex
	db     *sqlx.DB
}

var (
	//go:embed schema.sql
	schema string

	// regexp for matching comments and empty lines
	commentsAndEmptyLinesRegex = regexp.MustCompile("--.*?\n$|^\\s+$")
)

// New creates a new sqliteStore instance. If the database does not exist
// it is created.
func New(dbSpec string) (*SqliteStore, bool, error) {
	db, created, err := openDB(dbSpec)
	if err != nil {
		return nil, false, err
	}

	return &SqliteStore{
		dbSpec: dbSpec,
		db:     db,
	}, created, nil
}

// Close the sqliteStore.
func (s *SqliteStore) Close() error {
	return s.db.Close()
}

func openDB(dbSpec string) (*sqlx.DB, bool, error) {
	// If the file does not already exist or the database is not an in-memory database
	// we need to create the schema.
	dbNeedsCreation := true
	if !strings.Contains(dbSpec, ":memory:") {
		_, err := os.Stat(dbSpec)
		dbNeedsCreation = os.IsNotExist(err)
	}

	db, err := sqlx.Open("sqlite", dbSpec)
	if err != nil {
		return nil, false, fmt.Errorf("unable to open database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, false, fmt.Errorf("unable to ping database: %w", err)
	}

	if dbNeedsCreation {
		err := createSchema(db)
		if err != nil {
			return nil, false, fmt.Errorf("unable to create schema: %w", err)
		}
		log.Printf("created database [%s]", dbSpec)
	}

	return db, dbNeedsCreation, nil
}

// createSchema populates a schema into an sqlx database handle
func createSchema(db *sqlx.DB) error {
	// Create the schema first
	for n, statement := range strings.Split(schema, ";") {
		statement = trimCommentsAndWhitespace(statement)

		if statement == "" {
			continue
		}

		_, err := db.Exec(statement)
		if err != nil {
			return fmt.Errorf("statement %d failed: \"%s\" : %w", n+1, statement, err)
		}
	}

	return nil
}

// trimCommentsAndWhitespace removes comments and superfluous whitespace
func trimCommentsAndWhitespace(s string) string {
	sb := strings.Builder{}

	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		b := commentsAndEmptyLinesRegex.ReplaceAll([]byte(line), nil)
		_, err := sb.Write(b)
		if err != nil {
			log.Fatalf("error removing comments: %v", err)
		}
	}
	return sb.String()
}

// CheckForZeroRowsAffected ensures that if zero rows are affected by operations that
// should have side-effects, an error is returned.
func CheckForZeroRowsAffected(r sql.Result, err error) error {
	if r == nil {
		return err
	}
	affected, err2 := r.RowsAffected()
	if err2 != nil {
		return err2
	}
	if affected == 0 {
		return ErrNoRowsAffected
	}

	return err
}
