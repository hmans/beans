package cmd

import (
	"context"
	"fmt"

	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	completeSummary string
	completeJSON    bool
)

var completeCmd = &cobra.Command{
	Use:   "complete <id> [id...]",
	Short: "Mark one or more beans as completed",
	Long:  `Sets the status of one or more beans to "completed". Optionally appends a summary of changes.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &graph.Resolver{Core: core}
		var results []*result
		var errs []error

		for _, id := range args {
			b, _, err := resolveBean(resolver, id, completeJSON)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if b.Status == "completed" {
				if !completeJSON {
					fmt.Println(ui.Warning.Render("Already completed: ") + ui.ID.Render(b.ID))
				}
				results = append(results, &result{bean: b, warning: "already completed"})
				continue
			}

			status := "completed"
			input := model.UpdateBeanInput{
				Status: &status,
			}

			if completeSummary != "" {
				appendText := "## Summary of Changes\n\n" + completeSummary
				input.BodyMod = &model.BodyModification{
					Append: &appendText,
				}
			}

			b, err = resolver.Mutation().UpdateBean(context.Background(), b.ID, input)
			if err != nil {
				errs = append(errs, mutationError(completeJSON, err))
				continue
			}

			results = append(results, &result{bean: b})
		}

		if len(errs) > 0 && len(results) == 0 {
			return errs[0]
		}

		return outputResults(results, errs, completeJSON, "completed", "Completed")
	},
}

func init() {
	completeCmd.Flags().StringVarP(&completeSummary, "summary", "m", "", "Summary of changes to append")
	completeCmd.Flags().BoolVar(&completeJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(completeCmd)
}
