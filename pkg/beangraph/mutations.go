package beangraph

import (
	"context"
	"fmt"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/beancore"
)

// CreateBean creates a new bean from the given input.
func (r *CoreResolver) CreateBean(ctx context.Context, input model.CreateBeanInput) (*bean.Bean, error) {
	b := &bean.Bean{
		Slug:     bean.Slugify(input.Title),
		Title:    input.Title,
		Type:     "task", // default
		Blocking: []string{},
	}

	// Optional fields with defaults documented in schema
	if input.Type != nil {
		b.Type = *input.Type
	}
	if input.Status != nil {
		b.Status = *input.Status
	}
	if input.Priority != nil {
		b.Priority = *input.Priority
	}
	if input.Body != nil {
		b.Body = *input.Body
	}
	if len(input.Tags) > 0 {
		b.Tags = input.Tags
	}

	// Handle parent (with validation)
	if input.Parent != nil && *input.Parent != "" {
		// Normalise short ID to full ID
		parentID, _ := r.Core.NormalizeID(*input.Parent)
		if err := r.Core.ValidateParent(b, parentID); err != nil {
			return nil, err
		}
		b.Parent = parentID
	}

	// Handle blocking (with validation)
	if len(input.Blocking) > 0 {
		// Normalise short IDs to full IDs
		normalizedBlocking := make([]string, len(input.Blocking))
		for i, id := range input.Blocking {
			normalizedBlocking[i], _ = r.Core.NormalizeID(id)
			// Verify target exists
			if _, err := r.Core.Get(normalizedBlocking[i]); err != nil {
				return nil, fmt.Errorf("target bean not found: %s", id)
			}
		}
		b.Blocking = normalizedBlocking
	}

	// Handle blocked_by (with cycle validation)
	if len(input.BlockedBy) > 0 {
		// Normalise short IDs to full IDs
		normalizedBlockedBy := make([]string, len(input.BlockedBy))
		for i, id := range input.BlockedBy {
			normalizedBlockedBy[i], _ = r.Core.NormalizeID(id)
			// Verify blocker exists
			if _, err := r.Core.Get(normalizedBlockedBy[i]); err != nil {
				return nil, fmt.Errorf("blocker bean not found: %s", id)
			}
		}
		// Check for cycles with blocking relationships
		// (new bean being blocked_by X means X→newBean, check if newBean→X exists via blocking)
		for _, blockerID := range normalizedBlockedBy {
			for _, blockingID := range b.Blocking {
				if blockerID == blockingID {
					return nil, fmt.Errorf("would create cycle: new bean both blocks and is blocked by %s", blockerID)
				}
			}
		}
		b.BlockedBy = normalizedBlockedBy
	}

	// Handle custom prefix - pre-generate ID if prefix is provided
	if input.Prefix != nil && *input.Prefix != "" {
		idLength := 4 // default
		if cfg := r.Core.Config(); cfg != nil && cfg.Beans.IDLength > 0 {
			idLength = cfg.Beans.IDLength
		}
		id, err := bean.NewID(*input.Prefix, idLength)
		if err != nil {
			return nil, fmt.Errorf("generating bean ID: %w", err)
		}
		b.ID = id
	}

	if err := r.Core.Create(b); err != nil {
		return nil, err
	}

	return b, nil
}

