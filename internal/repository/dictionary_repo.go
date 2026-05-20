package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// DictEntry represents a word entry from the ECDICT dictionary.
type DictEntry struct {
	Word        string `json:"word"`
	Phonetic    string `json:"phonetic"`
	Translation string `json:"translation"`
	Definition  string `json:"definition"`
	POS         string `json:"pos"`
	Exchange    string `json:"exchange"`
	Tag         string `json:"tag"`
	Collins     int    `json:"collins"`
	Oxford      int    `json:"oxford"`
}

// DictionaryRepo provides read-only access to the ECDICT SQLite database.
type DictionaryRepo struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDictionaryRepo opens the ECDICT database at the given path.
// Returns nil (no error) if the file doesn't exist — the app works without a dictionary.
func NewDictionaryRepo(dbPath string) *DictionaryRepo {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("dictionary: no ecdict.db found, dictionary lookup disabled")
		return nil
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("dictionary: failed to open %s: %v\n", dbPath, err)
		return nil
	}
	// Read-only optimization
	db.Exec("PRAGMA journal_mode=DELETE")
	db.Exec("PRAGMA query_only=ON")

	fmt.Println("dictionary: loaded", dbPath)
	return &DictionaryRepo{db: db}
}

// OpenECDICT opens the ECDICT database from a data directory.
// Looks for data/ecdict.db or data/stardict.db.
func OpenECDICT(dataDir string) *DictionaryRepo {
	for _, name := range []string{"ecdict.db", "stardict.db"} {
		p := filepath.Join(dataDir, name)
		if r := NewDictionaryRepo(p); r != nil {
			return r
		}
	}
	return nil
}

// Lookup finds a word in the dictionary (case-insensitive).
func (r *DictionaryRepo) Lookup(word string) (*DictEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	row := r.db.QueryRow(
		`SELECT word, phonetic, translation, definition, pos, exchange, tag, COALESCE(collins,0), COALESCE(oxford,0) FROM stardict WHERE word = ? COLLATE NOCASE`,
		strings.ToLower(strings.TrimSpace(word)),
	)
	var e DictEntry
	err := row.Scan(&e.Word, &e.Phonetic, &e.Translation, &e.Definition, &e.POS, &e.Exchange, &e.Tag, &e.Collins, &e.Oxford)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// LookupByForm tries to find a word by its form (using the exchange field).
// ECDICT stores inflections in the exchange field, e.g. "s:cats/p:catting/2:catting"
// This function tries the base word first, then strips common suffixes.
func (r *DictionaryRepo) LookupByForm(word string) (*DictEntry, error) {
	// Direct lookup first
	if e, err := r.Lookup(word); err != nil || e != nil {
		return e, err
	}

	lc := strings.ToLower(word)
	// Try common inflection patterns
	candidates := []string{}

	// -ing -> base
	if strings.HasSuffix(lc, "ing") {
		base := lc[:len(lc)-3]
		candidates = append(candidates, base, base+"e")
	}
	// -ed -> base
	if strings.HasSuffix(lc, "ed") {
		base := lc[:len(lc)-2]
		candidates = append(candidates, base, base+"e")
	}
	// -s/-es -> base
	if strings.HasSuffix(lc, "ies") {
		candidates = append(candidates, lc[:len(lc)-3]+"y")
	} else if strings.HasSuffix(lc, "es") {
		candidates = append(candidates, lc[:len(lc)-2])
	} else if strings.HasSuffix(lc, "s") && len(lc) > 2 {
		candidates = append(candidates, lc[:len(lc)-1])
	}
	// -er/-est -> base
	if strings.HasSuffix(lc, "er") {
		candidates = append(candidates, lc[:len(lc)-2], lc[:len(lc)-1])
	} else if strings.HasSuffix(lc, "est") {
		candidates = append(candidates, lc[:len(lc)-3], lc[:len(lc)-2])
	}
	// -ly -> base
	if strings.HasSuffix(lc, "ly") {
		candidates = append(candidates, lc[:len(lc)-2])
	}

	for _, c := range candidates {
		if len(c) < 2 {
			continue
		}
		if e, err := r.Lookup(c); err == nil && e != nil {
			return e, nil
		}
	}
	return nil, nil
}

// Close closes the dictionary database.
func (r *DictionaryRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
