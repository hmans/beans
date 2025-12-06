package beans

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/output"
)

var (
	forceDelete bool
	deleteJSON  bool
)

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a bean",
	Long:    `Deletes a bean after confirmation (use -f to skip confirmation).`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		b, err := store.FindByID(args[0])
		if err != nil {
			if deleteJSON {
				return output.Error(output.ErrNotFound, err.Error())
			}
			return fmt.Errorf("failed to find bean: %w", err)
		}

		// JSON implies force (no prompts for machines)
		if !forceDelete && !deleteJSON {
			fmt.Printf("Delete '%s' (%s)? [y/N] ", b.Title, b.Path)
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		if err := store.Delete(args[0]); err != nil {
			if deleteJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to delete bean: %w", err)
		}

		if deleteJSON {
			return output.Success(b, "Bean deleted")
		}

		fmt.Printf("Deleted %s\n", b.Path)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&forceDelete, "force", "f", false, "Skip confirmation")
	deleteCmd.Flags().BoolVar(&deleteJSON, "json", false, "Output as JSON (implies --force)")
	rootCmd.AddCommand(deleteCmd)
}
