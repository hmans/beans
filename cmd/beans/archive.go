package beans

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/output"
)

var (
	archiveForce bool
	archiveJSON  bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Delete all beans with the archive status",
	Long:  `Deletes all beans that have the archive status (default: "done"). Asks for confirmation unless --force is provided.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		beans, err := store.FindAll()
		if err != nil {
			if archiveJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to list beans: %w", err)
		}

		archiveStatus := cfg.GetArchiveStatus()

		// Find beans with archive status
		var archiveBeans []string
		for _, b := range beans {
			if b.Status == archiveStatus {
				archiveBeans = append(archiveBeans, b.ID)
			}
		}

		if len(archiveBeans) == 0 {
			if archiveJSON {
				return output.SuccessMessage("No beans to archive")
			}
			fmt.Printf("No beans with status '%s' to archive.\n", archiveStatus)
			return nil
		}

		// JSON implies force (no prompts for machines)
		if !archiveForce && !archiveJSON {
			var confirm bool
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Archive %d bean(s) with status '%s'?", len(archiveBeans), archiveStatus)).
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

		// Delete all beans with archive status
		var deleted []string
		for _, id := range archiveBeans {
			if err := store.Delete(id); err != nil {
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

		fmt.Printf("Archived %d bean(s)\n", len(deleted))
		return nil
	},
}

func init() {
	archiveCmd.Flags().BoolVarP(&archiveForce, "force", "f", false, "Skip confirmation")
	archiveCmd.Flags().BoolVar(&archiveJSON, "json", false, "Output as JSON (implies --force)")
	rootCmd.AddCommand(archiveCmd)
}
