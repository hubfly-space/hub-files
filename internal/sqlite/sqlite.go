package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	Size       int64
	ModTime    time.Time
	Extension  string
}

type SearchRow struct {
	BaseName string
	RelPath  string
	IsDir    bool
	Size     int64
	ModTime  string
	Rank     float64
}

type FileIndex struct {
	Id       int64
	BaseName string
	RelPath  string
}

func New(fileName string) (*Storage, error) {
	dbPath := fileName
	if !filepath.IsAbs(fileName) {
		dbPath = filepath.Join("./data", fileName)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

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
	if _, err := db.Exec(`PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`); err != nil {
		return nil, err
	}
	if err := createFilesTable(db); err != nil {
		return nil, err
	}
	if err := createFtsTable(db); err != nil {
		return nil, err
	}
	if err := createFtsTriggers(db); err != nil {
		return nil, err
	}
	if err := rebuildFtsIndex(db); err != nil {
		return nil, err
	}

	success = true
	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// Ping verifies the database connection is alive.
func (s *Storage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func createFilesTable(db *sql.DB) error {
	var tableSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'files'`).Scan(&tableSQL)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if strings.Contains(tableSQL, "rel_path TEXT NOT NULL UNIQUE") && !strings.Contains(tableSQL, "UNIQUE(root, rel_path)") {
		if err := migrateFilesTable(db); err != nil {
			return err
		}
	}

	_, err = db.Exec(filesTableSchema())
	return err
}

func migrateFilesTable(db *sql.DB) error {
	_, err := db.Exec(`
		DROP TRIGGER IF EXISTS files_ai;
		DROP TRIGGER IF EXISTS files_ad;
		DROP TRIGGER IF EXISTS files_au;
		DROP TABLE IF EXISTS file_fts;
		ALTER TABLE files RENAME TO files_old_migration;
	`)
	if err != nil {
		return err
	}

	if _, err := db.Exec(filesTableSchema()); err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT OR IGNORE INTO files (id, root, rel_path, base_name, parent_path, is_dir, size, mod_time, extension)
		SELECT id, root, rel_path, base_name, parent_path, is_dir, size, mod_time, extension
		FROM files_old_migration;
		DROP TABLE files_old_migration;
	`)
	return err
}

func filesTableSchema() string {
	return `
		CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY,
			root TEXT NOT NULL,
			rel_path TEXT NOT NULL,
			base_name TEXT NOT NULL,
			parent_path TEXT NOT NULL,
			is_dir INTEGER NOT NULL,
			size INTEGER NOT NULL,
			mod_time TEXT NOT NULL,
			extension TEXT NOT NULL,
			UNIQUE(root, rel_path)
		);
		CREATE INDEX IF NOT EXISTS files_root_parent_idx ON files(root, parent_path);
		CREATE INDEX IF NOT EXISTS files_root_rel_path_idx ON files(root, rel_path);
	`
}

func createFtsTable(db *sql.DB) error {
	_, err := db.Exec(`
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

func createFtsTriggers(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TRIGGER IF NOT EXISTS files_ai AFTER INSERT ON files BEGIN
			INSERT INTO file_fts(rowid, rel_path, base_name)
			VALUES (new.id, new.rel_path, new.base_name);
		END;

		CREATE TRIGGER IF NOT EXISTS files_ad AFTER DELETE ON files BEGIN
			INSERT INTO file_fts(file_fts, rowid, rel_path, base_name)
			VALUES('delete', old.id, old.rel_path, old.base_name);
		END;

		CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
			INSERT INTO file_fts(file_fts, rowid, rel_path, base_name)
			VALUES('delete', old.id, old.rel_path, old.base_name);
			INSERT INTO file_fts(rowid, rel_path, base_name)
			VALUES (new.id, new.rel_path, new.base_name);
		END;
	`)
	return err
}

func rebuildFtsIndex(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO file_fts(file_fts) VALUES('rebuild')`)
	return err
}

func (s *Storage) UpsertFileEntry(f *FileEntries) error {
	_, err := s.db.Exec(`
		INSERT INTO files(root, rel_path, base_name, parent_path, is_dir, size, mod_time, extension)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root, rel_path) DO UPDATE SET
			base_name = excluded.base_name,
			parent_path = excluded.parent_path,
			is_dir = excluded.is_dir,
			size = excluded.size,
			mod_time = excluded.mod_time,
			extension = excluded.extension
	`, f.Root, f.RelPath, f.BaseName, f.ParentPath, f.IsDir, f.Size, f.ModTime.Format(time.RFC3339Nano), f.Extension)
	return err
}

func (s *Storage) RegisterFileEntry(f *FileEntries) (sql.Result, error) {
	return s.db.Exec(`
		INSERT INTO files(root, rel_path, base_name, parent_path, is_dir, size, mod_time, extension)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(root, rel_path) DO UPDATE SET
			base_name = excluded.base_name,
			parent_path = excluded.parent_path,
			is_dir = excluded.is_dir,
			size = excluded.size,
			mod_time = excluded.mod_time,
			extension = excluded.extension
	`, f.Root, f.RelPath, f.BaseName, f.ParentPath, f.IsDir, f.Size, f.ModTime.Format(time.RFC3339Nano), f.Extension)
}

func (s *Storage) RegisterFileIndex(f *FileIndex) error {
	_, err := s.db.Exec(`INSERT INTO file_fts(rowid, rel_path, base_name) VALUES(?, ?, ?)`, f.Id, f.RelPath, f.BaseName)
	return err
}

func (s *Storage) DeleteFileEntry(root, relPath string) error {
	_, err := s.db.Exec(`DELETE FROM files WHERE root = ? AND rel_path = ?`, root, relPath)
	return err
}

func (s *Storage) DeletePathPrefix(root, relPath string) error {
	if relPath == "" || relPath == "." {
		_, err := s.db.Exec(`DELETE FROM files WHERE root = ?`, root)
		return err
	}

	pattern := escapeLike(relPath) + "/%"
	_, err := s.db.Exec(`
		DELETE FROM files
		WHERE root = ? AND (rel_path = ? OR rel_path LIKE ? ESCAPE '\')
	`, root, relPath, pattern)
	return err
}

func (s *Storage) SearchFiles(root, query string, limit int) ([]SearchRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT files.base_name, files.rel_path, files.is_dir, files.size, files.mod_time, file_fts.rank
		FROM file_fts
		JOIN files ON files.id = file_fts.rowid
		WHERE file_fts MATCH ? AND files.root = ?
		ORDER BY file_fts.rank
		LIMIT ?
	`, query, root, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchRow, 0)
	for rows.Next() {
		var row SearchRow
		if err := rows.Scan(&row.BaseName, &row.RelPath, &row.IsDir, &row.Size, &row.ModTime, &row.Rank); err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

func (s *Storage) GetSearchResult(query string) (*sql.Rows, error) {
	return s.db.Query(`
		SELECT base_name, rel_path, rank
		FROM file_fts
		WHERE file_fts MATCH ? ORDER BY rank
	`, query)
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
