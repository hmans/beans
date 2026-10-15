package bean

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// NextLane groups the rows of `beans next`; lanes are listed in rank order.
type NextLane string

const (
	LaneInterrupt  NextLane = "interrupt"
	LaneFocus      NextLane = "focus"
	LaneRefine     NextLane = "refine"
	LaneRest       NextLane = "rest"
	LaneInProgress NextLane = "in-progress"
)

var laneRank = map[NextLane]int{
	LaneInProgress: 0,
	LaneInterrupt:  1,
	LaneFocus:      2,
	LaneRefine:     3,
	LaneRest:       4,
}

// NextRow is one ranked bean. Front is the nearest front ancestor, or nil.
type NextRow struct {
	Bean   *Bean
	Lane   NextLane
	Front  *Bean
	Reason string
}

// NextOptions configures RankNext. A nil Focus makes every in-progress
// milestone or epic a front. PriorityNames and TypeNames give rank order,
// most important first.
type NextOptions struct {
	Focus         *Bean
	InProgress    bool
	PriorityNames []string
	TypeNames     []string
}

// NextResult holds the fronts in use and the ranked rows.
type NextResult struct {
	Fronts []*Bean
	Rows   []NextRow
}

func isContainerType(t string) bool {
	return t == "milestone" || t == "epic"
}

func isClosedStatus(s string) bool {
	return s == "completed" || s == "scrapped"
}

// RankNext ranks candidates for `beans next`. The caller selects candidates
// by status and blocking; RankNext drops milestones, epics and beans with an
// open child, assigns lanes, and orders the rows. all is every bean, used to
// resolve parents and children.
func RankNext(candidates, all []*Bean, opts NextOptions) NextResult {
	byID := make(map[string]*Bean, len(all))
	hasOpenChild := make(map[string]bool)
	for _, b := range all {
		byID[b.ID] = b
	}
	for _, b := range all {
		if b.Parent != "" && !isClosedStatus(b.Status) {
			hasOpenChild[b.Parent] = true
		}
	}

	var fronts []*Bean
	if opts.Focus != nil {
		fronts = []*Bean{opts.Focus}
	} else {
		for _, b := range all {
			if isContainerType(b.Type) && b.Status == "in-progress" && closedAncestor(b, byID) == nil {
				fronts = append(fronts, b)
			}
		}
		sort.Slice(fronts, func(i, j int) bool { return fronts[i].ID < fronts[j].ID })
	}
	isFront := make(map[string]bool, len(fronts))
	for _, f := range fronts {
		isFront[f.ID] = true
	}

	nearestFront := func(b *Bean) *Bean {
		seen := map[string]bool{b.ID: true}
		for id := b.Parent; id != "" && !seen[id]; {
			seen[id] = true
			p, ok := byID[id]
			if !ok {
				return nil
			}
			if isFront[p.ID] {
				return p
			}
			id = p.Parent
		}
		return nil
	}

	var rows []NextRow
	for _, b := range candidates {
		if isContainerType(b.Type) || hasOpenChild[b.ID] {
			continue
		}
		front := nearestFront(b)
		row := NextRow{Bean: b, Front: front}
		switch {
		case opts.InProgress:
			row.Lane = LaneInProgress
		case b.Status == "draft" && front != nil:
			row.Lane = LaneRefine
		case b.Status != "draft" && isInterruptBug(b):
			row.Lane = LaneInterrupt
		case b.Status != "draft" && front != nil:
			row.Lane = LaneFocus
		default:
			row.Lane = LaneRest
		}
		row.Reason = nextReason(row, opts.Focus != nil)
		rows = append(rows, row)
	}

	r := newNextRanker(opts.PriorityNames, opts.TypeNames)
	sort.SliceStable(rows, func(i, j int) bool { return r.less(rows[i], rows[j]) })

	return NextResult{Fronts: fronts, Rows: rows}
}

func isInterruptBug(b *Bean) bool {
	return b.Type == "bug" && (b.Priority == "high" || b.Priority == "critical")
}

func nextReason(row NextRow, explicitFocus bool) string {
	var under string
	if row.Front != nil {
		why := row.Front.Status
		if explicitFocus {
			why = "focus"
		}
		under = fmt.Sprintf("under %s %s (%s)", row.Front.Type, row.Front.ID, why)
	}
	switch row.Lane {
	case LaneInterrupt:
		return strings.TrimSpace(fmt.Sprintf("%s bug %s", row.Bean.Priority, under))
	case LaneRefine:
		return "draft to refine " + under
	case LaneInProgress:
		if under == "" {
			return "in progress, no front"
		}
		return "in progress " + under
	case LaneRest:
		if row.Bean.Status == "draft" {
			return "draft, no front"
		}
		return "no front"
	default:
		return under
	}
}

