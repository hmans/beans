package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	startForce bool
	startJSON  bool
)

var startCmd = &cobra.Command{
	Use:   "start <id> [id...]",
	Short: "Start working on one or more beans",
	Long:  `Sets the status of one or more beans to "in-progress". Warns if a bean is blocked unless --force is used.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &graph.Resolver{Core: core}
		var results []*result
		var errs []error

		for _, id := range args {
			b, _, err := resolveBean(resolver, id, startJSON)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if b.Status == "in-progress" {
				if !startJSON {
					fmt.Println(ui.Warning.Render("Already in-progress: ") + ui.ID.Render(b.ID))
				}
				results = append(results, &result{bean: b, warning: "already in-progress"})
				continue
			}

			blockers := core.FindActiveBlockers(b.ID)
			if len(blockers) > 0 && !startForce {
				msg := formatBlockerMessage(b.ID, blockers)
				errs = append(errs, cmdError(startJSON, output.ErrValidation, "%s", msg))
				continue
			}

			if len(blockers) > 0 && startForce && !startJSON {
				fmt.Println(ui.Warning.Render("Warning: ") + ui.ID.Render(b.ID) + " is blocked, starting anyway")
			}

			status := "in-progress"
			input := model.UpdateBeanInput{
				Status: &status,
			}

			b, err = resolver.Mutation().UpdateBean(context.Background(), b.ID, input)
			if err != nil {
				errs = append(errs, mutationError(startJSON, err))
				continue
			}

			results = append(results, &result{bean: b})
		}

		if len(errs) > 0 && len(results) == 0 {
			return errs[0]
		}

		return outputResults(results, errs, startJSON, "started", "Started")
	},
}

func formatBlockerMessage(beanID string, blockers []*bean.Bean) string {
	var parts []string
	for _, bl := range blockers {
		parts = append(parts, fmt.Sprintf("%s (%s)", bl.ID, bl.Title))
	}
	return fmt.Sprintf("%s is blocked by: %s", beanID, strings.Join(parts, ", "))
}

func init() {
	startCmd.Flags().BoolVarP(&startForce, "force", "f", false, "Start even if blocked")
	startCmd.Flags().BoolVar(&startJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(startCmd)
}
