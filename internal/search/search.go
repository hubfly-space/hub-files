package search

import (
	"fmt"
	"hubfly-files/internal/sqlite"
	"os"
)

type Service struct {
	store *sqlite.Storage
}

type SearchResult struct {
	BaseName string
	RelPath  string
}

func (s *Service) Search(query string) ([]SearchResult, error) {
	rows, err := s.store.GetSearchResult(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)

	for rows.Next() {
		var r SearchResult
		var rank float64

		if err = rows.Scan(&r.BaseName, &r.RelPath, &rank); err != nil {
			return nil, err
		}

		results = append(results, r)
	}

	return results, nil

}

func (s *Service) InsertIndex(f *sqlite.FileEntries) error {
	r, err := s.store.RegisterFileEntry(f)

	if err != nil {
		fmt.Printf("Error occured when indexing the file,%v", err)
		return err
	}

	r_id, err := r.LastInsertId()

	if err != nil {
		return fmt.Errorf("indexing file entry: %w", err)
	}
	fi := &sqlite.FileIndex{
		Id:       r_id,
		BaseName: f.BaseName,
		RelPath:  f.RelPath,
	}
	if err := s.store.RegisterFileIndex(fi); err != nil {
		fmt.Printf("Error occured when indexing the file,%v", err)
		return err
	}
	return nil
}

func IndexFile(root string) error {

	f, err := os.Open(root)
	if err != nil {
		return err
	}

	defer f.Close()

	for {

		entries, err := f.ReadDir(100)

		for _, entry := range entries {


		}

	}

}
