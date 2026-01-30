package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	checkJSON bool
	checkFix  bool
)

type checkResult struct {
	Success      bool                      `json:"success"`
	ConfigErrors []string                  `json:"config_errors"`
	BeanIssues   *beancore.LinkCheckResult `json:"bean_issues,omitempty"`
	Fixed        int                       `json:"fixed,omitempty"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate configuration and bean integrity",
	Long: `Checks configuration and bean integrity, including:
- Configuration settings (colors, default type)
- Broken links (links to non-existent beans)
- Self-references (beans linking to themselves)
- Circular dependencies (cycles in blocks/parent relationships)

Use --fix to automatically remove broken links and self-references.
Note: Cycles cannot be auto-fixed and require manual intervention.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var configErrors []string
		var fixed int

		// === Configuration checks ===
		if !checkJSON {
			fmt.Println(ui.Bold.Render("Configuration"))
		}

		// 1. Check statuses are defined (always true since hardcoded)
		if !checkJSON {
			fmt.Printf("  %s Statuses defined (%d hardcoded)\n", ui.Success.Render("✓"), len(config.DefaultStatuses))
		}

		// 2. Check default_status exists in statuses (always true since hardcoded)
		if !checkJSON {
			fmt.Printf("  %s Default status '%s' exists\n", ui.Success.Render("✓"), cfg.GetDefaultStatus())
		}

		// 2b. Check default_type is a valid hardcoded type
		if cfg.GetDefaultType() != "" && !cfg.IsValidType(cfg.GetDefaultType()) {
			configErrors = append(configErrors, fmt.Sprintf("default_type '%s' is not a valid type", cfg.GetDefaultType()))
		} else if cfg.GetDefaultType() != "" {
			if !checkJSON {
				fmt.Printf("  %s Default type '%s' is valid\n", ui.Success.Render("✓"), cfg.GetDefaultType())
			}
		}

		// 3. Check all status colors are valid (hardcoded statuses)
		for _, s := range config.DefaultStatuses {
			if !ui.IsValidColor(s.Color) {
				configErrors = append(configErrors, fmt.Sprintf("invalid color '%s' for status '%s'", s.Color, s.Name))
			}
		}
		if !checkJSON {
			colorErrors := 0
			for _, e := range configErrors {
				if len(e) > 13 && e[:13] == "invalid color" {
					colorErrors++
				}
			}
			if colorErrors == 0 {
				fmt.Printf("  %s All status colors valid\n", ui.Success.Render("✓"))
			}
		}

		// 4. Check all type colors are valid (hardcoded types)
		for _, t := range config.DefaultTypes {
			if !ui.IsValidColor(t.Color) {
				configErrors = append(configErrors, fmt.Sprintf("invalid color '%s' for type '%s'", t.Color, t.Name))
			}
		}
		if !checkJSON {
			typeColorErrors := 0
			for _, e := range configErrors {
				if len(e) > 13 && e[:13] == "invalid color" {
					typeColorErrors++
				}
			}
			if typeColorErrors == 0 {
				fmt.Printf("  %s All type colors valid\n", ui.Success.Render("✓"))
			}
		}

		// 5. Check custom prime template exists (if configured)
		if cfg.Templates.Prime != "" {
			primePath := cfg.ResolvePrimeTemplatePath()
			if _, err := os.Stat(primePath); os.IsNotExist(err) {
				configErrors = append(configErrors, fmt.Sprintf("custom prime template not found: %s", primePath))
			} else if err != nil {
				configErrors = append(configErrors, fmt.Sprintf("cannot access custom prime template: %s: %v", primePath, err))
			} else if !checkJSON {
				fmt.Printf("  %s Custom prime template exists (%s)\n", ui.Success.Render("✓"), cfg.Templates.Prime)
			}
		}

		// Print config errors in human-readable mode
		if !checkJSON {
			for _, e := range configErrors {
				fmt.Printf("  %s %s\n", ui.Danger.Render("✗"), e)
			}
		}

		// === Bean link checks ===
		if !checkJSON {
			fmt.Println()
			fmt.Println(ui.Bold.Render("Bean Links"))
		}

		linkResult := core.CheckAllLinks()

		// Handle --fix mode
		if checkFix && (len(linkResult.BrokenLinks) > 0 || len(linkResult.SelfLinks) > 0) {
			fixedCount, err := core.FixBrokenLinks()
			if err != nil {
				return fmt.Errorf("fixing broken links: %w", err)
			}
			fixed = fixedCount

			if !checkJSON {
				for _, bl := range linkResult.BrokenLinks {
					fmt.Printf("  %s %s: removed broken link %s:%s\n", ui.Success.Render("✓"), bl.BeanID, bl.LinkType, bl.Target)
				}
				for _, sl := range linkResult.SelfLinks {
					fmt.Printf("  %s %s: removed self-reference in %s link\n", ui.Success.Render("✓"), sl.BeanID, sl.LinkType)
				}
			}

			// Clear the fixed issues from the result
			linkResult.BrokenLinks = []beancore.BrokenLink{}
			linkResult.SelfLinks = []beancore.SelfLink{}
		} else if !checkJSON {
			// Report issues without fixing
			for _, bl := range linkResult.BrokenLinks {
				fmt.Printf("  %s %s: broken link %s:%s\n", ui.Danger.Render("✗"), bl.BeanID, bl.LinkType, bl.Target)
			}
			for _, sl := range linkResult.SelfLinks {
				fmt.Printf("  %s %s: self-reference in %s link\n", ui.Danger.Render("✗"), sl.BeanID, sl.LinkType)
			}
		}

		// Cycles cannot be auto-fixed
		if !checkJSON {
			for _, c := range linkResult.Cycles {
				if checkFix {
					fmt.Printf("  %s Cannot auto-fix cycle: %s (via %s)\n", ui.Warning.Render("!"), formatCycle(c.Path), c.LinkType)
				} else {
					fmt.Printf("  %s Circular dependency: %s (via %s)\n", ui.Danger.Render("✗"), formatCycle(c.Path), c.LinkType)
				}
			}
		}

		// Show success if no issues
		if !checkJSON && !linkResult.HasIssues() && fixed == 0 {
			fmt.Printf("  %s No link issues found\n", ui.Success.Render("✓"))
		}

		// === Summary ===
		totalIssues := len(configErrors) + linkResult.TotalIssues()

		if checkJSON {
			result := checkResult{
				Success:      totalIssues == 0,
				ConfigErrors: configErrors,
				BeanIssues:   linkResult,
				Fixed:        fixed,
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Println()
			if totalIssues == 0 && fixed == 0 {
				fmt.Println(ui.Success.Render("All checks passed"))
			} else if totalIssues == 0 && fixed > 0 {
				fmt.Println(ui.Success.Render(fmt.Sprintf("Fixed %d issue(s)", fixed)))
			} else if fixed > 0 {
				// Some issues fixed, some remain (cycles)
				fmt.Println(ui.Warning.Render(fmt.Sprintf("Fixed %d issue(s), %d require manual intervention", fixed, totalIssues)))
			} else if totalIssues == 1 {
				fmt.Println(ui.Danger.Render("1 issue found"))
			} else {
				fmt.Println(ui.Danger.Render(fmt.Sprintf("%d issues found", totalIssues)))
			}
		}

		// Exit with error code if validation failed
		if totalIssues > 0 {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Output as JSON")
	checkCmd.Flags().BoolVar(&checkFix, "fix", false, "Automatically fix broken links and self-references")
	rootCmd.AddCommand(checkCmd)
}
