package search

import (
	"strings"

	"hubfly-files/internal/sqlite"
)

type Service struct {
	store *sqlite.Storage
}

type SearchResult struct {
	BaseName string `json:"baseName"`
	RelPath  string `json:"relPath"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size"`
	ModTime  string `json:"modTime"`
}

func New(store *sqlite.Storage) *Service {
	return &Service{store: store}
}

func (s *Service) Search(root, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchResult{}, nil
	}

	rows, err := s.store.SearchFiles(root, quoteFTS5(query), 50)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, SearchResult{
			BaseName: row.BaseName,
			RelPath:  row.RelPath,
			IsDir:    row.IsDir,
			Size:     row.Size,
			ModTime:  row.ModTime,
		})
	}

	return results, nil
}

func (s *Service) IndexFile(f *sqlite.FileEntries) error {
	return s.store.UpsertFileEntry(f)
}

func quoteFTS5(query string) string {
	return `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
}
