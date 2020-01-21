package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cbergoon/merkletree"
)

// FileItem represents a leaf in the Merkle Tree
type FileItem struct {
	Path string
	Hash []byte
}

// ensure the struct is compatible with the merkletree library
var _ merkletree.Content = (*FileItem)(nil)

// String returns a readable presentation of the item
func (fi *FileItem) String() string {
	return fmt.Sprintf("[%s,%x]", fi.Path, fi.Hash)
}

// CalculateHash reads the contents of the file located at the
// given path and calculates the SHA256 hash of the data
func (fi *FileItem) CalculateHash() ([]byte, error) {
	if len(fi.Hash) > 0 {
		return fi.Hash, nil
	}

	f, err := os.Open(fi.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sha := sha256.New()
	if _, err := io.Copy(sha, f); err != nil {
		return nil, err
	}

	hash := sha.Sum(nil)
	fi.Hash = hash
	return hash[:], nil
}

// Equals returns true if the two items have the same path and content
func (fi *FileItem) Equals(otherContent merkletree.Content) (bool, error) {
	other, ok := otherContent.(*FileItem)
	if !ok {
		return false, nil
	}

	otherHash, err := other.CalculateHash()
	if err != nil {
		return false, err
	}

	myHash, err := fi.CalculateHash()
	if err != nil {
		return false, err
	}

	return strings.Compare(fi.Path, other.Path) == 0 && bytes.Equal(otherHash, myHash), nil
}

// BuildList recursively crawls a directory and adds every file to a slice
func BuildList(path string) ([]merkletree.Content, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var list []merkletree.Content
	for _, f := range files {
		path := filepath.Join(path, f.Name())
		if f.IsDir() {
			children, err := BuildList(path)
			if err != nil {
				return nil, err
			}
			list = append(list, children...)
		} else {
			fi := &FileItem{Path: path}
			if _, err := fi.CalculateHash(); err != nil {
				return nil, err
			}
			list = append(list, fi)
		}
	}

	return list, nil
}
