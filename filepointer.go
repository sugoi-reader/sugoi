package main

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve/v2"
)

type FilePointerList struct {
	List      []*FilePointer
	ByKey     map[string]*FilePointer
	ByPathKey map[string]*FilePointer
	ByHash    map[string]*FilePointer
}

func NewFilePointerList() FilePointerList {
	return FilePointerList{
		List:      make([]*FilePointer, 0),
		ByKey:     make(map[string]*FilePointer),
		ByPathKey: make(map[string]*FilePointer),
		ByHash:    make(map[string]*FilePointer),
	}
}

func (this *FilePointerList) Clear() {
	this.List = make([]*FilePointer, 0)
	this.ByKey = make(map[string]*FilePointer)
	this.ByPathKey = make(map[string]*FilePointer)
	this.ByHash = make(map[string]*FilePointer)
}

func (this *FilePointerList) Push(n *FilePointer) {
	this.List = append(this.List, n)
	this.ByKey[n.Key] = n
	this.ByPathKey[n.PathKey] = n
	this.ByHash[n.Hash] = n
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type FilePointer struct {
	Key      string
	Hash     string
	PathKey  string
	MetaPath string
}

func NewFilePointer(key string) (*FilePointer, error) {
	ret := FilePointer{}

	byteKey := []byte(key)
	byteHash := sha1.Sum(byteKey)

	ret.Key = key
	ret.Hash = fmt.Sprintf("%x", byteHash)
	ret.PathKey = ret.BuildPathKey()
	ret.MetaPath = path.Join(config.DatabaseDir, "meta", ret.PathKey)

	return &ret, nil
}

func (this *FilePointer) BuildPathKey() string {
	p := this.Key

	var re = regexp.MustCompile(`{{.*?}}`)

	p = re.ReplaceAllStringFunc(p, func(s string) string {
		s = strings.Replace(s, "{{", "", -1)
		s = strings.Replace(s, "}}", "", -1)
		return strings.ToLower(s)
	})

	return path.Clean(p)
}

func (this *FilePointer) RealLocation() string {
	p := this.Key
	for key, val := range config.DirVars {
		p = strings.ReplaceAll(p, fmt.Sprintf("{{%s}}", key), val)
	}
	return path.Clean(p)
}

func (this *FilePointer) StaticMetaPath() string {
	return path.Join(this.MetaPath, "static.json")
}

func (this *FilePointer) DynamicMetaPath() string {
	return path.Join(this.MetaPath, "dynamic.json")
}

func (this *FilePointer) DirHash() string {
	dir := path.Dir(this.PathKey)

	byteKey := []byte(dir)
	byteHash := crc32.ChecksumIEEE(byteKey)

	return fmt.Sprintf("%08X", byteHash)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func InitializeFilePointers() error {
	file, err := os.Open(path.Join(config.DatabaseDir, "files.txt"))

	if err != nil {
		return err
	}

	r := bufio.NewReader(file)

	filePointers = NewFilePointerList()

	for {
		line, err := r.ReadString('\n')
		line = strings.TrimSpace(line)

		if err == io.EOF && len(line) == 0 {
			return nil
		}

		if err != nil && err != io.EOF {
			filePointers.Clear()
			return err
		}

		if len(line) == 0 {
			continue
		}

		n, err := NewFilePointer(line)

		if err != nil {
			filePointers.Clear()
			return err
		}

		filePointers.Push(n)
	}
}

func (this *FilePointer) ReindexIntoBatch(idx *bleve.Batch) error {
	doc := this.BuildReindexDoc()

	err := idx.Index(this.Hash, doc)
	if err != nil {
		return err
	}

	return nil
}

func (this *FilePointer) Reindex() error {
	doc := this.BuildReindexDoc()

	err := bleveIndex.Index(this.Hash, doc)
	if err != nil {
		return err
	}

	return nil
}

func (this *FilePointer) BuildReindexDoc() FileMetadata {
	var err error
	var file FileMetadata
	static, err := NewFileMetadataStaticFromFile(this.StaticMetaPath())
	if err != nil {
		file.FileMetadataStatic = FileMetadataStatic{}
	} else {
		file.FileMetadataStatic = *static
	}
	file.FileMetadataStatic.FillEmptyFields(this)

	dynamic, err := NewFileMetadataDynamicFromFile(this.DynamicMetaPath())
	if err != nil {
		file.FileMetadataDynamic = FileMetadataDynamic{}
	} else {
		file.FileMetadataDynamic = *dynamic
	}
	file.FileMetadataDynamic.FillEmptyFields(this)

	return file
}
