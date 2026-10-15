package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	nextJSON       bool
	nextAll        bool
	nextInProgress bool
	nextLimit      int
)

// nextScope says where the fronts of `beans next` come from.
type nextScope struct {
	Kind  string `json:"kind"` // "focus", "in-progress-fronts" or "no-fronts"
	Focus string `json:"focus,omitempty"`
	File  string `json:"file,omitempty"`
}

type nextJSONRow struct {
	output.Bean
	Lane   bean.NextLane `json:"lane"`
	Reason string        `json:"reason"`
	Front  string        `json:"front,omitempty"`
}

type nextJSONResult struct {
	Scope   nextScope     `json:"scope"`
	Hint    string        `json:"hint,omitempty"`
	Warning string        `json:"warning,omitempty"`
	Rows    []nextJSONRow `json:"rows"`
}

// resolvedFocus is the focus setting checked against the beans.
type resolvedFocus struct {
	bean    *bean.Bean // nil: in-progress fronts apply
	id      string     // configured ID, also when invalid
	source  string     // file the ID came from
	problem string     // why id cannot be the focus, or ""
}

func resolveFocus(all []*bean.Bean) resolvedFocus {
	id, source := cfg.Focus()
	if id == "" {
		return resolvedFocus{}
	}
	full, _ := core.NormalizeID(id)
	r := resolvedFocus{id: full, source: source, problem: bean.FocusProblem(full, all)}
	if r.problem == "" {
		r.bean, _ = core.Get(full)
	}
	return r
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show a ranked list of beans to start next",
	Long: `Ranks the beans available to start, in lanes:

  interrupt  bugs with priority high or critical, anywhere in the tree
  focus      beans under a front
  refine     drafts under a front, to refine before they can start
  rest       everything else (only with --all, or when there is no front)

A front is the focus set with "beans focus set", or else every
in-progress milestone or epic.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if nextAll && nextInProgress {
			return fmt.Errorf("--all and --in-progress are mutually exclusive")
		}
		if nextLimit < 0 {
			return fmt.Errorf("--limit must be 0 (unlimited) or positive")
		}

		resolver := &beangraph.CoreResolver{Core: core}
		ctx := context.Background()
		all, err := resolver.Beans(ctx, nil)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}
		candidates, err := nextCandidates(ctx, resolver)
		if err != nil {
			return err
		}

		focus := resolveFocus(all)
		result := bean.RankNext(candidates, all, bean.NextOptions{
			Focus:         focus.bean,
			InProgress:    nextInProgress,
			PriorityNames: cfg.PriorityNames(),
			TypeNames:     cfg.TypeNames(),
		})

		// Without any front every ready bean is in the rest lane, so hiding
		// that lane would hide all work.
		noFronts := len(result.Fronts) == 0
		rows := result.Rows
		if !nextAll && !noFronts {
			kept := rows[:0:0]
			for _, r := range rows {
				if r.Lane != bean.LaneRest {
					kept = append(kept, r)
				}
			}
			rows = kept
		}
		if nextLimit > 0 && len(rows) > nextLimit {
			rows = rows[:nextLimit]
		}

		scope := nextScope{Kind: "in-progress-fronts"}
		if noFronts {
			scope = nextScope{Kind: "no-fronts"}
		}
		if focus.bean != nil {
			scope = nextScope{Kind: "focus", Focus: focus.bean.ID, File: focus.source}
		}
		var warning, hint string
		if focus.problem != "" {
			warning = fmt.Sprintf("focus %s invalid: %s", focus.id, focus.problem)
		}
		if focus.bean == nil && len(result.Fronts) > 1 {
			hint = fmt.Sprintf("%d active fronts – consider \"beans focus set\"", len(result.Fronts))
		}

		if nextJSON {
			return writeNextJSON(cmd.OutOrStdout(), resolver, scope, hint, warning, rows)
		}
		writeNextText(cmd.OutOrStdout(), scope, hint, warning, rows)
		return nil
	},
}

// nextCandidates returns the beans RankNext ranks: in-progress beans with
// --in-progress, otherwise the --ready set plus unblocked drafts.
func nextCandidates(ctx context.Context, resolver *beangraph.CoreResolver) ([]*bean.Bean, error) {
	excludeTerminal := true
	notBlocked := false
	var filters []*model.BeanFilter
	if nextInProgress {
		filters = append(filters, &model.BeanFilter{
			Status:                  []string{"in-progress"},
			ExcludeImplicitTerminal: &excludeTerminal,
		})
	} else {
		ready := &model.BeanFilter{}
		beangraph.AddReadyFilter(ready)
		filters = append(filters, ready, &model.BeanFilter{
			Status:                  []string{"draft"},
			IsBlocked:               &notBlocked,
			ExcludeImplicitTerminal: &excludeTerminal,
		})
	}
	var candidates []*bean.Bean
	for _, f := range filters {
		beans, err := resolver.Beans(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("querying beans: %w", err)
		}
		candidates = append(candidates, beans...)
	}
	return candidates, nil
}

func (s nextScope) String() string {
	switch s.Kind {
	case "focus":
		return fmt.Sprintf("focus %s (%s)", s.Focus, s.File)
	case "no-fronts":
		return "no fronts – all ready beans"
	}
	return "in-progress fronts"
}

func writeNextText(w io.Writer, scope nextScope, hint, warning string, rows []bean.NextRow) {
	header := "scope: " + scope.String()
	if warning != "" {
		header += " (" + warning + ")"
	}
	fmt.Fprintln(w, ui.Muted.Render(header))
	if hint != "" {
		fmt.Fprintln(w, ui.Warning.Render(hint))
	}
	fmt.Fprintln(w)

	if len(rows) == 0 {
		if nextInProgress {
			fmt.Fprintln(w, ui.Muted.Render("Nothing in progress."))
			return
		}
		fmt.Fprintln(w, ui.Muted.Render("Nothing ready. See blocked beans with \"beans list --is-blocked\", or all beans with \"beans list\"."))
		return
	}

	idWidth := 0
	for _, r := range rows {
		idWidth = max(idWidth, len(r.Bean.ID))
	}
	for _, r := range rows {
		lane := fmt.Sprintf("%-11s", r.Lane)
		id := fmt.Sprintf("%-*s", idWidth, r.Bean.ID)
		fmt.Fprintf(w, "%s %s  %s  %s\n",
			ui.Muted.Render(lane), ui.ID.Render(id), r.Bean.Title, ui.Muted.Render("– "+r.Reason))
	}
}

func writeNextJSON(w io.Writer, resolver *beangraph.CoreResolver, scope nextScope, hint, warning string, rows []bean.NextRow) error {
	beans := make([]*bean.Bean, len(rows))
	for i, r := range rows {
		beans[i] = r.Bean
	}
	views := jsonBeans(resolver, beans, false)
	out := nextJSONResult{Scope: scope, Hint: hint, Warning: warning, Rows: make([]nextJSONRow, len(rows))}
	for i, r := range rows {
		out.Rows[i] = nextJSONRow{Bean: views[i], Lane: r.Lane, Reason: r.Reason}
		if r.Front != nil {
			out.Rows[i].Front = r.Front.ID
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func RegisterNextCmd(root *cobra.Command) {
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output as JSON")
	nextCmd.Flags().BoolVar(&nextAll, "all", false, "Also show beans outside every front (rest lane; always shown when there is no front)")
	nextCmd.Flags().BoolVar(&nextInProgress, "in-progress", false, "List in-progress beans to resume instead")
	nextCmd.Flags().IntVar(&nextLimit, "limit", 10, "Maximum number of rows (0 = unlimited)")
	root.AddCommand(nextCmd)
}