type nextRanker struct {
	priority map[string]int
	typ      map[string]int
	normal   int
}

func newNextRanker(priorityNames, typeNames []string) nextRanker {
	r := nextRanker{
		priority: make(map[string]int, len(priorityNames)),
		typ:      make(map[string]int, len(typeNames)),
		normal:   len(priorityNames),
	}
	for i, p := range priorityNames {
		r.priority[p] = i
		if p == "normal" {
			r.normal = i
		}
	}
	for i, t := range typeNames {
		r.typ[t] = i
	}
	return r
}

func (r nextRanker) priorityOf(b *Bean) int {
	if b.Priority == "" {
		return r.normal
	}
	if i, ok := r.priority[b.Priority]; ok {
		return i
	}
	return len(r.priority)
}

func (r nextRanker) typeOf(b *Bean) int {
	if i, ok := r.typ[b.Type]; ok {
		return i
	}
	return len(r.typ)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// cmpOrder sorts beans with a manual order first, then by the order string.
func cmpOrder(a, b string) int {
	if (a != "") != (b != "") {
		if a != "" {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// cmpTime sorts nil last; newestFirst reverses the order of set times.
func cmpTime(a, b *time.Time, newestFirst bool) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	c := a.Compare(*b)
	if newestFirst {
		return -c
	}
	return c
}

// cmpFront orders rows by their front. A row with a front sorts before one
// without, and rows of the same front stay together.
func (r nextRanker) cmpFront(a, b *Bean) int {
	if (a != nil) != (b != nil) {
		if a != nil {
			return -1
		}
		return 1
	}
	if a == nil || a.ID == b.ID {
		return 0
	}
	if c := cmpInt(r.priorityOf(a), r.priorityOf(b)); c != 0 {
		return c
	}
	if c := cmpOrder(a.Order, b.Order); c != 0 {
		return c
	}
	if c := cmpTime(a.UpdatedAt, b.UpdatedAt, true); c != 0 {
		return c
	}
	return strings.Compare(a.ID, b.ID)
}

func (r nextRanker) less(a, b NextRow) bool {
	if c := cmpInt(laneRank[a.Lane], laneRank[b.Lane]); c != 0 {
		return c < 0
	}
	beanPriority := cmpInt(r.priorityOf(a.Bean), r.priorityOf(b.Bean))
	if a.Lane == LaneInterrupt && beanPriority != 0 {
		return beanPriority < 0
	}
	if c := r.cmpFront(a.Front, b.Front); c != 0 {
		return c < 0
	}
	if beanPriority != 0 {
		return beanPriority < 0
	}
	if c := cmpInt(r.typeOf(a.Bean), r.typeOf(b.Bean)); c != 0 {
		return c < 0
	}
	if c := cmpOrder(a.Bean.Order, b.Bean.Order); c != 0 {
		return c < 0
	}
	if c := cmpTime(a.Bean.CreatedAt, b.Bean.CreatedAt, false); c != 0 {
		return c < 0
	}
	return a.Bean.ID < b.Bean.ID
}

// FocusProblem reports why the bean with id cannot be the focus of
// `beans next`, or "" when it can. all is every bean.
func FocusProblem(id string, all []*Bean) string {
	byID := make(map[string]*Bean, len(all))
	for _, b := range all {
		byID[b.ID] = b
	}
	b, ok := byID[id]
	if !ok {
		return "not found"
	}
	if !isContainerType(b.Type) {
		return fmt.Sprintf("is a %s, not a milestone or epic", b.Type)
	}
	if isClosedStatus(b.Status) {
		return b.Status
	}
	if p := closedAncestor(b, byID); p != nil {
		return fmt.Sprintf("ancestor %s is %s", p.ID, p.Status)
	}
	return ""
}

// closedAncestor returns the nearest completed or scrapped ancestor of b,
// or nil.
func closedAncestor(b *Bean, byID map[string]*Bean) *Bean {
	seen := map[string]bool{b.ID: true}
	for pid := b.Parent; pid != "" && !seen[pid]; {
		seen[pid] = true
		p, ok := byID[pid]
		if !ok {
			return nil
		}
		if isClosedStatus(p.Status) {
			return p
		}
		pid = p.Parent
	}
	return nil
}
