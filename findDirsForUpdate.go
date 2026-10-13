package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/lxma/golist/v2"
)

func findDirsForUpdate(root string, since time.Time, exclusions TExclusions) ([]string, error) {
	f, err := os.Open(root)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s ist not a directory", root)
	}
	if info.ModTime().After(since) {
		return []string{root}, nil
	}

	changedDirs, _, err := scanDir(root, since, exclusions)
	if err != nil {
		return nil, err
	}
	changedDirsSlice := changedDirs.ToSlice()
	sort.Strings(changedDirsSlice)
	return changedDirsSlice, nil
}

func scanDir(dir string, since time.Time, exclusions TExclusions) (changedDirs *golist.List[string], numUnchangedFiles int, err error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	changedDirs = golist.New[string]()
	numUnchangedFiles = 0
	const cBatchSize = 256
	for {
		entries, readErr := f.Readdir(cBatchSize)
		for _, entry := range entries {
			if exclusions.IsEntryExcluded(entry) {
				fmt.Printf("Excluded: %s/%s\n", dir, entry.Name())
				continue
			}
			fullEntryPath := filepath.Join(dir, entry.Name())
			if entry.ModTime().After(since) {
				if entry.IsDir() {
					changedDirs.PushBack(fullEntryPath)
				} else {
					return golist.MakeList(dir), 1, nil
				}
			} else if entry.IsDir() {
				changedSubDirs, numUnchangedFilesInSubdir, err := scanDir(fullEntryPath, since, exclusions)
				if err != nil {
					return nil, 0, err
				}
				numUnchangedFiles += numUnchangedFilesInSubdir
				changedDirs.PushBackList(changedSubDirs)
			} else {
				numUnchangedFiles += 1
			}
		}
		if readErr == io.EOF {
			break
		} else if readErr != nil {
			return nil, 0, readErr
		}
	}
	if changedDirs.Len()*gConfig.DirSizeFactor > numUnchangedFiles {
		// It's not worth storing each change individually
		return golist.MakeList(dir), 1, nil
	}
	return changedDirs, numUnchangedFiles, nil
}
