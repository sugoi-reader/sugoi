package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mholt/archiver/v4"
)

type Thing struct {
	File *FilePointer
	FileMetadataDynamic
	FileMetadataStatic
}

func NewThingFromHash(hash string) (*Thing, error) {
	file, found := filePointers.ByHash[hash]
	if !found {
		return nil, fmt.Errorf("File %s not found", hash)
	}

	ret := Thing{}
	ret.File = file

	var err error
	static, err := NewFileMetadataStaticFromFile(file.StaticMetaPath())
	if err != nil {
		log.Println(err)
		ret.FileMetadataStatic = FileMetadataStatic{}
	} else {
		ret.FileMetadataStatic = *static
	}
	ret.FileMetadataStatic.FillEmptyFields(file)

	dynamic, err := NewFileMetadataDynamicFromFile(file.DynamicMetaPath())
	if err != nil {
		log.Println(err)
		ret.FileMetadataDynamic = FileMetadataDynamic{}
	} else {
		ret.FileMetadataDynamic = *dynamic
	}
	ret.FileMetadataDynamic.FillEmptyFields(file)

	return &ret, nil
}

func (this *Thing) FillEmptyFields(file *FilePointer) {
	if len(this.Title) == 0 {
		this.Title = file.PathKey
	}

	if len(this.Collection) == 0 {
		this.Collection = fmt.Sprintf("No Collection (%s)", file.DirHash())
	}

	// if len(this.Cover) == 0 {
	// 	this.Cover = config.DefaultCoverFileName
	// }
}

func (this *Thing) Key() string {
	return this.File.Key
}

func (this *Thing) Hash() string {
	return this.File.Hash
}

func (this *Thing) BuildPathKey() string {
	p := this.Key()

	var re = regexp.MustCompile(`{{.*?}}`)

	p = re.ReplaceAllStringFunc(p, func(s string) string {
		s = strings.Replace(s, "{{", "", -1)
		s = strings.Replace(s, "}}", "", -1)
		return strings.ToLower(s)
	})

	return path.Clean(p)
}

func (this *Thing) TrySaveDynamic() error {
	var err error
	metaFilePath := this.File.DynamicMetaPath()
	_, err = os.Stat(metaFilePath)
	if os.IsNotExist(err) {
		os.MkdirAll(path.Dir(metaFilePath), 0755)
	}

	f, err := os.OpenFile(metaFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	e := json.NewEncoder(f)
	e.SetIndent("", "\t")

	err = e.Encode(this.FileMetadataDynamic)
	if err != nil {
		return err
	}

	return nil
}

func (this *Thing) TrySaveRating(rating int) error {
	old := this.FileMetadataDynamic
	this.Rating = rating
	this.UpdatedAt = time.Now()

	err := this.TrySaveDynamic()
	if err != nil {
		this.FileMetadataDynamic = old
		return err
	}
	this.File.Reindex()

	return nil
}

func (this *Thing) AddMark() error {
	old := this.FileMetadataDynamic
	this.Marks++

	err := this.TrySaveDynamic()
	if err != nil {
		this.FileMetadataDynamic = old
		return err
	}
	this.File.Reindex()

	return nil
}

func (this *Thing) SubMark() error {
	old := this.FileMetadataDynamic
	this.Marks--

	err := this.TrySaveDynamic()
	if err != nil {
		this.FileMetadataDynamic = old
		return err
	}
	this.File.Reindex()

	return nil
}

func (this *Thing) TrySaveCover(file string, isUpdate bool) error {
	prefix := this.FileUrlPrefix()
	realLocation := this.File.RealLocation()
	newCover, err := filepath.Rel(prefix, file)

	files, err := this.ListFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		if file == file {
			old := this.FileMetadataDynamic
			this.Cover = newCover
			if isUpdate {
				this.UpdatedAt = time.Now()
			}

			err := this.TrySaveDynamic()
			if err != nil {
				this.FileMetadataDynamic = old
				return err
			}
			this.File.Reindex()

			return nil
		}
	}

	return fmt.Errorf("File %s doesn't exists in %s", newCover, realLocation)
}

