package graph

import (
	"fmt"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
)

//go:generate go tool gqlgen generate

// Resolver is the root resolver for the GraphQL schema.
// It holds a reference to beancore.Core for data access.
type Resolver struct {
	Core *beancore.Core
}

// ETagMismatchError is returned when an ETag validation fails.
// This allows callers to distinguish concurrency conflicts from other errors.
type ETagMismatchError struct {
	Provided string
	Current  string
}

func (e *ETagMismatchError) Error() string {
	return fmt.Sprintf("etag mismatch: provided %s, current is %s", e.Provided, e.Current)
}

// ETagRequiredError is returned when require_if_match is enabled and no ETag is provided.
type ETagRequiredError struct{}

func (e *ETagRequiredError) Error() string {
	return "if-match etag is required (set require_if_match: false in config to disable)"
}

// validateETag checks if the provided ifMatch etag matches the bean's current etag.
// Returns an error if validation fails or if require_if_match is enabled and no etag provided.
func (r *Resolver) validateETag(b *bean.Bean, ifMatch *string) error {
	cfg := r.Core.Config()
	requireIfMatch := cfg != nil && cfg.Beans.RequireIfMatch

	// If require_if_match is enabled and no etag provided, reject
	if requireIfMatch && (ifMatch == nil || *ifMatch == "") {
		return &ETagRequiredError{}
	}

	// If ifMatch provided, validate it
	if ifMatch != nil && *ifMatch != "" {
		currentETag := b.ETag()
		if currentETag != *ifMatch {
			return &ETagMismatchError{Provided: *ifMatch, Current: currentETag}
		}
	}

	return nil
}

// validateAndSetParent validates and sets the parent relationship.
func (r *Resolver) validateAndSetParent(b *bean.Bean, parentID string) error {
	if parentID == "" {
		b.Parent = ""
		return nil
	}

	// Normalise short ID to full ID
	normalizedParent, _ := r.Core.NormalizeID(parentID)

	// Validate parent type hierarchy
	if err := r.Core.ValidateParent(b, normalizedParent); err != nil {
		return err
	}

	// Check for cycles
	if cycle := r.Core.DetectCycle(b.ID, "parent", normalizedParent); cycle != nil {
		return fmt.Errorf("setting parent would create cycle: %v", cycle)
	}

	b.Parent = normalizedParent
	return nil
}

// validateAndAddBlocking validates and adds blocking relationships.
func (r *Resolver) validateAndAddBlocking(b *bean.Bean, targetIDs []string) error {
	for _, targetID := range targetIDs {
		// Normalise short ID to full ID
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)

		// Validate: cannot block itself
		if normalizedTargetID == b.ID {
			return fmt.Errorf("bean cannot block itself")
		}

		// Validate: target must exist
		if _, err := r.Core.Get(normalizedTargetID); err != nil {
			return fmt.Errorf("blocking target bean not found: %s", targetID)
		}

		// Check for cycles in both directions
		if cycle := r.Core.DetectCycle(b.ID, "blocking", normalizedTargetID); cycle != nil {
			return fmt.Errorf("adding blocking relationship would create cycle: %v", cycle)
		}
		if cycle := r.Core.DetectCycle(normalizedTargetID, "blocked_by", b.ID); cycle != nil {
			return fmt.Errorf("adding blocking relationship would create cycle: %v", cycle)
		}

		b.AddBlocking(normalizedTargetID)
	}
	return nil
}

// removeBlockingRelationships removes blocking relationships.
func (r *Resolver) removeBlockingRelationships(b *bean.Bean, targetIDs []string) {
	for _, targetID := range targetIDs {
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)
		b.RemoveBlocking(normalizedTargetID)
	}
}

// validateAndAddBlockedBy validates and adds blocked-by relationships.
func (r *Resolver) validateAndAddBlockedBy(b *bean.Bean, targetIDs []string) error {
	for _, targetID := range targetIDs {
		// Normalise short ID to full ID
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)

		// Validate: cannot be blocked by itself
		if normalizedTargetID == b.ID {
			return fmt.Errorf("bean cannot be blocked by itself")
		}

		// Validate: blocker must exist
		if _, err := r.Core.Get(normalizedTargetID); err != nil {
			return fmt.Errorf("blocker bean not found: %s", targetID)
		}

		// Check for cycles in both directions
		if cycle := r.Core.DetectCycle(normalizedTargetID, "blocking", b.ID); cycle != nil {
			return fmt.Errorf("adding blocked-by relationship would create cycle: %v", cycle)
		}
		if cycle := r.Core.DetectCycle(b.ID, "blocked_by", normalizedTargetID); cycle != nil {
			return fmt.Errorf("adding blocked-by relationship would create cycle: %v", cycle)
		}

		b.AddBlockedBy(normalizedTargetID)
	}
	return nil
}

// removeBlockedByRelationships removes blocked-by relationships.
func (r *Resolver) removeBlockedByRelationships(b *bean.Bean, targetIDs []string) {
	for _, targetID := range targetIDs {
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)
		b.RemoveBlockedBy(normalizedTargetID)
	}
}
