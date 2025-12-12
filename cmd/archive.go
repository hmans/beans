package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/output"
)

var (
	archiveForce bool
	archiveJSON  bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Delete all beans with an archive status",
	Long: `Deletes all beans with status "completed" or "scrapped". Asks for confirmation unless --force is provided.

If other beans reference beans being archived (as parent or via blocking), you will be
warned and those references will be removed. Use -f to skip all warnings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		beans := core.All()

		// Find beans with any archive status
		var archiveBeans []string
		archiveSet := make(map[string]bool)
		for _, b := range beans {
			if cfg.IsArchiveStatus(b.Status) {
				archiveBeans = append(archiveBeans, b.ID)
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

		// Find incoming links from non-archived beans to beans being archived
		var externalLinks []beancore.IncomingLink
		for _, id := range archiveBeans {
			links := core.FindIncomingLinks(id)
			for _, link := range links {
				// Only count links from beans NOT being archived
				if !archiveSet[link.FromBean.ID] {
					externalLinks = append(externalLinks, link)
				}
			}
		}
		hasExternalLinks := len(externalLinks) > 0

		// JSON implies force (no prompts for machines)
		if !archiveForce && !archiveJSON {
			// Show warning if there are external links
			if hasExternalLinks {
				fmt.Printf("Warning: %d bean(s) link to beans being archived:\n", len(externalLinks))
				for _, link := range externalLinks {
					fmt.Printf("  - %s (%s) links to %s via %s\n",
						link.FromBean.ID, link.FromBean.Title,
						link.LinkType, link.LinkType)
				}
				fmt.Println()
			}

			var confirm bool
			title := fmt.Sprintf("Archive %d bean(s)?", len(archiveBeans))
			if hasExternalLinks {
				title = fmt.Sprintf("Archive %d bean(s) and remove %d reference(s)?", len(archiveBeans), len(externalLinks))
			}

			err := huh.NewConfirm().
				Title(title).
				Affirmative("Yes").
				Negative("No").
				Value(&confirm).
				Run()

			if err != nil {
				return err
			}

			if !confirm {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Remove external links before deletion
		removedRefs := 0
		for _, id := range archiveBeans {
			removed, err := core.RemoveLinksTo(id)
			if err != nil {
				if archiveJSON {
					return output.Error(output.ErrFileError, fmt.Sprintf("failed to remove references to %s: %s", id, err))
				}
				return fmt.Errorf("failed to remove references to %s: %w", id, err)
			}
			removedRefs += removed
		}

		// Delete all beans with archive status
		var deleted []string
		for _, id := range archiveBeans {
			if err := core.Delete(id); err != nil {
				if archiveJSON {
					return output.Error(output.ErrFileError, fmt.Sprintf("failed to delete bean %s: %s", id, err.Error()))
				}
				return fmt.Errorf("failed to delete bean %s: %w", id, err)
			}
			deleted = append(deleted, id)
		}

		if archiveJSON {
			return output.SuccessMessage(fmt.Sprintf("Archived %d bean(s)", len(deleted)))
		}

		if removedRefs > 0 {
			fmt.Printf("Removed %d reference(s)\n", removedRefs)
		}
		fmt.Printf("Archived %d bean(s)\n", len(deleted))
		return nil
	},
}

func init() {
	archiveCmd.Flags().BoolVarP(&archiveForce, "force", "f", false, "Skip confirmation and warnings")
	archiveCmd.Flags().BoolVar(&archiveJSON, "json", false, "Output as JSON (implies --force)")
	rootCmd.AddCommand(archiveCmd)
}
