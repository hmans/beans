package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/hmans/beans/internal/config"
	"github.com/spf13/cobra"
)

//go:embed prompt.tmpl
var agentPromptTemplate string

// promptData holds all data needed to render the prompt template.
type promptData struct {
	GraphQLSchema string
	Types         []config.TypeConfig
	Statuses      []config.StatusConfig
	Priorities    []config.PriorityConfig
	OriginalPrime string
}

// primeResult holds the output of renderPrime for inspection.
type primeResult struct {
	Output  string // The rendered prime output
	Warning string // Non-empty if a fallback occurred
}

// renderPrime renders the prime output using the given config.
// If a custom template is configured but cannot be loaded or parsed,
// it falls back to the built-in template and sets a warning.
func renderPrime(primeCfg *config.Config, data promptData) (*primeResult, error) {
	builtinTmpl, err := template.New("prompt").Parse(agentPromptTemplate)
	if err != nil {
		return nil, err
	}

	primePath := primeCfg.ResolvePrimeTemplatePath()
	if primePath == "" {
		var buf bytes.Buffer
		if err := builtinTmpl.Execute(&buf, data); err != nil {
			return nil, err
		}
		return &primeResult{Output: buf.String()}, nil
	}

	// Custom template configured - render built-in to string first
	var builtinBuf bytes.Buffer
	if err := builtinTmpl.Execute(&builtinBuf, data); err != nil {
		return nil, fmt.Errorf("rendering built-in prime template: %w", err)
	}
	data.OriginalPrime = builtinBuf.String()

	// Read and parse the custom template
	customTmplContent, err := os.ReadFile(primePath)
	if err != nil {
		return &primeResult{
			Output:  builtinBuf.String(),
			Warning: fmt.Sprintf("custom prime template not found: %s (falling back to built-in)", primePath),
		}, nil
	}

	customTmpl, err := template.New("custom-prompt").Parse(string(customTmplContent))
	if err != nil {
		return &primeResult{
			Output:  builtinBuf.String(),
			Warning: fmt.Sprintf("parsing custom prime template: %v (falling back to built-in)", err),
		}, nil
	}

	var customBuf bytes.Buffer
	if err := customTmpl.Execute(&customBuf, data); err != nil {
		return nil, fmt.Errorf("executing custom prime template: %w", err)
	}
	return &primeResult{Output: customBuf.String()}, nil
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output instructions for AI coding agents",
	Long:  `Outputs a prompt that primes AI coding agents on how to use the beans CLI to manage project issues.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no explicit path given, check if a beans project exists by searching
		// upward for a .beans.yml config file
		var primeCfg *config.Config
		if beansPath == "" && configPath == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return nil // Silently exit on error
			}
			configFile, err := config.FindConfig(cwd)
			if err != nil || configFile == "" {
				// No config file found - silently exit
				return nil
			}
			primeCfg, err = config.Load(configFile)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
		} else if configPath != "" {
			var err error
			primeCfg, err = config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config from %s: %w", configPath, err)
			}
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}
			primeCfg, err = config.LoadFromDirectory(cwd)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
		}

		data := promptData{
			GraphQLSchema: GetGraphQLSchema(),
			Types:         config.DefaultTypes,
			Statuses:      config.DefaultStatuses,
			Priorities:    config.DefaultPriorities,
		}

		result, err := renderPrime(primeCfg, data)
		if err != nil {
			return err
		}

		if result.Warning != "" {
			fmt.Fprintf(os.Stderr, "warning: %s\n", result.Warning)
		}

		_, err = io.WriteString(os.Stdout, result.Output)
		return err
	},
}

func init() {
	rootCmd.AddCommand(primeCmd)
}
