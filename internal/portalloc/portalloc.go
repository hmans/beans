// Package portalloc manages workspace port allocation.
// Each workspace gets a unique port, starting at a base port and incrementing
// by a fixed step. Freed ports are recycled for new workspaces.
package portalloc

import (
	"fmt"
	"sync"
)

const (
	DefaultBasePort = 44000
	DefaultStep     = 10
)

// Allocator assigns unique ports to workspace IDs.
// Ports are managed in RAM only — no persistence.
type Allocator struct {
	mu        sync.Mutex
	basePort  int
	step      int
	assigned  map[string]int // workspaceID → port
	freed     []int          // recycled ports available for reuse
	nextIndex int            // next multiplier for generating new ports
}

// New creates a port allocator with the given base port and step.
func New(basePort, step int) *Allocator {
	return &Allocator{
		basePort: basePort,
		step:     step,
		assigned: make(map[string]int),
	}
}

// NewDefault creates a port allocator with default settings (base 44000, step 10).
func NewDefault() *Allocator {
	return New(DefaultBasePort, DefaultStep)
}

// Allocate assigns a port to the given workspace ID.
// If the workspace already has a port, it returns the existing one.
// Returns the assigned port.
func (a *Allocator) Allocate(workspaceID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()

	if port, ok := a.assigned[workspaceID]; ok {
		return port
	}

	var port int
	if len(a.freed) > 0 {
		port = a.freed[len(a.freed)-1]
		a.freed = a.freed[:len(a.freed)-1]
	} else {
		port = a.basePort + a.nextIndex*a.step
		a.nextIndex++
	}

	a.assigned[workspaceID] = port
	return port
}

// Free releases the port assigned to the given workspace ID,
// making it available for future allocations.
func (a *Allocator) Free(workspaceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	port, ok := a.assigned[workspaceID]
	if !ok {
		return
	}

	delete(a.assigned, workspaceID)
	a.freed = append(a.freed, port)
}

// Get returns the port assigned to the given workspace ID.
// Returns 0 and an error if no port is assigned.
func (a *Allocator) Get(workspaceID string) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	port, ok := a.assigned[workspaceID]
	if !ok {
		return 0, fmt.Errorf("no port allocated for workspace %q", workspaceID)
	}
	return port, nil
}
