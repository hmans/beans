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
	scrapReason string
	scrapJSON   bool
)

var scrapCmd = &cobra.Command{
	Use:   "scrap <id> [id...]",
	Short: "Mark one or more beans as scrapped",
	Long:  `Sets the status of one or more beans to "scrapped". Requires a reason explaining why.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolver := &graph.Resolver{Core: core}
		var results []*result
		var errs []error

		for _, id := range args {
			b, _, err := resolveBean(resolver, id, scrapJSON)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if b.Status == "scrapped" {
				if !scrapJSON {
					fmt.Println(ui.Warning.Render("Already scrapped: ") + ui.ID.Render(b.ID))
				}
				results = append(results, &result{bean: b, warning: "already scrapped"})
				continue
			}

			status := "scrapped"
			appendText := "## Reasons for Scrapping\n\n" + scrapReason
			input := model.UpdateBeanInput{
				Status: &status,
				BodyMod: &model.BodyModification{
					Append: &appendText,
				},
			}

			b, err = resolver.Mutation().UpdateBean(context.Background(), b.ID, input)
			if err != nil {
				errs = append(errs, mutationError(scrapJSON, err))
				continue
			}

			results = append(results, &result{bean: b})
		}

		if len(errs) > 0 && len(results) == 0 {
			return errs[0]
		}

		return outputResults(results, errs, scrapJSON, "scrapped", "Scrapped")
	},
}

func init() {
	scrapCmd.Flags().StringVarP(&scrapReason, "reason", "m", "", "Reason for scrapping (required)")
	_ = scrapCmd.MarkFlagRequired("reason")
	scrapCmd.Flags().BoolVar(&scrapJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(scrapCmd)
}
