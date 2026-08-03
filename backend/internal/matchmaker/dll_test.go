package matchmaker

import (
	"testing"
	"time"
)

type mockClient struct {
	id       string
	keywords []string
}

func (m *mockClient) ID() string                      { return m.id }
func (m *mockClient) Keywords() []string              { return m.keywords }
func (m *mockClient) EnqueueTime() time.Time          { return time.Now() }
func (m *mockClient) SendMatch(otherID string, o bool) {}

func TestDLL_PushAndPop(t *testing.T) {
	dll := NewDLL()

	c1 := &mockClient{id: "1"}
	c2 := &mockClient{id: "2"}

	dll.PushBack(c1)
	dll.PushBack(c2)

	if dll.Size != 2 {
		t.Fatalf("Expected size 2, got %d", dll.Size)
	}

	n1 := dll.PopFront()
	if n1.Client.ID() != "1" {
		t.Fatalf("Expected client 1 to pop first")
	}

	if dll.Size != 1 {
		t.Fatalf("Expected size 1, got %d", dll.Size)
	}
}

func TestDLL_RemoveMiddle(t *testing.T) {
	dll := NewDLL()

	n1 := dll.PushBack(&mockClient{id: "1"})
	n2 := dll.PushBack(&mockClient{id: "2"})
	n3 := dll.PushBack(&mockClient{id: "3"})

	// Remove middle element in O(1)
	dll.Remove(n2)

	if dll.Size != 2 {
		t.Fatalf("Expected size 2 after removal, got %d", dll.Size)
	}

	if n1.Next != n3 || n3.Prev != n1 {
		t.Fatalf("DLL links were not updated correctly after middle removal")
	}
}

func TestDLL_RemoveHeadTail(t *testing.T) {
	dll := NewDLL()

	n1 := dll.PushBack(&mockClient{id: "1"})
	n2 := dll.PushBack(&mockClient{id: "2"})

	dll.Remove(n1)
	if dll.Head != n2 || dll.Tail != n2 {
		t.Fatalf("Head and Tail should point to n2")
	}
	if dll.Size != 1 {
		t.Fatalf("Size should be 1")
	}

	dll.Remove(n2)
	if dll.Head != nil || dll.Tail != nil || dll.Size != 0 {
		t.Fatalf("List should be empty")
	}
}
