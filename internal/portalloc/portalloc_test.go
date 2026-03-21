package portalloc

import (
	"testing"
)

func TestAllocateSequential(t *testing.T) {
	a := New(44000, 10)

	port1 := a.Allocate("ws-1")
	port2 := a.Allocate("ws-2")
	port3 := a.Allocate("ws-3")

	if port1 != 44000 {
		t.Errorf("first port = %d, want 44000", port1)
	}
	if port2 != 44010 {
		t.Errorf("second port = %d, want 44010", port2)
	}
	if port3 != 44020 {
		t.Errorf("third port = %d, want 44020", port3)
	}
}

func TestAllocateIdempotent(t *testing.T) {
	a := New(44000, 10)

	port1 := a.Allocate("ws-1")
	port2 := a.Allocate("ws-1")

	if port1 != port2 {
		t.Errorf("same workspace got different ports: %d vs %d", port1, port2)
	}
}

func TestFreeAndRecycle(t *testing.T) {
	a := New(44000, 10)

	a.Allocate("ws-1") // 44000
	a.Allocate("ws-2") // 44010
	a.Free("ws-1")

	// Next allocation should reuse the freed port
	port := a.Allocate("ws-3")
	if port != 44000 {
		t.Errorf("recycled port = %d, want 44000", port)
	}

	// ws-2 should still have its port
	port2, err := a.Get("ws-2")
	if err != nil {
		t.Fatalf("Get ws-2: %v", err)
	}
	if port2 != 44010 {
		t.Errorf("ws-2 port = %d, want 44010", port2)
	}
}

func TestFreeUnknownIsNoOp(t *testing.T) {
	a := New(44000, 10)
	a.Free("nonexistent") // should not panic
}

func TestGetUnallocated(t *testing.T) {
	a := New(44000, 10)

	_, err := a.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unallocated workspace")
	}
}

func TestGetAllocated(t *testing.T) {
	a := New(44000, 10)

	expected := a.Allocate("ws-1")
	got, err := a.Get("ws-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != expected {
		t.Errorf("Get = %d, want %d", got, expected)
	}
}

func TestFreeAndGetReturnsError(t *testing.T) {
	a := New(44000, 10)

	a.Allocate("ws-1")
	a.Free("ws-1")

	_, err := a.Get("ws-1")
	if err == nil {
		t.Error("expected error after freeing")
	}
}

func TestAllocateSpecific(t *testing.T) {
	a := New(44000, 10)

	// Allocate a specific port
	port := a.AllocateSpecific("ws-1", 44050)
	if port != 44050 {
		t.Errorf("specific port = %d, want 44050", port)
	}

	// Verify it's retrievable
	got, err := a.Get("ws-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != 44050 {
		t.Errorf("Get = %d, want 44050", got)
	}
}

func TestAllocateSpecificIdempotent(t *testing.T) {
	a := New(44000, 10)

	port1 := a.AllocateSpecific("ws-1", 44050)
	port2 := a.AllocateSpecific("ws-1", 44060) // different port requested

	if port1 != port2 {
		t.Errorf("same workspace got different ports: %d vs %d", port1, port2)
	}
	if port1 != 44050 {
		t.Errorf("port = %d, want 44050 (original)", port1)
	}
}

func TestAllocateSpecificConflict(t *testing.T) {
	a := New(44000, 10)

	a.Allocate("ws-1") // takes 44000
	port := a.AllocateSpecific("ws-2", 44000) // conflict

	if port == 44000 {
		t.Errorf("conflicting port should not be 44000")
	}
	// Should get the next available port
	if port != 44010 {
		t.Errorf("fallback port = %d, want 44010", port)
	}
}

func TestAllocateSpecificAdvancesNextIndex(t *testing.T) {
	a := New(44000, 10)

	// Allocate a port well ahead of the current nextIndex
	a.AllocateSpecific("ws-1", 44050)

	// Next sequential allocation should not collide
	port := a.Allocate("ws-2")
	if port == 44050 {
		t.Errorf("sequential allocation collided with specific allocation")
	}
	// nextIndex should have advanced past index 5 (44050)
	if port != 44060 {
		t.Errorf("next port = %d, want 44060", port)
	}
}

func TestNewDefault(t *testing.T) {
	a := NewDefault()

	port := a.Allocate("ws-1")
	if port != 44000 {
		t.Errorf("default first port = %d, want 44000", port)
	}
}
