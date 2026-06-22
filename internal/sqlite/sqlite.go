package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "github.com/glebarez/sqlite"
)

type Storage struct {
	db *sql.DB
}

type FileEntries struct {
	Root       string
	RelPath    string
	BaseName   string
	ParentPath string
	IsDir      bool
	Size       int
	ModTime    time.Time
	Extension  string
}

type FileIndex struct {
	Id       int64
	BaseName string
	RelPath  string
}

func (db *Storage) RegisterFileEntry(f *FileEntries) (sql.Result, error) {
	result, err := db.db.Exec(
		`
		INSERT INTO files(root,rel_path,base_name,parent_path,is_dir,size,mod_time,extension) VALUES(?,?,?,?,?,?,?,?)

		`, f.Root, f.RelPath, f.BaseName, f.ParentPath, f.IsDir, f.Size, f.ModTime, f.Extension)
	if err != nil {
		return nil, err
	}

	return result, nil
}
func (db *Storage) RegisterFileIndex(f *FileIndex) error {
	_, err := db.db.Exec(
		`
		INSERT INTO file_fts(rowid,rel_path,base_name) VALUES(?,?,?)

		`, f.Id, f.RelPath, f.BaseName)

	return err
}

func New(fileName string) (*Storage, error) {
	dir := "./data"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, fileName)
	db, err := sql.Open("sqlite", dbPath)

	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			db.Close()
		}
	}()
	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := createFilesTable(db); err != nil {

		return nil, err
	}

	if err := createFtsTable(db); err != nil {

		return nil, err
	}

	success = true
	return &Storage{db: db}, nil

}

func createFilesTable(db *sql.DB) error {

	_, err := db.Exec(`

		CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY,
    root TEXT NOT NULL,
    rel_path TEXT NOT NULL UNIQUE,
    base_name TEXT NOT NULL,
    parent_path TEXT NOT NULL,
    is_dir INTEGER NOT NULL,
    size INTEGER NOT NULL,
    mod_time TEXT NOT NULL,
    extension TEXT NOT NULL
  );
		`)

	return err
}
func createFtsTable(db *sql.DB) error {
	_, err := db.Exec(
		`
	 CREATE VIRTUAL TABLE IF NOT EXISTS file_fts USING fts5(
       rel_path,
       base_name,
       content=files,
       content_rowid=id,
       tokenize="trigram"

);
		`)

	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (db *Storage) GetSearchResult(query string) (*sql.Rows, error) {
	return db.db.Query(`
		SELECT base_name, rel_path, rank
		FROM file_fts
		WHERE file_fts MATCH ? ORDER BY rank
	`, query)
}
