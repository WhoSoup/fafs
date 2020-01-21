package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/cbergoon/merkletree"
)

type FileItem struct {
	Path string
	Size int64
}

var _ merkletree.Content = (*FileItem)(nil)

func (fi FileItem) String() string {
	return fmt.Sprintf("[%s,%d]", fi.Path, fi.Size)
}

func (fi FileItem) CalculateHash() ([]byte, error) {
	simplehash := sha256.Sum256([]byte(fi.Path))
	return simplehash[:], nil
}

func (fi FileItem) Equals(other merkletree.Content) (bool, error) {
	return fi.Path == other.(FileItem).Path, nil
}

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
			list = append(list, FileItem{
				Path: path,
				Size: f.Size(),
			})
		}
	}

	return list, nil
}
