// Package gittest makes git commands run by tests hermetic: they act only on
// the temp-dir repositories the tests create, see no user or system config,
// and cannot reach the network.
package gittest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Isolate rewrites the process environment for every git command started
// afterwards, by the tests or by the code under test. Call it from TestMain
// before m.Run.
//
// Git ranks GIT_DIR and its relatives above the working directory, and git
// itself sets them for hooks, `rebase --exec` and `!`-aliases; Isolate unsets
// every variable `git rev-parse --local-env-vars` lists. It also hides the
// global and system config, stops repository discovery at the temp root,
// and allows only the file protocol.
func Isolate() {
	out, err := exec.Command("git", "rev-parse", "--local-env-vars").Output()
	if err != nil {
		panic(fmt.Sprintf("gittest: git rev-parse --local-env-vars: %v", err))
	}
	for _, name := range strings.Fields(string(out)) {
		mustDo(os.Unsetenv(name))
	}
	for name, value := range map[string]string{
		"GIT_CONFIG_GLOBAL":       os.DevNull,
		"GIT_CONFIG_NOSYSTEM":     "1",
		"GIT_CEILING_DIRECTORIES": os.TempDir(),
		"GIT_ALLOW_PROTOCOL":      "file",
	} {
		mustDo(os.Setenv(name, value))
	}
}

func mustDo(err error) {
	if err != nil {
		panic("gittest: " + err.Error())
	}
}

// RequireGitDirInside fails the test unless the git directory that git
// resolves for dir lies inside dir. Call it right after creating a repository.
func RequireGitDirInside(t testing.TB, dir string) {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--absolute-git-dir")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --absolute-git-dir in %s: %v", dir, err)
	}
	gitDir := realPath(t, strings.TrimSpace(string(out)))
	root := realPath(t, dir)
	if gitDir != root && !strings.HasPrefix(gitDir, root+string(filepath.Separator)) {
		t.Fatalf("git directory %s lies outside the test repository %s", gitDir, root)
	}
}

func realPath(t testing.TB, path string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", path, err)
	}
	return p
}
