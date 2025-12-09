package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/output"
)

var initJSON bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a beans project",
	Long:  `Creates a .beans directory and .beans.yml config file in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var projectDir string
		var beansDir string
		var dirName string

		if beansPath != "" {
			// Use explicit path for beans directory
			beansDir = beansPath
			projectDir = filepath.Dir(beansDir)
			dirName = filepath.Base(projectDir)
			// Create the directory using Core.Init to set up .gitignore
			core := beancore.New(beansDir, nil)
			if err := core.Init(); err != nil {
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

			if err := beancore.Init(dir); err != nil {
				if initJSON {
					return output.Error(output.ErrFileError, err.Error())
				}
				return fmt.Errorf("failed to initialize: %w", err)
			}

			projectDir = dir
			beansDir = filepath.Join(dir, ".beans")
			dirName = filepath.Base(dir)
		}

		// Create default config file with directory name as prefix
		// Config is saved at project root (not inside .beans/)
		defaultCfg := config.DefaultWithPrefix(dirName + "-")
		defaultCfg.SetConfigDir(projectDir)
		if err := defaultCfg.Save(projectDir); err != nil {
			if initJSON {
				return output.Error(output.ErrFileError, err.Error())
			}
			return fmt.Errorf("failed to create config: %w", err)
		}

		if initJSON {
			return output.SuccessInit(beansDir)
		}

		fmt.Println("Initialized beans project")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(initCmd)
}
