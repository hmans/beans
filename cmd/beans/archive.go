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
	Short: "Delete all beans with an archive status",
	Long:  `Deletes all beans that have a status marked with archive=true in beans.toml. Asks for confirmation unless --force is provided.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		beans, err := store.FindAll()
		if err != nil {
			if archiveJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to list beans: %w", err)
		}

		// Find beans with any archive status
		var archiveBeans []string
		for _, b := range beans {
			if cfg.IsArchiveStatus(b.Status) {
				archiveBeans = append(archiveBeans, b.ID)
			}
		}

		if len(archiveBeans) == 0 {
			if archiveJSON {
				return output.SuccessMessage("No beans to archive")
			}
			fmt.Println("No beans with archive status to archive.")
			return nil
		}

		// JSON implies force (no prompts for machines)
		if !archiveForce && !archiveJSON {
			var confirm bool
			err := huh.NewConfirm().
				Title(fmt.Sprintf("Archive %d bean(s)?", len(archiveBeans))).
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
