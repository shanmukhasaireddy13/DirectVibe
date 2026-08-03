package matchmaker

// Node represents a client in the DLL
type Node struct {
	Client Client
	Prev   *Node
	Next   *Node
}

// DLL is an O(1) doubly linked list for the global connection queue
type DLL struct {
	Head *Node
	Tail *Node
	Size int
}

// NewDLL creates a new doubly linked list
func NewDLL() *DLL {
	return &DLL{}
}

// PushBack inserts a client at the tail in O(1)
func (l *DLL) PushBack(c Client) *Node {
	node := &Node{Client: c}
	if l.Tail == nil {
		l.Head = node
		l.Tail = node
	} else {
		node.Prev = l.Tail
		l.Tail.Next = node
		l.Tail = node
	}
	l.Size++
	return node
}

// Remove removes a specific node from the DLL in O(1)
func (l *DLL) Remove(node *Node) {
	if node == nil {
		return
	}

	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		l.Head = node.Next
	}

	if node.Next != nil {
		node.Next.Prev = node.Prev
	} else {
		l.Tail = node.Prev
	}

	// Unlink completely to help garbage collection
	node.Prev = nil
	node.Next = nil
	l.Size--
}

// PopFront removes and returns the head of the list in O(1)
func (l *DLL) PopFront() *Node {
	if l.Head == nil {
		return nil
	}
	node := l.Head
	l.Remove(node)
	return node
}
