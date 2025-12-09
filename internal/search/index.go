// Package search provides full-text search functionality for beans using Bleve.
package search

import (
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/hmans/beans/internal/bean"
)

// Index wraps a Bleve index for searching beans.
type Index struct {
	index bleve.Index
	path  string
}

// beanDocument is the structure stored in the Bleve index.
type beanDocument struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// NewIndex creates or opens a Bleve index at the given path.
// Returns the index and a boolean indicating if the index was rebuilt (true = rebuilt, false = opened existing).
func NewIndex(indexPath string) (*Index, bool, error) {
	var idx bleve.Index
	var err error
	rebuilt := false

	// Try to open existing index
	idx, err = bleve.Open(indexPath)
	if err != nil {
		// Index doesn't exist or is corrupted, create new one
		rebuilt = true
		if err := os.RemoveAll(indexPath); err != nil && !os.IsNotExist(err) {
			return nil, false, err
		}

		indexMapping := buildIndexMapping()
		idx, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, false, err
		}
	}

	return &Index{
		index: idx,
		path:  indexPath,
	}, rebuilt, nil
}

// buildIndexMapping creates the Bleve index mapping for bean documents.
func buildIndexMapping() mapping.IndexMapping {
	// Create a text field mapping with the standard analyzer
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = "standard"

	// Create a keyword field mapping for ID (stored but not analyzed)
	keywordFieldMapping := bleve.NewKeywordFieldMapping()

	// Create the document mapping
	beanMapping := bleve.NewDocumentMapping()
	beanMapping.AddFieldMappingsAt("id", keywordFieldMapping)
	beanMapping.AddFieldMappingsAt("slug", textFieldMapping)
	beanMapping.AddFieldMappingsAt("title", textFieldMapping)
	beanMapping.AddFieldMappingsAt("body", textFieldMapping)

	// Create the index mapping with BM25 scoring for better relevance ranking
	indexMapping := bleve.NewIndexMapping()
	indexMapping.DefaultMapping = beanMapping
	indexMapping.DefaultAnalyzer = "standard"
	indexMapping.IndexDynamic = false
	indexMapping.StoreDynamic = false

	// Use BM25 scoring algorithm (available in Bleve v2.5.0+)
	// BM25 provides better relevance ranking than TF-IDF, especially for:
	// - Handling term frequency saturation (repeated terms don't over-boost)
	// - Normalizing for document length (short docs aren't unfairly penalized)
	indexMapping.ScoringModel = "bm25"

	return indexMapping
}

// Close closes the index.
func (idx *Index) Close() error {
	return idx.index.Close()
}

// Path returns the index path.
func (idx *Index) Path() string {
	return idx.path
}

// IndexBean adds or updates a bean in the search index.
func (idx *Index) IndexBean(b *bean.Bean) error {
	doc := beanDocument{
		ID:    b.ID,
		Slug:  b.Slug,
		Title: b.Title,
		Body:  b.Body,
	}
	return idx.index.Index(b.ID, doc)
}

// DeleteBean removes a bean from the search index.
func (idx *Index) DeleteBean(id string) error {
	return idx.index.Delete(id)
}

// DefaultSearchLimit is the default maximum number of search results.
const DefaultSearchLimit = 1000

// Search executes a search query and returns matching bean IDs.
// The limit parameter controls the maximum number of results (0 uses DefaultSearchLimit).
func (idx *Index) Search(queryStr string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}

	// Use query string syntax which supports:
	// - Simple terms: "authentication"
	// - Boolean operators: "user AND password"
	// - Wildcards: "auth*"
	// - Phrases: "\"user login\""
	// - Field-specific: "title:login"
	query := bleve.NewQueryStringQuery(queryStr)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"id"} // Only return ID field

	result, err := idx.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result.Hits))
	for _, hit := range result.Hits {
		ids = append(ids, hit.ID)
	}

	return ids, nil
}

// RebuildFromBeans rebuilds the entire index from a slice of beans.
// This clears the existing index and re-indexes all provided beans.
func (idx *Index) RebuildFromBeans(beans []*bean.Bean) error {
	// Create a new index at the same path
	if err := idx.index.Close(); err != nil {
		return err
	}

	// Remove old index
	if err := os.RemoveAll(idx.path); err != nil {
		return err
	}

	// Create fresh index
	indexMapping := buildIndexMapping()
	newIdx, err := bleve.New(idx.path, indexMapping)
	if err != nil {
		return err
	}
	idx.index = newIdx

	// Batch index all beans
	batch := idx.index.NewBatch()
	for _, b := range beans {
		doc := beanDocument{
			ID:    b.ID,
			Slug:  b.Slug,
			Title: b.Title,
			Body:  b.Body,
		}
		if err := batch.Index(b.ID, doc); err != nil {
			return err
		}
	}

	return idx.index.Batch(batch)
}

// IndexPath returns the default index path for a beans directory.
func IndexPath(beansRoot string) string {
	return filepath.Join(beansRoot, ".index")
}
