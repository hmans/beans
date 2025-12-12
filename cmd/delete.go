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

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a bean",
	Long: `Deletes a bean after confirmation (use -f to skip confirmation).

If other beans reference this bean (as parent or via blocking), you will be warned
and those references will be removed after confirmation. Use -f to skip all warnings.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &graph.Resolver{Core: core}

		// Find the bean
		b, err := resolver.Query().Bean(ctx, args[0])
		if err != nil {
			return cmdError(deleteJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}
		if b == nil {
			return cmdError(deleteJSON, output.ErrNotFound, "bean not found: %s", args[0])
		}

		// Check for incoming links
		incomingLinks := core.FindIncomingLinks(b.ID)
		hasIncoming := len(incomingLinks) > 0

		// Prompt for confirmation (JSON implies force)
		if !forceDelete && !deleteJSON {
			if !confirmDelete(b, incomingLinks) {
				fmt.Println("Cancelled")
				return nil
			}
		}

		// Delete via GraphQL mutation
		_, err = resolver.Mutation().DeleteBean(ctx, b.ID)
		if err != nil {
			return cmdError(deleteJSON, output.ErrFileError, "failed to delete bean: %v", err)
		}

		if deleteJSON {
			return output.Success(b, "Bean deleted")
		}

		if hasIncoming {
			fmt.Printf("Removed %d reference(s)\n", len(incomingLinks))
		}
		fmt.Printf("Deleted %s\n", b.Path)
		return nil
	},
}

// confirmDelete prompts the user to confirm deletion, returning true if confirmed.
func confirmDelete(b *bean.Bean, incomingLinks []beancore.IncomingLink) bool {
	if len(incomingLinks) > 0 {
		fmt.Printf("Warning: %d bean(s) link to '%s':\n", len(incomingLinks), b.Title)
		for _, link := range incomingLinks {
			fmt.Printf("  - %s (%s) via %s\n", link.FromBean.ID, link.FromBean.Title, link.LinkType)
		}
		fmt.Print("Delete anyway and remove references? [y/N] ")
	} else {
		fmt.Printf("Delete '%s' (%s)? [y/N] ", b.Title, b.Path)
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
