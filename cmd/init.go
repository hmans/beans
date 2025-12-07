package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"hmans.dev/beans/internal/bean"
	"hmans.dev/beans/internal/config"
	"hmans.dev/beans/internal/output"
)

var initJSON bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a .beans directory",
	Long:  `Creates a .beans directory in the current directory for storing issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var beansDir string
		var dirName string

		if beansPath != "" {
			// Use explicit path
			beansDir = beansPath
			dirName = filepath.Base(filepath.Dir(beansDir))
			// Create the directory
			if err := os.MkdirAll(beansDir, 0755); err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Use current working directory
			dir, err := os.Getwd()
			if err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return err
			}

			if err := bean.Init(dir); err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return fmt.Errorf("failed to initialize: %w", err)
			}

			beansDir = filepath.Join(dir, ".beans")
			dirName = filepath.Base(dir)
		}

		// Create default config file with directory name as prefix
		defaultCfg := config.DefaultWithPrefix(dirName + "-")
		if err := defaultCfg.Save(beansDir); err != nil {
			if initJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to create config: %w", err)
		}

		if initJSON {
			return output.SuccessInit(beansDir)
		}

		fmt.Println("Initialized .beans directory")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(initCmd)
}
