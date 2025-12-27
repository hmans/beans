package cmd

import (
	"fmt"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/output"
	"github.com/spf13/cobra"
)

var archiveJSON bool

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Move completed/scrapped beans to the archive",
	Long: `Moves all beans with status "completed" or "scrapped" to the archive directory (.beans/archive/).
Archived beans are preserved for project memory but excluded from normal queries.
Use --with-archived flag on other commands to include archived beans.

If other beans reference beans being archived (as parent or via blocking), those
references will be automatically removed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		allBeans := core.All()

		// Find beans with any archive status
		var archiveBeans []*bean.Bean
		archiveSet := make(map[string]bool)
		for _, b := range allBeans {
			if cfg.IsArchiveStatus(b.Status) {
				archiveBeans = append(archiveBeans, b)
				archiveSet[b.ID] = true
			}
		}

		if len(archiveBeans) == 0 {
			if archiveJSON {
				return output.SuccessMessage("No beans to archive")
			}
			fmt.Println("No beans with archive status to archive.")
			return nil
		}

		// Sort beans for consistent display
		bean.SortByStatusPriorityAndType(archiveBeans, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

		// Remove external links before archiving
		removedRefs := 0
		for _, b := range archiveBeans {
			removed, err := core.RemoveLinksTo(b.ID)
			if err != nil {
				if archiveJSON {
					return output.Error(output.ErrFileError, fmt.Sprintf("failed to remove references to %s: %s", b.ID, err))
				}
				return fmt.Errorf("failed to remove references to %s: %w", b.ID, err)
			}
			removedRefs += removed
		}

		// Archive all beans with archive status
		var archived []string
		for _, b := range archiveBeans {
			if err := core.Archive(b.ID); err != nil {
				if archiveJSON {
					return output.Error(output.ErrFileError, fmt.Sprintf("failed to archive bean %s: %s", b.ID, err.Error()))
				}
				return fmt.Errorf("failed to archive bean %s: %w", b.ID, err)
			}
			archived = append(archived, b.ID)
		}

		if archiveJSON {
			return output.SuccessMessage(fmt.Sprintf("Archived %d bean(s) to .beans/archive/", len(archived)))
		}

		if removedRefs > 0 {
			fmt.Printf("Removed %d reference(s)\n", removedRefs)
		}
		fmt.Printf("Archived %d bean(s) to .beans/archive/\n", len(archived))
		return nil
	},
}

func init() {
	archiveCmd.Flags().BoolVar(&archiveJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(archiveCmd)
}
