package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	blockedJSON  bool
	blockedQuiet bool
)

type blockedEntry struct {
	Bean     *bean.Bean   `json:"bean"`
	Blockers []*bean.Bean `json:"blockers"`
}

var blockedCmd = &cobra.Command{
	Use:   "blocked",
	Short: "List beans that are blocked",
	Long:  `Lists beans that are blocked by other beans, showing what blocks each one.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		isBlocked := true
		filter := &model.BeanFilter{
			IsBlocked:     &isBlocked,
			ExcludeStatus: []string{"completed", "scrapped"},
		}

		resolver := &graph.Resolver{Core: core}
		beans, err := resolver.Query().Beans(context.Background(), filter)
		if err != nil {
			return cmdError(blockedJSON, output.ErrValidation, "querying beans: %v", err)
		}

		sortBeans(beans, "", cfg)

		if blockedJSON {
			entries := make([]blockedEntry, 0, len(beans))
			for _, b := range beans {
				blockers := core.FindActiveBlockers(b.ID)
				entries = append(entries, blockedEntry{Bean: b, Blockers: blockers})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		if blockedQuiet {
			for _, b := range beans {
				fmt.Println(b.ID)
			}
			return nil
		}

		if len(beans) == 0 {
			fmt.Println(ui.Muted.Render("No blocked beans."))
			return nil
		}

		for _, b := range beans {
			blockers := core.FindActiveBlockers(b.ID)
			var blockerStrs []string
			for _, bl := range blockers {
				blockerStrs = append(blockerStrs, ui.ID.Render(bl.ID)+" "+ui.Muted.Render(bl.Title))
			}

			fmt.Println(ui.ID.Render(b.ID) + " " + b.Title)
			fmt.Println("  " + ui.Warning.Render("Blocked by: ") + strings.Join(blockerStrs, ", "))
		}

		return nil
	},
}

func init() {
	blockedCmd.Flags().BoolVar(&blockedJSON, "json", false, "Output as JSON")
	blockedCmd.Flags().BoolVarP(&blockedQuiet, "quiet", "q", false, "Only output IDs")
	rootCmd.AddCommand(blockedCmd)
}
