package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	progressJSON bool
)

type progressData struct {
	Total        int            `json:"total"`
	ByStatus     map[string]int `json:"by_status"`
	ByType       map[string]int `json:"by_type"`
	BlockedCount int            `json:"blocked_count"`
}

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show overall project progress",
	Long:  `Shows a summary of all beans by status and type, including blocked count.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &graph.Resolver{Core: core}
		allBeans, err := resolver.Query().Beans(context.Background(), nil)
		if err != nil {
			return cmdError(progressJSON, output.ErrValidation, "querying beans: %v", err)
		}

		byStatus := make(map[string]int)
		byType := make(map[string]int)
		for _, b := range allBeans {
			byStatus[b.Status]++
			byType[b.Type]++
		}

		// Count blocked beans
		isBlocked := true
		blockedFilter := &model.BeanFilter{
			IsBlocked:     &isBlocked,
			ExcludeStatus: []string{"completed", "scrapped"},
		}
		blockedBeans, err := resolver.Query().Beans(context.Background(), blockedFilter)
		if err != nil {
			return cmdError(progressJSON, output.ErrValidation, "querying blocked beans: %v", err)
		}

		data := progressData{
			Total:        len(allBeans),
			ByStatus:     byStatus,
			ByType:       byType,
			BlockedCount: len(blockedBeans),
		}

		if progressJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		}

		fmt.Println(ui.Title.Render(fmt.Sprintf("Project Progress (%d beans)", data.Total)))
		fmt.Println()

		// Status table
		fmt.Println(ui.Bold.Render("By Status"))
		maxCount := 0
		for _, count := range byStatus {
			if count > maxCount {
				maxCount = count
			}
		}
		barMaxWidth := 20
		for _, s := range cfg.StatusNames() {
			count := byStatus[s]
			if count == 0 {
				continue
			}
			sCfg := cfg.GetStatus(s)
			sColor := "gray"
			if sCfg != nil {
				sColor = sCfg.Color
			}
			isArchive := cfg.IsArchiveStatus(s)

			barWidth := barMaxWidth
			if maxCount > 0 {
				barWidth = count * barMaxWidth / maxCount
			}
			if barWidth == 0 {
				barWidth = 1
			}
			bar := strings.Repeat("█", barWidth)

			label := ui.RenderStatusTextWithColor(
				fmt.Sprintf("%-12s", s), sColor, isArchive,
			)
			fmt.Printf("  %s %s %d\n", label, ui.Success.Render(bar), count)
		}

		fmt.Println()

		// Type breakdown
		fmt.Println(ui.Bold.Render("By Type"))
		for _, t := range cfg.TypeNames() {
			count := byType[t]
			if count == 0 {
				continue
			}
			tCfg := cfg.GetType(t)
			tColor := "gray"
			if tCfg != nil {
				tColor = tCfg.Color
			}
			label := ui.RenderTypeText(fmt.Sprintf("%-12s", t), tColor)
			fmt.Printf("  %s %d\n", label, count)
		}

		fmt.Println()

		// Blocked count
		if data.BlockedCount > 0 {
			fmt.Println(ui.Warning.Render(fmt.Sprintf("Blocked: %d bean(s)", data.BlockedCount)))
		} else {
			fmt.Println(ui.Success.Render("No blocked beans"))
		}

		return nil
	},
}

func init() {
	progressCmd.Flags().BoolVar(&progressJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(progressCmd)
}