// UpdateBean updates an existing bean.
func (r *CoreResolver) UpdateBean(ctx context.Context, id string, input model.UpdateBeanInput, opts ...beancore.UpdateOption) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err != nil {
		return nil, err
	}

	// Validate body and bodyMod are mutually exclusive
	if input.Body != nil && input.BodyMod != nil {
		return nil, fmt.Errorf("cannot specify both body and bodyMod")
	}

	// Validate tags and addTags/removeTags are mutually exclusive
	if input.Tags != nil && (input.AddTags != nil || input.RemoveTags != nil) {
		return nil, fmt.Errorf("cannot specify both tags and addTags/removeTags")
	}

	// Update fields if provided
	if input.Title != nil {
		b.Title = *input.Title
	}
	if input.Status != nil {
		b.Status = *input.Status
	}
	if input.Type != nil {
		b.Type = *input.Type
	}
	if input.Priority != nil {
		b.Priority = *input.Priority
	}
	if input.Order != nil {
		b.Order = *input.Order
	}
	if input.Body != nil {
		b.Body = *input.Body
	} else if input.BodyMod != nil {
		// Apply body modifications
		workingBody := b.Body

		// Apply replacements sequentially
		if input.BodyMod.Replace != nil {
			for i, replaceOp := range input.BodyMod.Replace {
				newBody, err := bean.ReplaceOnce(workingBody, replaceOp.Old, replaceOp.New)
				if err != nil {
					return nil, fmt.Errorf("replacement %d failed: %w", i, err)
				}
				workingBody = newBody
			}
		}

		// Apply append if provided
		if input.BodyMod.Append != nil && *input.BodyMod.Append != "" {
			workingBody = bean.AppendWithSeparator(workingBody, *input.BodyMod.Append)
		}

		b.Body = workingBody
	}
	// Handle tags
	if input.Tags != nil {
		b.Tags = input.Tags
	} else if input.AddTags != nil || input.RemoveTags != nil {
		// Build a set of current tags
		tagSet := make(map[string]bool)
		for _, tag := range b.Tags {
			tagSet[tag] = true
		}

		// Add new tags
		if input.AddTags != nil {
			for _, tag := range input.AddTags {
				tagSet[tag] = true
			}
		}

		// Remove tags
		if input.RemoveTags != nil {
			for _, tag := range input.RemoveTags {
				delete(tagSet, tag)
			}
		}

		// Convert back to slice
		newTags := make([]string, 0, len(tagSet))
		for tag := range tagSet {
			newTags = append(newTags, tag)
		}
		b.Tags = newTags
	}

	// Handle parent relationship
	if input.Parent != nil {
		if err := r.ValidateAndSetParent(b, *input.Parent); err != nil {
			return nil, err
		}
	}

	// Handle blocking relationships
	if input.AddBlocking != nil {
		if err := r.ValidateAndAddBlocking(b, input.AddBlocking); err != nil {
			return nil, err
		}
	}
	if input.RemoveBlocking != nil {
		r.RemoveBlockingRelationships(b, input.RemoveBlocking)
	}

	// Handle blocked-by relationships
	if input.AddBlockedBy != nil {
		if err := r.ValidateAndAddBlockedBy(b, input.AddBlockedBy); err != nil {
			return nil, err
		}
	}
	if input.RemoveBlockedBy != nil {
		r.RemoveBlockedByRelationships(b, input.RemoveBlockedBy)
	}

	// ETag validation now happens inside Update() under write lock.
	// If the bean is linked to a worktree, Core auto-routes the write there.
	if err := r.Core.Update(b, input.IfMatch, opts...); err != nil {
		return nil, err
	}

	return b, nil
}

// DeleteBean removes a bean and its incoming links.
func (r *CoreResolver) DeleteBean(ctx context.Context, id string) (bool, error) {
	// Verify bean exists
	_, err := r.Core.Get(id)
	if err != nil {
		return false, err
	}

	// Remove incoming links first
	if _, err := r.Core.RemoveLinksTo(id); err != nil {
		return false, err
	}

	// Delete the bean
	if err := r.Core.Delete(id); err != nil {
		return false, err
	}

	return true, nil
}

// SetParent sets or clears the parent of a bean.
func (r *CoreResolver) SetParent(ctx context.Context, id string, parentID *string, ifMatch *string) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err != nil {
		return nil, err
	}

	newParent := ""
	if parentID != nil {
		// Normalise short ID to full ID
		newParent, _ = r.Core.NormalizeID(*parentID)
	}

	// Validate parent type hierarchy
	if newParent != "" {
		if err := r.Core.ValidateParent(b, newParent); err != nil {
			return nil, err
		}
		// Check for cycles
		if cycle := r.Core.DetectCycle(b.ID, "parent", newParent); cycle != nil {
			return nil, fmt.Errorf("would create cycle: %v", cycle)
		}
	}

	b.Parent = newParent
	// ETag validation now happens inside Update() under write lock
	if err := r.Core.Update(b, ifMatch); err != nil {
		return nil, err
	}
	return b, nil
}

