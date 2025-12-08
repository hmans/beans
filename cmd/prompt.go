package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/config"
)

//go:embed prompt.md
var agentPrompt string

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Output instructions for AI coding agents",
	Long:  `Outputs a prompt that instructs AI coding agents on how to use the beans CLI to manage project issues.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no explicit path given, check if a beans project exists
		if beansPath == "" && configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return nil // Silently exit on error
			}
			cfg, err := config.LoadFromDirectory(cwd)
			if err != nil {
				return nil // Silently exit on error
			}
			// Check if the beans directory exists
			beansDir := cfg.ResolveBeansPath()
			if _, err := os.Stat(beansDir); os.IsNotExist(err) {
				// No beans directory found - silently exit
				return nil
			}
		}

		fmt.Print(agentPrompt)

		// Append dynamic sections
		var sb strings.Builder

		// GraphQL schema section (generated dynamically)
		sb.WriteString("\n**GraphQL Schema:**\n\n")
		sb.WriteString("```graphql\n")
		sb.WriteString(GetGraphQLSchema())
		sb.WriteString("```\n")

		// Issue types section
		sb.WriteString("\n## Issue Types\n\n")
		sb.WriteString("This project has the following issue types configured. Always specify a type with `-t` when creating beans:\n\n")
		for _, t := range config.DefaultTypes {
			if t.Description != "" {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Name, t.Description))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s**\n", t.Name))
			}
		}

		// Statuses section (hardcoded statuses)
		sb.WriteString("\n## Statuses\n\n")
		sb.WriteString("This project has the following statuses configured:\n\n")
		for _, s := range config.DefaultStatuses {
			if s.Description != "" {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", s.Name, s.Description))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s**\n", s.Name))
			}
		}

		// Priorities section (hardcoded priorities)
		sb.WriteString("\n## Priorities\n\n")
		sb.WriteString("Beans can have an optional priority. Use `-p` when creating or `--priority` when updating:\n\n")
		for _, p := range config.DefaultPriorities {
			if p.Description != "" {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.Name, p.Description))
			} else {
				sb.WriteString(fmt.Sprintf("- **%s**\n", p.Name))
			}
		}
		sb.WriteString("\nBeans without a priority are treated as `normal` priority for sorting purposes.\n")

		sb.WriteString("\n")
		fmt.Print(sb.String())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(promptCmd)
}
