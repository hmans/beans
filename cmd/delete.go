package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/output"
	"github.com/spf13/cobra"
)

var (
	forceDelete bool
	deleteJSON  bool
)

// beanWithLinks holds a bean and its incoming links for batch processing
type beanWithLinks struct {
	bean  *bean.Bean
	links []beancore.IncomingLink
}

var deleteCmd = &cobra.Command{
	Use:     "delete <id> [id...]",
	Aliases: []string{"rm"},
	Short:   "Delete one or more beans",
	Long: `Deletes one or more beans after confirmation (use -f to skip confirmation).

If other beans reference the target bean(s) (as parent or via blocking), you will be
warned and those references will be removed after confirmation. Use -f to skip all warnings.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &graph.Resolver{Core: core}

		// Collect all beans and their incoming links upfront (validate before deleting)
		var targets []beanWithLinks
		for _, id := range args {
			b, err := resolver.Query().Bean(ctx, id)
			if err != nil {
				return cmdError(deleteJSON, output.ErrNotFound, "failed to find bean: %v", err)
			}
			if b == nil {
				return cmdError(deleteJSON, output.ErrNotFound, "bean not found: %s", id)
			}
			targets = append(targets, beanWithLinks{
				bean:  b,
				links: core.FindIncomingLinks(b.ID),
			})
		}

		// Prompt for confirmation (JSON implies force)
		if !forceDelete && !deleteJSON {
			if !confirmDeleteMultiple(targets) {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Delete all beans via GraphQL mutation
		var deleted []*bean.Bean
		var totalLinksRemoved int
		for _, target := range targets {
			_, err := resolver.Mutation().DeleteBean(ctx, target.bean.ID)
			if err != nil {
				return cmdError(deleteJSON, output.ErrFileError, "failed to delete bean %s: %v", target.bean.ID, err)
			}
			deleted = append(deleted, target.bean)
			totalLinksRemoved += len(target.links)
		}

		// Output results
		if deleteJSON {
			if len(deleted) == 1 {
				return output.Success(deleted[0], "Bean deleted")
			}
			return output.JSON(output.Response{
				Success: true,
				Beans:   deleted,
				Count:   len(deleted),
				Message: fmt.Sprintf("%d beans deleted", len(deleted)),
			})
		}

		if totalLinksRemoved > 0 {
			fmt.Printf("Removed %d reference(s)\n", totalLinksRemoved)
		}
		for _, b := range deleted {
			fmt.Printf("Deleted %s\n", b.Path)
		}
		return nil
	},
}

// confirmDeleteMultiple prompts the user to confirm deletion of one or more beans.
func confirmDeleteMultiple(targets []beanWithLinks) bool {
	beansWithLinks := 0
	totalLinks := 0
	for _, t := range targets {
		if len(t.links) > 0 {
			beansWithLinks++
			totalLinks += len(t.links)
		}
	}

	// Single bean: use simpler format
	if len(targets) == 1 {
		t := targets[0]
		if len(t.links) > 0 {
			fmt.Printf("Warning: %d bean(s) link to '%s':\n", len(t.links), t.bean.Title)
			for _, link := range t.links {
				fmt.Printf("  - %s (%s) via %s\n", link.FromBean.ID, link.FromBean.Title, link.LinkType)
			}
			fmt.Print("Delete anyway and remove references? [y/N] ")
		} else {
			fmt.Printf("Delete '%s' (%s)? [y/N] ", t.bean.Title, t.bean.Path)
		}
	} else {
		// Multiple beans: show batch summary
		fmt.Printf("About to delete %d bean(s):\n", len(targets))
		for _, t := range targets {
			if len(t.links) > 0 {
				fmt.Printf("  - %s (%s) ← %d incoming link(s)\n", t.bean.ID, t.bean.Title, len(t.links))
			} else {
				fmt.Printf("  - %s (%s)\n", t.bean.ID, t.bean.Title)
			}
		}
		if beansWithLinks > 0 {
			fmt.Printf("\nWarning: %d bean(s) have incoming references (%d total) that will be removed.\n", beansWithLinks, totalLinks)
		}
		fmt.Print("\nProceed with deletion? [y/N] ")
	}

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation and warnings")
	deleteCmd.Flags().BoolVar(&deleteJSON, "json", false, "Output as JSON (implies --force)")
	rootCmd.AddCommand(deleteCmd)
}
