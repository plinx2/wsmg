package fs

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrStop is returned to stop the search
var ErrStop = errors.New("stop search")

// FindFunc is a function that is called for each file or directory
// It should return true if the file/directory matches the search criteria
// and ErrStop to stop searching deeper in the directory tree
type FindFunc func(path string) (bool, error)

// Find searches for files/directories in the given directory
// It walks the directory tree and calls the FindFunc for each file/directory
func Find(root string, fn FindFunc) ([]string, error) {
	var matches []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories that can't be accessed
		}

		match, err := fn(path)
		if match {
			matches = append(matches, path)
		}
		if err != nil {
			if errors.Is(err, ErrStop) {
				// Stop walking this branch
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}
