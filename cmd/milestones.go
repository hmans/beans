package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	milestonesJSON        bool
	milestonesIncludeDone bool
)

type milestoneInfo struct {
	Milestone  *bean.Bean        `json:"milestone"`
	Total      int               `json:"total"`
	ByStatus   map[string]int    `json:"by_status"`
	Completion float64           `json:"completion_pct"`
}

var milestonesCmd = &cobra.Command{
	Use:   "milestones",
	Short: "Show milestone progress",
	Long:  `Shows all milestones with child counts and completion percentages.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &graph.Resolver{Core: core}
		allBeans, err := resolver.Query().Beans(context.Background(), nil)
		if err != nil {
			return cmdError(milestonesJSON, output.ErrValidation, "querying beans: %v", err)
		}

		// Build children index
		children := make(map[string][]*bean.Bean)
		for _, b := range allBeans {
			if b.Parent != "" {
				children[b.Parent] = append(children[b.Parent], b)
			}
		}

		// Collect all descendants (not just direct children)
		var collectDescendants func(id string) []*bean.Bean
		collectDescendants = func(id string) []*bean.Bean {
			var all []*bean.Bean
			for _, child := range children[id] {
				all = append(all, child)
				all = append(all, collectDescendants(child.ID)...)
			}
			return all
		}

		// Find milestones
		var milestones []*bean.Bean
		for _, b := range allBeans {
			if b.Type != "milestone" {
				continue
			}
			if !milestonesIncludeDone && cfg.IsArchiveStatus(b.Status) {
				continue
			}
			milestones = append(milestones, b)
		}

		sortByStatusThenCreated(milestones, cfg)

		var infos []milestoneInfo
		for _, m := range milestones {
			descendants := collectDescendants(m.ID)
			byStatus := make(map[string]int)
			for _, d := range descendants {
				byStatus[d.Status]++
			}
			total := len(descendants)
			completed := byStatus["completed"]
			pct := 0.0
			if total > 0 {
				pct = float64(completed) / float64(total) * 100
			}
			infos = append(infos, milestoneInfo{
				Milestone:  m,
				Total:      total,
				ByStatus:   byStatus,
				Completion: pct,
			})
		}

		if milestonesJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(infos)
		}

		if len(infos) == 0 {
			fmt.Println(ui.Muted.Render("No milestones found."))
			return nil
		}

		for i, info := range infos {
			if i > 0 {
				fmt.Println()
			}

			statusCfg := cfg.GetStatus(info.Milestone.Status)
			statusColor := "gray"
			if statusCfg != nil {
				statusColor = statusCfg.Color
			}
			isArchive := cfg.IsArchiveStatus(info.Milestone.Status)

			fmt.Println(
				ui.ID.Render(info.Milestone.ID) + " " +
					ui.RenderStatusWithColor(info.Milestone.Status, statusColor, isArchive) + " " +
					ui.Title.Render(info.Milestone.Title),
			)

			if info.Total == 0 {
				fmt.Println("  " + ui.Muted.Render("No children"))
				continue
			}

			// Progress bar
			barWidth := 30
			filled := int(info.Completion / 100 * float64(barWidth))
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			fmt.Printf("  %s %.0f%% (%d/%d)\n",
				ui.Success.Render(bar),
				info.Completion,
				info.ByStatus["completed"],
				info.Total,
			)

			// Status breakdown
			var parts []string
			for _, s := range cfg.StatusNames() {
				if count, ok := info.ByStatus[s]; ok && count > 0 {
					sCfg := cfg.GetStatus(s)
					sColor := "gray"
					if sCfg != nil {
						sColor = sCfg.Color
					}
					sArchive := cfg.IsArchiveStatus(s)
					parts = append(parts, ui.RenderStatusTextWithColor(s, sColor, sArchive)+fmt.Sprintf(": %d", count))
				}
			}
			if len(parts) > 0 {
				fmt.Println("  " + strings.Join(parts, "  "))
			}
		}

		return nil
	},
}

func init() {
	milestonesCmd.Flags().BoolVar(&milestonesJSON, "json", false, "Output as JSON")
	milestonesCmd.Flags().BoolVar(&milestonesIncludeDone, "include-done", false, "Include completed/scrapped milestones")
	rootCmd.AddCommand(milestonesCmd)
}
