package main

import (
	"os"
	"path"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
)

var inQuotesReplacer *strings.Replacer

func init() {
	inQuotesReplacer = strings.NewReplacer(
		"\"", "\\\"",
		"\\", "\\\\",
	)
}

func InitializeBleve() error {
	path := path.Join(config.DatabaseDir, "sugoi.bleve")
	var err error

	stat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	if stat == nil {
		mapping := BuildNewMapping()
		bleveIndex, err = bleve.New(path, mapping)
	} else {
		bleveIndex, err = bleve.Open(path)
	}

	if err != nil {
		return err
	}

	return nil
}

func BuildNewMapping() *mapping.IndexMappingImpl {
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddCustomAnalyzer("simpleUnicode", map[string]interface{}{
		"type":      custom.Name,
		"tokenizer": unicode.Name,
		"token_filters": []string{
			lowercase.Name,
		},
	})
	indexMapping.DefaultAnalyzer = "simpleUnicode"

	thingMapping := bleve.NewDocumentMapping()
	indexMapping.DefaultMapping = thingMapping

	TitleMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("title", TitleMapping)

	TagsMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("tags", TagsMapping)

	ArtistMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("artist", ArtistMapping)

	CollectionMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("collection", CollectionMapping)

	CoverMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("cover", CoverMapping)

	DescriptionMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("description", DescriptionMapping)

	IdMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("id", IdMapping)

	LanguageMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("language", LanguageMapping)

	MagazineMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("magazine", MagazineMapping)

	ParodyMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("parody", ParodyMapping)

	PublisherMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("publisher", PublisherMapping)

	RatingMapping := bleve.NewNumericFieldMapping()
	thingMapping.AddFieldMappingsAt("rating", RatingMapping)

	MarksMapping := bleve.NewNumericFieldMapping()
	thingMapping.AddFieldMappingsAt("marks", MarksMapping)

	TypeMapping := bleve.NewTextFieldMapping()
	thingMapping.AddFieldMappingsAt("type", TypeMapping)

	UpdatedAtMapping := bleve.NewDateTimeFieldMapping()
	thingMapping.AddFieldMappingsAt("updated_at", UpdatedAtMapping)

	CreatedAtMapping := bleve.NewDateTimeFieldMapping()
	thingMapping.AddFieldMappingsAt("created_at", CreatedAtMapping)

	// b, _ := json.MarshalIndent(indexMapping, "", "\t")
	// log.Println(string(b))

	return indexMapping
}

func BuildBleveSearchTerm(key string, val string) string {
	sb := strings.Builder{}

	sb.WriteString("+")
	sb.WriteString(strings.ToLower(key))
	sb.WriteString(":\"")
	sb.WriteString(inQuotesReplacer.Replace(val))
	sb.WriteString("\"")

	return sb.String()
}
