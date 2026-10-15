package beangraph

import (
	"fmt"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
)

// CoreResolver implements the core bean GraphQL operations (CRUD, relationships,
// filtering). It depends only on beancore.Core and has no UI-specific dependencies
// (agents, worktrees, terminals, etc.).
//
// CLI commands use CoreResolver directly. The serve command's full GraphQL resolver
// embeds CoreResolver and adds UI-specific operations on top.
type CoreResolver struct {
	Core *beancore.Core
}

// validateETag checks if the provided ifMatch etag matches the bean's current etag.
// Returns an error if validation fails or if require_if_match is enabled and no etag provided.
func (r *CoreResolver) validateETag(b *bean.Bean, ifMatch *string) error {
	cfg := r.Core.Config()
	requireIfMatch := cfg != nil && cfg.Beans.RequireIfMatch

	// If require_if_match is enabled and no etag provided, reject
	if requireIfMatch && (ifMatch == nil || *ifMatch == "") {
		return &beancore.ETagRequiredError{}
	}

	// If ifMatch provided, validate it
	if ifMatch != nil && *ifMatch != "" {
		currentETag := b.ETag()
		if currentETag != *ifMatch {
			return &beancore.ETagMismatchError{Provided: *ifMatch, Current: currentETag}
		}
	}

	return nil
}

// cloneForUpdate returns a clone of the stored bean id for the caller to
// modify, and base, the ETag that Core.Update compares the stored bean against.
// It rejects a stale ifMatch before the caller changes anything.
func (r *CoreResolver) cloneForUpdate(id string, ifMatch *string) (*bean.Bean, string, error) {
	stored, err := r.Core.Get(id)
	if err != nil {
		return nil, "", err
	}
	b := stored.Clone()
	if err := r.validateETag(b, ifMatch); err != nil {
		return nil, "", err
	}
	return b, b.ETag(), nil
}

// ValidateAndSetParent validates and sets the parent relationship.
func (r *CoreResolver) ValidateAndSetParent(b *bean.Bean, parentID string) error {
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
	if cycle := r.Core.DetectCycleWith(b, b.ID, "parent", normalizedParent); cycle != nil {
		return fmt.Errorf("setting parent would create cycle: %v", cycle)
	}

	b.Parent = normalizedParent
	return nil
}

// ValidateAndAddBlocking validates and adds blocking relationships.
func (r *CoreResolver) ValidateAndAddBlocking(b *bean.Bean, targetIDs []string) error {
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
		if cycle := r.Core.DetectCycleWith(b, b.ID, "blocking", normalizedTargetID); cycle != nil {
			return fmt.Errorf("adding blocking relationship would create cycle: %v", cycle)
		}
		if cycle := r.Core.DetectCycleWith(b, normalizedTargetID, "blocked_by", b.ID); cycle != nil {
			return fmt.Errorf("adding blocking relationship would create cycle: %v", cycle)
		}

		b.AddBlocking(normalizedTargetID)
	}
	return nil
}

// RemoveBlockingRelationships removes blocking relationships.
func (r *CoreResolver) RemoveBlockingRelationships(b *bean.Bean, targetIDs []string) {
	for _, targetID := range targetIDs {
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)
		b.RemoveBlocking(normalizedTargetID)
	}
}

// ValidateAndAddBlockedBy validates and adds blocked-by relationships.
func (r *CoreResolver) ValidateAndAddBlockedBy(b *bean.Bean, targetIDs []string) error {
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
		if cycle := r.Core.DetectCycleWith(b, normalizedTargetID, "blocking", b.ID); cycle != nil {
			return fmt.Errorf("adding blocked-by relationship would create cycle: %v", cycle)
		}
		if cycle := r.Core.DetectCycleWith(b, b.ID, "blocked_by", normalizedTargetID); cycle != nil {
			return fmt.Errorf("adding blocked-by relationship would create cycle: %v", cycle)
		}

		b.AddBlockedBy(normalizedTargetID)
	}
	return nil
}

// RemoveBlockedByRelationships removes blocked-by relationships.
func (r *CoreResolver) RemoveBlockedByRelationships(b *bean.Bean, targetIDs []string) {
	for _, targetID := range targetIDs {
		normalizedTargetID, _ := r.Core.NormalizeID(targetID)
		b.RemoveBlockedBy(normalizedTargetID)
	}
}
