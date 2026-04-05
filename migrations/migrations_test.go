package migrations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddingMigrationPinsVectorDimensionBeforeIndexCreation(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("0003_create_indexes.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(content)
	if !strings.Contains(sql, "ALTER TABLE document_chunks") {
		t.Fatal("expected migration to alter document_chunks before creating vector index")
	}

	if !strings.Contains(sql, "ALTER COLUMN embedding TYPE VECTOR(64)") {
		t.Fatal("expected migration to pin embedding dimension to VECTOR(64)")
	}

	if !strings.Contains(sql, "USING embedding::vector(64)") {
		t.Fatal("expected migration to cast existing embeddings to vector(64)")
	}
}
