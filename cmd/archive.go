package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	archiveForce bool
	archiveJSON  bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Move completed/scrapped beans to the archive",
	Long: `Moves all beans with status "completed" or "scrapped" to the archive directory (.beans/archive/).
Archived beans are preserved for project memory but excluded from normal queries.
Use --with-archived flag on other commands to include archived beans.

If other beans reference beans being archived (as parent or via blocking), you will be
warned and those references will be removed. Use -f to skip all warnings.`,
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

		// Find incoming links from non-archived beans to beans being archived
		var externalLinks []beancore.IncomingLink
		for _, b := range archiveBeans {
			links := core.FindIncomingLinks(b.ID)
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
			// Show list of beans to be archived
			fmt.Printf("Beans to archive (%d):\n\n", len(archiveBeans))
			printBeanList(archiveBeans)
			fmt.Println()

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

// printBeanList prints a formatted list of beans
func printBeanList(beans []*bean.Bean) {
	// Calculate max ID width
	maxIDWidth := 0
	for _, b := range beans {
		if len(b.ID) > maxIDWidth {
			maxIDWidth = len(b.ID)
		}
	}
	maxIDWidth += 2 // padding

	// Check if any beans have tags
	hasTags := false
	for _, b := range beans {
		if len(b.Tags) > 0 {
			hasTags = true
			break
		}
	}

	// Print each bean
	for _, b := range beans {
		colors := cfg.GetBeanColors(b.Status, b.Type, b.Priority)
		row := ui.RenderBeanRow(b.ID, b.Status, b.Type, b.Title, ui.BeanRowConfig{
			StatusColor:   colors.StatusColor,
			TypeColor:     colors.TypeColor,
			PriorityColor: colors.PriorityColor,
			Priority:      b.Priority,
			IsArchive:     colors.IsArchive,
			MaxTitleWidth: 60,
			ShowCursor:    false,
			IsSelected:    false,
			Tags:          b.Tags,
			ShowTags:      hasTags,
			IDColWidth:    maxIDWidth,
		})
		fmt.Println(row)
	}
}

func init() {
	archiveCmd.Flags().BoolVarP(&archiveForce, "force", "f", false, "Skip confirmation and warnings")
	archiveCmd.Flags().BoolVar(&archiveJSON, "json", false, "Output as JSON (implies --force)")
	rootCmd.AddCommand(archiveCmd)
}