func (this *Thing) CoverImageUrl() string {
	if len(this.Cover) > 0 {
		return this.Cover
	}

	f, err := this.ListFilesRaw()
	if err != nil || len(f) == 0 {
		return "/static/empty.jpg"
	}

	if this.Thumbnail > 0 {
		if len(f) >= this.Thumbnail {
			this.TrySaveCover(f[this.Thumbnail-1], false)

			return f[this.Thumbnail-1]
		}
	}
	return f[0]
}

func (this *Thing) FileUrlPrefix() string {
	return fmt.Sprintf("/thing/file/%s", this.Hash())
}

func (this *Thing) FileUrl(f string) string {
	return fmt.Sprintf("%s/%s", this.FileUrlPrefix(), strings.TrimLeft(f, "/"))
}

func (this *Thing) ReadFileUrl(i int) string {
	return fmt.Sprintf("%s/%d", this.ReadUrl(), i)
}

func (this *Thing) ThumbUrl(f string) string {
	if len(f) > 0 {
		return fmt.Sprintf("%s?size=thumb", this.FileUrl(f))
	}
	return "/static/empty-256.jpg"
}

func (this *Thing) DetailsUrl() string {
	return fmt.Sprintf("/thing/details/%s", this.Hash())
}

func (this *Thing) ReadUrl() string {
	return fmt.Sprintf("/thing/read/%s", this.Hash())
}

func (this *Thing) SortedTags() map[string][]SearchTerm {
	ret := make(map[string][]SearchTerm)

	if len(this.Artist) != 0 {
		ret["Artist"] = append(ret["Artist"], NewSearchTerm("artist", this.Artist))
	}

	if len(this.Language) != 0 {
		ret["Language"] = append(ret["Language"], NewSearchTerm("language", this.Language))
	}

	if len(this.Parody) != 0 {
		ret["Parody"] = append(ret["Parody"], NewSearchTerm("parody", this.Parody))
	}

	if len(this.Magazine) != 0 {
		ret["Magazine"] = append(ret["Magazine"], NewSearchTerm("magazine", this.Magazine))
	}

	if len(this.Publisher) != 0 {
		ret["Publisher"] = append(ret["Publisher"], NewSearchTerm("publisher", this.Publisher))
	}

	for _, tag := range this.Tags {
		if len(tag) != 0 {
			ret["Tags"] = append(ret["Tags"], NewSearchTerm("tags", tag))
		}
	}

	return ret
}

func (this *Thing) CollectionDetailsUrl() string {
	u := new(url.URL)
	u.Path = "/"
	q := u.Query()
	q.Set("q", BuildBleveSearchTerm("Collection", this.Collection))
	u.RawQuery = q.Encode()
	return u.String()
}

func (this *Thing) FilledStarsRepeat(str string) string {
	i := this.Rating

	if i > 5 {
		i = 5
	}

	if i < 0 {
		i = 0
	}

	return strings.Repeat(str, i)
}

func (this *Thing) EmptyStarsRepeat(str string) string {
	i := this.Rating

	if i > 5 {
		i = 5
	}

	if i < 0 {
		i = 0
	}

	return strings.Repeat(str, 5-i)
}

func (this *Thing) ListFiles() ([]string, error) {
	raw, err := this.ListFilesRaw()
	if err != nil {
		return nil, err
	}
	ret := make([]string, len(raw))
	for key, val := range raw {
		ret[key] = this.FileUrl(val)
	}
	return ret, nil
}

func (this *Thing) ListFilesRaw() ([]string, error) {
	var files []string

	compressedFileName := this.File.RealLocation()

	fsys, err := archiver.FileSystem(nil, compressedFileName)
	if err != nil {
		return nil, err
	}

	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.Index(path, ".yaml") != -1 {
			return nil
		}

		files = append(files, path)
		return nil
	})

	sort.Strings(files)
	return files, nil
}

func (this *Thing) getFileReader(file string) (io.Reader, MultiCloser, error) {
	var closers MultiCloser

	if len(file) > 0 && file[len(file)-1] != '/' {
		compressedFileName := path.Clean(path.Join(this.File.RealLocation()))

		fsys, err := archiver.FileSystem(nil, compressedFileName)
		if err != nil {
			return nil, closers, err
		}

		ret, err := fsys.Open(file)

		if err != nil {
			return nil, closers, fmt.Errorf("Couldn't read file %s from %s", compressedFileName, file)
		}
		closers = append(closers, ret)

		return ret, closers, nil
	}

	return nil, closers, fmt.Errorf("Invalid file: %s", file)
}
