package repository

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.DB.Begin()
}

func InitDB(dbPath string, migrationsFS fs.FS) *DB {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	db.Exec("PRAGMA journal_mode=WAL")

	var colCount int

	// Ensure import_source_id column exists in articles (safe for existing DBs)
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name='import_source_id'").Scan(&colCount)
	if colCount == 0 {
		db.Exec("ALTER TABLE articles ADD COLUMN import_source_id TEXT NOT NULL DEFAULT ''")
	}

	// Ensure segments_json column exists in articles
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name='segments_json'").Scan(&colCount)
	if colCount == 0 {
		db.Exec("ALTER TABLE articles ADD COLUMN segments_json TEXT NOT NULL DEFAULT '[]'")
	}

	// Ensure html_content column exists in articles
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name='html_content'").Scan(&colCount)
	if colCount == 0 {
		db.Exec("ALTER TABLE articles ADD COLUMN html_content TEXT NOT NULL DEFAULT ''")
	}

	// Ensure tags column exists in articles
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('articles') WHERE name='tags'").Scan(&colCount)
	if colCount == 0 {
		db.Exec("ALTER TABLE articles ADD COLUMN tags TEXT NOT NULL DEFAULT '[]'")
	}

	// Ensure content_hash column exists in import_sources
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('import_sources') WHERE name='content_hash'").Scan(&colCount)
	if colCount == 0 {
		db.Exec("ALTER TABLE import_sources ADD COLUMN content_hash TEXT NOT NULL DEFAULT ''")
	}

	// Ensure vocabulary table has merged study_words fields
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('vocabulary') WHERE name='status'").Scan(&colCount)
	if colCount == 0 {
		db.Exec("ALTER TABLE vocabulary ADD COLUMN status TEXT NOT NULL DEFAULT 'new'")
		db.Exec("ALTER TABLE vocabulary ADD COLUMN encounter_count INTEGER NOT NULL DEFAULT 1")
		db.Exec("ALTER TABLE vocabulary ADD COLUMN source_articles TEXT NOT NULL DEFAULT '[]'")
		db.Exec("ALTER TABLE vocabulary ADD COLUMN updated_at TEXT NOT NULL DEFAULT '' ")
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		log.Fatalf("read migrations dir: %v", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		sqlBytes, err := fs.ReadFile(migrationsFS, entry.Name())
		if err != nil {
			log.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			log.Fatalf("run migration %s: %v", entry.Name(), err)
		}
	}

	fmt.Println("database initialized:", dbPath)
	return &DB{db}
}
