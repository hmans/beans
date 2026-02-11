package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	readyJSON  bool
	readyQuiet bool
	readySort  string
	readyFull  bool
)

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List beans ready to start",
	Long:  `Lists beans that are not blocked and not in-progress, completed, scrapped, or draft status.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		isBlocked := false
		filter := &model.BeanFilter{
			IsBlocked:     &isBlocked,
			ExcludeStatus: []string{"in-progress", "completed", "scrapped", "draft"},
		}

		resolver := &graph.Resolver{Core: core}
		beans, err := resolver.Query().Beans(context.Background(), filter)
		if err != nil {
			return cmdError(readyJSON, output.ErrValidation, "querying beans: %v", err)
		}

		sortBeans(beans, readySort, cfg)

		if readyJSON {
			if !readyFull {
				for _, b := range beans {
					b.Body = ""
				}
			}
			return output.SuccessMultiple(beans)
		}

		if readyQuiet {
			for _, b := range beans {
				fmt.Println(b.ID)
			}
			return nil
		}

		// Tree view (same as list command)
		allBeans, err := resolver.Query().Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying all beans for tree: %w", err)
		}

		sortFn := func(b []*bean.Bean) {
			sortBeans(b, readySort, cfg)
		}

		tree := ui.BuildTree(beans, allBeans, sortFn)

		if len(tree) == 0 {
			fmt.Println(ui.Muted.Render("No beans ready to start."))
			return nil
		}

		maxIDWidth := 2
		for _, b := range allBeans {
			if len(b.ID) > maxIDWidth {
				maxIDWidth = len(b.ID)
			}
		}
		maxIDWidth += 2

		hasTags := false
		for _, b := range beans {
			if len(b.Tags) > 0 {
				hasTags = true
				break
			}
		}

		termWidth := 80
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			termWidth = w
		}

		fmt.Print(ui.RenderTree(tree, cfg, maxIDWidth, hasTags, termWidth))
		return nil
	},
}

func init() {
	readyCmd.Flags().BoolVar(&readyJSON, "json", false, "Output as JSON")
	readyCmd.Flags().BoolVarP(&readyQuiet, "quiet", "q", false, "Only output IDs")
	readyCmd.Flags().StringVar(&readySort, "sort", "", "Sort by: created, updated, status, priority, id")
	readyCmd.Flags().BoolVar(&readyFull, "full", false, "Include bean body in JSON output")
	rootCmd.AddCommand(readyCmd)
}
