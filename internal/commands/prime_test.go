package commands

import (
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

// Register* adds flags to package-level commands, so a second call in the
// same process (go test -count=2) panics on a redefined flag.
var (
	testRootOnce sync.Once
	testRoot     *cobra.Command
)

func registeredRoot() *cobra.Command {
	testRootOnce.Do(func() {
		testRoot = &cobra.Command{Use: "beans"}
		RegisterCoreCommands(testRoot)
	})
	return testRoot
}

// plannedCommands are workflow commands proposed in beans-mmyp. A bare
// backticked mention of one reads as a claim that the command exists.
var plannedCommands = []string{"blocked", "complete", "milestones", "progress", "ready", "scrap", "start"}

func renderPrime(t *testing.T, template string) string {
	t.Helper()
	oldTemplate, oldConfigPath := agentPromptTemplate, configPath
	agentPromptTemplate = template
	configPath = "set" // skips the upward search for .beans.yml
	defer func() { agentPromptTemplate, configPath = oldTemplate, oldConfigPath }()
	return string(captureStdout(t, func() {
		if err := primeCmd.RunE(primeCmd, nil); err != nil {
			t.Fatal(err)
		}
	}))
}

// commandPath returns the leading command words of a `beans ...` usage,
// stopping at the first flag, placeholder, quote or variable.
func commandPath(usage string) []string {
	var path []string
	for _, w := range strings.Fields(usage) {
		if !regexp.MustCompile(`^[a-z][a-z-]*$`).MatchString(w) {
			break
		}
		path = append(path, w)
	}
	return path
}

// primeProblems lists every command the prompt names that does not exist.
func primeProblems(out string) []string {
	root := registeredRoot()
	var problems []string
	usages := regexp.MustCompile("(?m)`beans ([^`]*)`|^beans (.*)$").FindAllStringSubmatch(out, -1)
	for _, m := range usages {
		path := commandPath(m[1] + m[2])
		if len(path) == 0 {
			continue
		}
		cmd, rest, err := root.Find(path)
		switch {
		case err != nil || cmd == root:
			problems = append(problems, "beans "+strings.Join(path, " "))
		case len(rest) > 0 && cmd.HasSubCommands():
			problems = append(problems, "beans "+strings.Join(path, " "))
		}
	}
	for _, name := range plannedCommands {
		if c, _, err := root.Find([]string{name}); err == nil && c != root {
			continue
		}
		if strings.Contains(out, "`"+name+"`") {
			problems = append(problems, "`"+name+"`")
		}
	}
	return problems
}

func TestPrimeNamesOnlyExistingCommands(t *testing.T) {
	out := renderPrime(t, agentPromptTemplate)
	for _, p := range primeProblems(out) {
		t.Errorf("prompt names %s, which is not a command", p)
	}
	if !strings.Contains(out, "beans next --json") {
		t.Error("prompt does not point agents at `beans next --json`")
	}
}

func TestPrimeProblemsDetectsMissingCommands(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{"existing command and alias", "`beans next --json` and\nbeans query '{}'\n", nil},
		{"existing subcommand", "`beans focus set <id>`", nil},
		{"unknown command", "`beans ready --json`", []string{"beans ready"}},
		{"unknown subcommand", "beans focus sett <id>\n", []string{"beans focus sett"}},
		{"bare planned command", "Commands like `ready`, `next`, and `start` respect this.", []string{"`ready`", "`start`"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := primeProblems(tt.prompt)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("problems = %q, want %q", got, tt.want)
			}
		})
	}
}
