package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"time"
)

type FileMetadataStatic struct {
	Id          int       `json:"id"`
	Collection  string    `json:"collection"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Tags        []string  `json:"tags"`
	Language    string    `json:"language"`
	Artist      string    `json:"artist"`
	CreatedAt   time.Time `json:"created_at"`
	Parody      string    `json:"parody"`
	Magazine    string    `json:"magazine"`
	Publisher   string    `json:"publisher"`
	Description string    `json:"description"`
	Pages       int       `json:"pages"`
	Thumbnail   int       `json:"thumbnail"`
}

type FileMetadataDynamic struct {
	Cover     string    `json:"cover"`
	UpdatedAt time.Time `json:"updated_at"`
	Rating    int       `json:"rating"`
	Marks     int       `json:"marks"`
}

type FileMetadata struct {
	FileMetadataStatic
	FileMetadataDynamic
}

func NewFileMetadataStaticFromFile(file string) (*FileMetadataStatic, error) {
	var err error
	var stat fs.FileInfo
	var mode fs.FileMode

	stat, err = os.Stat(file)
	if err != nil {
		return nil, err
	}

	mode = stat.Mode()
	if !mode.IsRegular() {
		return nil, fmt.Errorf("'%s' is not a file", file)
	}

	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var ret FileMetadataStatic

	err = json.Unmarshal(bytes, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func NewFileMetadataDynamicFromFile(file string) (*FileMetadataDynamic, error) {
	var err error
	var stat fs.FileInfo
	var mode fs.FileMode

	stat, err = os.Stat(file)
	if err != nil {
		return nil, err
	}

	mode = stat.Mode()
	if !mode.IsRegular() {
		return nil, fmt.Errorf("'%s' is not a file", file)
	}

	reader, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var ret FileMetadataDynamic
	err = json.Unmarshal(bytes, &ret)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (this *FileMetadataStatic) FillEmptyFields(file *FilePointer) {
	if file == nil {
		return
	}

	if len(this.Title) == 0 {
		this.Title = file.PathKey
	}

	if len(this.Collection) == 0 {
		this.Collection = fmt.Sprintf("No Collection (%s)", file.DirHash())
	}

	if this.Pages == 0 {
		p, ok := filePointers.ByHash[file.Hash]

		if ok {
			t := Thing{File: p}
			f, err := t.ListFilesRaw()
			if err == nil {
				log.Printf("Dynamic page count for %s\n", file.Key)
				this.Pages = len(f)
			}
		}
	}
}

func (this *FileMetadataDynamic) FillEmptyFields(file *FilePointer) {
	// if len(this.Cover) == 0 {
	// 	this.Cover = config.DefaultCoverFileName
	// }
}
