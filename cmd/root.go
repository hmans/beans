package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

var core *beancore.Core
var cfg *config.Config
var beansPath string
var configPath string

var rootCmd = &cobra.Command{
	Use:   "beans",
	Short: "A file-based issue tracker for AI-first workflows",
	Long: `Beans is a lightweight issue tracker that stores issues as markdown files.
Track your work alongside your code and supercharge your coding agent with
a full view of your project.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip core initialization for init, prime, and version commands
		if cmd.Name() == "init" || cmd.Name() == "prime" || cmd.Name() == "version" {
			return nil
		}

		var err error

		// Load configuration
		if configPath != "" {
			// Use explicit config path
			cfg, err = config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config from %s: %w", configPath, err)
			}
		} else {
			// Search upward for .beans.yml
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			cfg, err = config.LoadFromDirectory(cwd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
		}

		// Determine beans directory
		var root string
		if beansPath != "" {
			// Use explicit beans path (overrides config)
			root = beansPath
			// Verify it exists
			if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
				return fmt.Errorf("beans path does not exist or is not a directory: %s", root)
			}
		} else {
			// Use path from config
			root = cfg.ResolveBeansPath()
			// Verify it exists
			if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
				return fmt.Errorf("no .beans directory found at %s (run 'beans init' to create one)", root)
			}
		}

		core = beancore.New(root, cfg)
		if err := core.Load(); err != nil {
			return fmt.Errorf("loading beans: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&beansPath, "beans-path", "", "Path to data directory (overrides config)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: searches upward for .beans.yml)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
