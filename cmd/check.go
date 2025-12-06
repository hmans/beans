package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/ui"
)

var checkJSON bool

type checkResult struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate config.yaml configuration",
	Long:  `Checks config.yaml for configuration issues such as invalid statuses, colors, or missing required values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var errors []string

		// 1. Check statuses not empty
		if len(cfg.Statuses) == 0 {
			errors = append(errors, "no statuses defined")
		} else {
			if !checkJSON {
				fmt.Printf("%s Statuses defined (%d)\n", ui.Success.Render("✓"), len(cfg.Statuses))
			}
		}

		// 2. Check default_status exists in statuses
		if len(cfg.Statuses) > 0 && !cfg.IsValidStatus(cfg.GetDefaultStatus()) {
			errors = append(errors, fmt.Sprintf("default_status '%s' is not a defined status", cfg.GetDefaultStatus()))
		} else if len(cfg.Statuses) > 0 {
			if !checkJSON {
				fmt.Printf("%s Default status '%s' exists\n", ui.Success.Render("✓"), cfg.GetDefaultStatus())
			}
		}

		// 3. Check all colors are valid
		for _, s := range cfg.Statuses {
			if !ui.IsValidColor(s.Color) {
				errors = append(errors, fmt.Sprintf("invalid color '%s' for status '%s'", s.Color, s.Name))
			}
		}
		if len(cfg.Statuses) > 0 && !checkJSON {
			// Only show success if we checked colors and found no errors related to colors
			colorErrors := 0
			for _, e := range errors {
				if len(e) > 13 && e[:13] == "invalid color" {
					colorErrors++
				}
			}
			if colorErrors == 0 {
				fmt.Printf("%s All status colors valid\n", ui.Success.Render("✓"))
			}
		}

		// Print errors in human-readable mode
		if !checkJSON {
			for _, e := range errors {
				fmt.Printf("%s %s\n", ui.Danger.Render("✗"), e)
			}
		}

		// Output results
		if checkJSON {
			result := checkResult{
				Success: len(errors) == 0,
				Errors:  errors,
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println()
			if len(errors) == 0 {
				fmt.Println(ui.Success.Render("Configuration valid"))
			} else if len(errors) == 1 {
				fmt.Println(ui.Danger.Render("1 error found"))
			} else {
				fmt.Println(ui.Danger.Render(fmt.Sprintf("%d errors found", len(errors))))
			}
		}

		// Exit with error code if validation failed
		if len(errors) > 0 {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(checkCmd)
}
