package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

var focusJSON bool

// focusJSONResult is `beans focus --json`. Focus is empty when no focus is
// set; Problem is set when the configured focus is invalid.
type focusJSONResult struct {
	Focus   string `json:"focus,omitempty"`
	File    string `json:"file,omitempty"`
	Problem string `json:"problem,omitempty"`
}

func writeFocus(w io.Writer, f resolvedFocus) error {
	if !focusJSON {
		_, err := fmt.Fprintln(w, focusSummary(f))
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(focusJSONResult{Focus: f.id, File: f.source, Problem: f.problem})
}

// focusFail reports err as JSON under --json, like other commands do.
func focusFail(code string, err error) error {
	if focusJSON {
		return output.ErrorFrom(code, err)
	}
	return err
}

var focusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Show the milestone or epic that \"beans next\" ranks first",
	Long: `Shows the focus of "beans next" and the file it comes from.

The focus is read from .beans.yml (next.focus) and from the per-user,
gitignored .beans.local.yml, which wins. "focus set" and "focus clear"
write only .beans.local.yml.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := allBeans()
		if err != nil {
			return focusFail(output.ErrFileError, err)
		}
		return writeFocus(cmd.OutOrStdout(), resolveFocus(all))
	},
}

var focusSetCmd = &cobra.Command{
	Use:   "set <id>",
	Short: "Set the focus to a milestone or epic (in " + config.LocalConfigFileName + ")",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		all, err := allBeans()
		if err != nil {
			return focusFail(output.ErrFileError, err)
		}
		id, _ := core.NormalizeID(args[0])
		if problem := bean.FocusProblem(id, all); problem != "" {
			return focusFail(output.ErrValidation, fmt.Errorf("cannot focus %s: %s", id, problem))
		}
		if err := cfg.SetLocalFocus(id); err != nil {
			return focusFail(output.ErrFileError, fmt.Errorf("writing %s: %w", config.LocalConfigFileName, err))
		}
		return writeFocus(cmd.OutOrStdout(), resolveFocus(all))
	},
}

var focusClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove the focus from " + config.LocalConfigFileName,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.ClearLocalFocus(); err != nil {
			return focusFail(output.ErrFileError, fmt.Errorf("removing %s: %w", config.LocalConfigFileName, err))
		}
		all, err := allBeans()
		if err != nil {
			return focusFail(output.ErrFileError, err)
		}
		return writeFocus(cmd.OutOrStdout(), resolveFocus(all))
	},
}

func allBeans() ([]*bean.Bean, error) {
	resolver := &beangraph.CoreResolver{Core: core}
	all, err := resolver.Beans(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("querying beans: %w", err)
	}
	return all, nil
}

func RegisterFocusCmd(root *cobra.Command) {
	focusCmd.PersistentFlags().BoolVar(&focusJSON, "json", false, "Output as JSON")
	focusCmd.AddCommand(focusSetCmd, focusClearCmd)
	root.AddCommand(focusCmd)
}

// focusSummary describes the focus setting for `beans focus`.
func focusSummary(f resolvedFocus) string {
	switch {
	case f.id == "":
		return "no focus set; every in-progress milestone or epic is a front"
	case f.problem != "":
		return fmt.Sprintf("focus %s (%s) is invalid: %s; every in-progress milestone or epic is a front", f.id, f.source, f.problem)
	default:
		return fmt.Sprintf("focus: %s %s %s (%s)", f.bean.ID, f.bean.Type, strings.TrimSpace(f.bean.Title), f.source)
	}
}
