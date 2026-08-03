package matchmaker

// Index is the O(1) inverted index mapping Keywords -> SocketIDs -> Nodes
type Index struct {
	KeywordMap map[string]map[string]*Node
}

// NewIndex creates a new inverted index
func NewIndex() *Index {
	return &Index{
		KeywordMap: make(map[string]map[string]*Node),
	}
}

// Add indexes a node by its client's keywords in O(K) where K is number of keywords
func (idx *Index) Add(keywords []string, node *Node) {
	clientID := node.Client.ID()
	for _, kw := range keywords {
		if _, exists := idx.KeywordMap[kw]; !exists {
			idx.KeywordMap[kw] = make(map[string]*Node)
		}
		idx.KeywordMap[kw][clientID] = node
	}
}

// Remove removes a client from all its keyword buckets in O(K)
func (idx *Index) Remove(keywords []string, clientID string) {
	for _, kw := range keywords {
		if m, exists := idx.KeywordMap[kw]; exists {
			delete(m, clientID)
			if len(m) == 0 {
				delete(idx.KeywordMap, kw) // Free memory if bucket is empty
			}
		}
	}
}