// AddBlocking adds a blocking relationship.
func (r *CoreResolver) AddBlocking(ctx context.Context, id string, targetID string, ifMatch *string) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err != nil {
		return nil, err
	}

	// Normalise short ID to full ID
	normalizedTargetID, _ := r.Core.NormalizeID(targetID)

	if normalizedTargetID == b.ID {
		return nil, fmt.Errorf("bean cannot block itself")
	}

	// Check target exists
	if _, err := r.Core.Get(normalizedTargetID); err != nil {
		return nil, fmt.Errorf("target bean not found: %s", targetID)
	}

	// Check for cycles in both directions
	if cycle := r.Core.DetectCycle(b.ID, "blocking", normalizedTargetID); cycle != nil {
		return nil, fmt.Errorf("would create cycle: %v", cycle)
	}
	if cycle := r.Core.DetectCycle(normalizedTargetID, "blocked_by", b.ID); cycle != nil {
		return nil, fmt.Errorf("would create cycle: %v", cycle)
	}

	b.AddBlocking(normalizedTargetID)
	if err := r.Core.Update(b, ifMatch); err != nil {
		return nil, err
	}
	return b, nil
}

// RemoveBlocking removes a blocking relationship.
func (r *CoreResolver) RemoveBlocking(ctx context.Context, id string, targetID string, ifMatch *string) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err != nil {
		return nil, err
	}

	normalizedTargetID, _ := r.Core.NormalizeID(targetID)

	b.RemoveBlocking(normalizedTargetID)
	if err := r.Core.Update(b, ifMatch); err != nil {
		return nil, err
	}
	return b, nil
}

// AddBlockedBy adds a blocked-by relationship.
func (r *CoreResolver) AddBlockedBy(ctx context.Context, id string, targetID string, ifMatch *string) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err != nil {
		return nil, err
	}

	normalizedTargetID, _ := r.Core.NormalizeID(targetID)

	if normalizedTargetID == b.ID {
		return nil, fmt.Errorf("bean cannot be blocked by itself")
	}

	// Check target exists
	if _, err := r.Core.Get(normalizedTargetID); err != nil {
		return nil, fmt.Errorf("blocker bean not found: %s", targetID)
	}

	// Check for cycles in both directions
	if cycle := r.Core.DetectCycle(normalizedTargetID, "blocking", b.ID); cycle != nil {
		return nil, fmt.Errorf("would create cycle: %v", cycle)
	}
	if cycle := r.Core.DetectCycle(b.ID, "blocked_by", normalizedTargetID); cycle != nil {
		return nil, fmt.Errorf("would create cycle: %v", cycle)
	}

	b.AddBlockedBy(normalizedTargetID)
	if err := r.Core.Update(b, ifMatch); err != nil {
		return nil, err
	}
	return b, nil
}

// RemoveBlockedBy removes a blocked-by relationship.
func (r *CoreResolver) RemoveBlockedBy(ctx context.Context, id string, targetID string, ifMatch *string) (*bean.Bean, error) {
	b, err := r.Core.Get(id)
	if err != nil {
		return nil, err
	}

	normalizedTargetID, _ := r.Core.NormalizeID(targetID)

	b.RemoveBlockedBy(normalizedTargetID)
	if err := r.Core.Update(b, ifMatch); err != nil {
		return nil, err
	}
	return b, nil
}

// ArchiveBean archives a bean.
func (r *CoreResolver) ArchiveBean(ctx context.Context, id string) (bool, error) {
	if err := r.Core.Archive(id); err != nil {
		return false, err
	}
	return true, nil
}
