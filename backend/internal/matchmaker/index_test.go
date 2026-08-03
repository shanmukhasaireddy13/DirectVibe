package matchmaker

import (
	"testing"
)

func TestIndex_AddAndRemove(t *testing.T) {
	idx := NewIndex()

	c1 := &mockClient{id: "1"}
	n1 := &Node{Client: c1}
	
	keywords := []string{"gaming", "music"}

	// Add to index
	idx.Add(keywords, n1)

	if len(idx.KeywordMap["gaming"]) != 1 {
		t.Fatalf("Expected gaming bucket size 1")
	}
	if len(idx.KeywordMap["music"]) != 1 {
		t.Fatalf("Expected music bucket size 1")
	}

	// Verify pointer
	if idx.KeywordMap["gaming"]["1"] != n1 {
		t.Fatalf("Index pointer mismatch")
	}

	// Remove from index
	idx.Remove(keywords, "1")

	if len(idx.KeywordMap) != 0 {
		t.Fatalf("Expected entire map to be empty as buckets should be cleaned up")
	}
}
