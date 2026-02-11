package cmd

import (
	"context"
	"fmt"

	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	nextJSON bool
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show the highest-priority bean ready to start",
	Long:  `Shows the single highest-priority bean that is not blocked and ready to start.`,
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
			return cmdError(nextJSON, output.ErrValidation, "querying beans: %v", err)
		}

		sortBeans(beans, "priority", cfg)

		if len(beans) == 0 {
			if nextJSON {
				return output.SuccessMessage("No beans ready to start")
			}
			fmt.Println(ui.Muted.Render("No beans ready to start."))
			return nil
		}

		b := beans[0]

		if nextJSON {
			return output.SuccessSingle(b)
		}

		showStyledBean(b)
		return nil
	},
}

func init() {
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(nextCmd)
}
