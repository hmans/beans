package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	Isolate()
	os.Exit(m.Run())
}

// isolateRestoring runs Isolate and restores every variable it touches when
// the test ends, so tests that change the environment first do not leak it.
func isolateRestoring(t *testing.T) {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--local-env-vars").Output()
	if err != nil {
		t.Fatal(err)
	}
	names := append(strings.Fields(string(out)),
		"GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_CEILING_DIRECTORIES", "GIT_ALLOW_PROTOCOL")
	for _, name := range names {
		value, set := os.LookupEnv(name)
		t.Setenv(name, value)
		if !set {
			os.Unsetenv(name)
		}
	}
	Isolate()
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return string(out)
}

// snapshot captures the refs and the local config of a repository.
func snapshot(t *testing.T, gitDir string) string {
	t.Helper()
	config, err := os.ReadFile(filepath.Join(gitDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	return git(t, gitDir, "for-each-ref") + "\n" + string(config)
}

func TestIsolateProtectsRepositoryBehindGitDir(t *testing.T) {
	sentinel := t.TempDir()
	git(t, sentinel, "init", "-q", "-b", "main")
	git(t, sentinel, "-c", "user.name=S", "-c", "user.email=s@s", "commit", "-q", "--allow-empty", "-m", "sentinel")
	sentinelGitDir := filepath.Join(sentinel, ".git")
	before := snapshot(t, sentinelGitDir)

	t.Setenv("GIT_DIR", sentinelGitDir)
	t.Setenv("GIT_WORK_TREE", sentinel)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(sentinelGitDir, "index"))
	isolateRestoring(t)

	repo := t.TempDir()
	git(t, repo, "init", "-q", "-b", "main")
	RequireGitDirInside(t, repo)
	git(t, repo, "config", "user.name", "Test")
	git(t, repo, "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "test")
	git(t, repo, "branch", "feature")

	if after := snapshot(t, sentinelGitDir); after != before {
		t.Errorf("repository behind GIT_DIR changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestIsolateHidesGlobalConfig(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[push]\n\tpushOption = ci.skip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	isolateRestoring(t)

	cmd := exec.Command("git", "config", "--get", "push.pushOption")
	cmd.Dir = t.TempDir()
	if out, err := cmd.Output(); err == nil {
		t.Errorf("global config visible after Isolate: push.pushOption = %q", strings.TrimSpace(string(out)))
	}
}

func TestIsolateBlocksNetworkTransports(t *testing.T) {
	isolateRestoring(t)

	cmd := exec.Command("git", "ls-remote", "https://example.invalid/repo.git")
	cmd.Dir = t.TempDir()
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "not allowed") {
		t.Errorf("https transport not blocked: err=%v output=%s", err, out)
	}
}

func TestIsolateStopsDiscoveryAtTempRoot(t *testing.T) {
	enclosing := t.TempDir()
	git(t, enclosing, "init", "-q")
	tmpRoot := filepath.Join(enclosing, "tmp")
	nonRepo := filepath.Join(tmpRoot, "test")
	if err := os.MkdirAll(nonRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmpRoot)
	isolateRestoring(t)

	cmd := exec.Command("git", "rev-parse", "--absolute-git-dir")
	cmd.Dir = nonRepo
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Errorf("git discovered a repository above the temp root: %s", out)
	}
}

func TestRequireGitDirInsideRejectsForeignGitDir(t *testing.T) {
	outside := t.TempDir()
	git(t, outside, "init", "-q")
	dir := t.TempDir()
	t.Setenv("GIT_DIR", filepath.Join(outside, ".git"))

	rec := &fatalRecorder{TB: t}
	RequireGitDirInside(rec, dir)
	if !rec.failed {
		t.Error("RequireGitDirInside accepted a git directory outside the test repository")
	}
}

type fatalRecorder struct {
	testing.TB
	failed bool
}

func (r *fatalRecorder) Helper() {}

func (r *fatalRecorder) Fatalf(format string, args ...any) { r.failed = true }
