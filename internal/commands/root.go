package commands

import (
	"fmt"
	"os"

	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/spf13/cobra"
)

var core *beancore.Core
var cfg *config.Config
var beansPath string
var configPath string

// NewRootCmd creates the root cobra command with shared persistent flags
// and core initialization logic.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
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
				cfg, err = config.Load(configPath)
				if err != nil {
					return fmt.Errorf("loading config from %s: %w", configPath, err)
				}
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting current directory: %w", err)
				}
				cfg, err = config.LoadFromDirectory(cwd)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
			}

			root, err := resolveBeansPath(beansPath, cfg)
			if err != nil {
				return err
			}

			core = beancore.New(root, cfg)
			if err := core.Load(); err != nil {
				return fmt.Errorf("loading beans: %w", err)
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&beansPath, "beans-path", "", "Path to data directory (overrides config and BEANS_PATH env var)")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default: searches upward for .beans.yml)")

	return rootCmd
}

// resolveBeansPath determines the beans data directory path.
// Precedence: --beans-path flag > BEANS_PATH env var > config.
func resolveBeansPath(flagPath string, c *config.Config) (string, error) {
	var root string
	if flagPath != "" {
		root = flagPath
	} else if envPath := os.Getenv("BEANS_PATH"); envPath != "" {
		root = envPath
	} else {
		root = c.ResolveBeansPath()
	}

	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		if flagPath != "" || os.Getenv("BEANS_PATH") != "" {
			return "", fmt.Errorf("beans path does not exist or is not a directory: %s", root)
		}
		return "", fmt.Errorf("no .beans directory found at %s (run 'beans init' to create one)", root)
	}

	return root, nil
}

// Execute runs the given root command and exits on error.
func Execute(rootCmd *cobra.Command) {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
