package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit
// and a .beans directory inside it.
func initTestRepo(t *testing.T) (repoDir, beansDir string) {
	t.Helper()
	dir := t.TempDir()

	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	bd := filepath.Join(dir, ".beans")
	if err := os.MkdirAll(bd, 0755); err != nil {
		t.Fatalf("MkdirAll .beans: %v", err)
	}

	return dir, bd
}

func TestParsePorcelain(t *testing.T) {
	tests := []struct {
		name string
		input string
		want int
		id   string
	}{
		{
			name:  "empty",
			input: "",
			want:  0,
			id:    "",
		},
		{
			name: "main worktree only",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

`,
			want: 0,
			id:   "",
		},
		{
			name: "one beans worktree",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-beans-a1b2
HEAD def456
branch refs/heads/beans/beans-a1b2

`,
			want: 1,
			id:   "beans-a1b2",
		},
		{
			name: "mixed worktrees",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-feature
HEAD def456
branch refs/heads/feature/login

worktree /home/user/project-beans-x1y2
HEAD ghi789
branch refs/heads/beans/beans-x1y2

`,
			want: 1,
			id:   "beans-x1y2",
		},
		{
			name: "no trailing newline",
			input: `worktree /tmp/repo
HEAD abc
branch refs/heads/main

worktree /tmp/repo-beans-foo
HEAD def
branch refs/heads/beans/beans-foo`,
			want: 1,
			id:   "beans-foo",
		},
		{
			name: "prunable entry is skipped",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/project-beans-stale
HEAD def456
branch refs/heads/beans/beans-stale
prunable gitdir file points to non-existent location

worktree /home/user/project-beans-good
HEAD ghi789
branch refs/heads/beans/beans-good

`,
			want: 1,
			id:   "beans-good",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePorcelain(tt.input)
			if len(got) != tt.want {
				t.Fatalf("got %d worktrees, want %d", len(got), tt.want)
			}
			if tt.want > 0 && got[0].ID != tt.id {
				t.Errorf("got ID %q, want %q", got[0].ID, tt.id)
			}
		})
	}
}

func TestCreateAndList(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "")

	// List should be empty initially
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 0 {
		t.Fatalf("expected 0 worktrees, got %d", len(wts))
	}

	// Create a worktree
	wt, err := mgr.Create("test-worktree")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if wt.ID != "test-worktree" {
		t.Errorf("ID = %q, want %q", wt.ID, "test-worktree")
	}
	if wt.Name != "test-worktree" {
		t.Errorf("Name = %q, want %q", wt.Name, "test-worktree")
	}
	if wt.Branch != "beans/test-worktree" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "beans/test-worktree")
	}

	expectedPath := filepath.Join(beansDir, ".worktrees", wt.ID)
	if wt.Path != expectedPath {
		t.Errorf("Path = %q, want %q", wt.Path, expectedPath)
	}

	// Verify the directory exists
	if _, err := os.Stat(wt.Path); err != nil {
		t.Errorf("worktree directory does not exist: %v", err)
	}

	// List should now return 1
	wts, err = mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	if wts[0].ID != wt.ID {
		t.Errorf("listed ID = %q, want %q", wts[0].ID, wt.ID)
	}
	if wts[0].Name != "test-worktree" {
		t.Errorf("listed Name = %q, want %q", wts[0].Name, "test-worktree")
	}
}

func TestCreateEmptyName(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "")

	_, err := mgr.Create("")
	if err == nil {
		t.Fatal("expected error creating worktree with empty name")
	}
}

func TestRemove(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "")

	wt, err := mgr.Create("to-remove")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mgr.Remove(wt.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Directory should be gone
	if _, err := os.Stat(wt.Path); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists after remove")
	}

	// List should be empty
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 0 {
		t.Fatalf("expected 0 worktrees after remove, got %d", len(wts))
	}
}

func TestRemoveStaleWorktree(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "")

	// Create a worktree, then delete its directory out from under git
	wt, err := mgr.Create("to-stale")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := os.RemoveAll(wt.Path); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	// Remove should return an error for the stale worktree (no implicit prune)
	if err := mgr.Remove(wt.ID); err == nil {
		t.Fatal("expected error removing stale worktree, got nil")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "")

	// Remove a worktree that doesn't exist should return an error
	err := mgr.Remove("wt-nonexistent")
	if err == nil {
		t.Fatal("expected error removing nonexistent worktree, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestCreateUsesBaseRef(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)

	// Create a second commit on a new branch so we have a distinct ref to branch from
	commands := [][]string{
		{"git", "checkout", "-b", "other"},
		{"git", "commit", "--allow-empty", "-m", "other commit"},
		{"git", "checkout", "main"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	// Get the commit SHA of the "other" branch
	otherSHA := exec.Command("git", "rev-parse", "other")
	otherSHA.Dir = repoDir
	otherOut, err := otherSHA.Output()
	if err != nil {
		t.Fatalf("rev-parse other: %v", err)
	}
	otherCommit := strings.TrimSpace(string(otherOut))

	// Create a worktree manager with baseRef pointing to "other"
	mgr := NewManager(repoDir, beansDir, "other")
	wt, err := mgr.Create("baseref-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The worktree's HEAD should match the "other" branch commit, not main
	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headCmd.Dir = wt.Path
	headOut, err := headCmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD in worktree: %v", err)
	}
	wtCommit := strings.TrimSpace(string(headOut))

	if wtCommit != otherCommit {
		t.Errorf("worktree HEAD = %s, want %s (from base ref 'other')", wtCommit, otherCommit)
	}
}

func TestSubscription(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "")

	ch := mgr.Subscribe()
	defer mgr.Unsubscribe(ch)

	// Create should notify
	wt, err := mgr.Create("sub-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	select {
	case <-ch:
		// Got notification
	default:
		t.Error("expected notification after Create")
	}

	// Remove should notify
	if err := mgr.Remove(wt.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	select {
	case <-ch:
		// Got notification
	default:
		t.Error("expected notification after Remove")
	}
}

func TestDetectBeanIDs(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)

	// Commit a file in .beans so the directory exists on main
	if err := os.WriteFile(filepath.Join(beansDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, repoDir, "add", ".beans/.gitkeep")
	gitRun(t, repoDir, "commit", "-m", "add .beans dir")

	mgr := NewManager(repoDir, beansDir, "main")

	// Create a worktree
	wt, err := mgr.Create("detect-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create bean files in the worktree's .beans/ directory
	wtBeansDir := filepath.Join(wt.Path, ".beans")
	if err := os.MkdirAll(wtBeansDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Add a bean file
	if err := os.WriteFile(filepath.Join(wtBeansDir, "beans-abc1--my-task.md"), []byte("---\ntitle: My Task\n---\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Add another bean file
	if err := os.WriteFile(filepath.Join(wtBeansDir, "beans-def2--another-task.md"), []byte("---\ntitle: Another Task\n---\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Add a file in .beans/.worktrees/ (should be excluded)
	wtSubDir := filepath.Join(wtBeansDir, ".worktrees")
	if err := os.MkdirAll(wtSubDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtSubDir, "meta.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Add a file in .beans/archive/ (should be excluded)
	archiveDir := filepath.Join(wtBeansDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "beans-old1.md"), []byte("---\ntitle: Old\n---\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Add a non-.md file (should be excluded)
	if err := os.WriteFile(filepath.Join(wtBeansDir, "config.yaml"), []byte("key: val"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Stage and commit the bean files
	gitRun(t, wt.Path, "add", "-A")
	gitRun(t, wt.Path, "commit", "-m", "add beans")

	ids := mgr.DetectBeanIDs(wt.Path)

	// Should find exactly the two bean files, sorted
	if len(ids) != 2 {
		t.Fatalf("got %d IDs, want 2: %v", len(ids), ids)
	}
	if ids[0] != "beans-abc1" {
		t.Errorf("ids[0] = %q, want %q", ids[0], "beans-abc1")
	}
	if ids[1] != "beans-def2" {
		t.Errorf("ids[1] = %q, want %q", ids[1], "beans-def2")
	}
}

func TestDetectBeanIDs_NoChanges(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir, "main")

	// Create a worktree with no bean changes
	wt, err := mgr.Create("no-changes")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ids := mgr.DetectBeanIDs(wt.Path)
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs for worktree with no changes, got %d: %v", len(ids), ids)
	}
}

func TestDetectBeanIDs_UncommittedChanges(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)

	// Commit a file in .beans so the directory exists on main
	if err := os.WriteFile(filepath.Join(beansDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, repoDir, "add", ".beans/.gitkeep")
	gitRun(t, repoDir, "commit", "-m", "add .beans dir")

	mgr := NewManager(repoDir, beansDir, "main")

	wt, err := mgr.Create("uncommitted-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Add an untracked bean file (not committed)
	wtBeansDir := filepath.Join(wt.Path, ".beans")
	if err := os.MkdirAll(wtBeansDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wtBeansDir, "beans-xyz9--untracked.md"), []byte("---\ntitle: Untracked\n---\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ids := mgr.DetectBeanIDs(wt.Path)

	// Should detect untracked bean files too
	if len(ids) != 1 {
		t.Fatalf("got %d IDs, want 1: %v", len(ids), ids)
	}
	if ids[0] != "beans-xyz9" {
		t.Errorf("ids[0] = %q, want %q", ids[0], "beans-xyz9")
	}
}

func TestDetectBeanIDs_DeletedFile(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)

	// Create a bean on main
	if err := os.WriteFile(filepath.Join(beansDir, "beans-del1--to-delete.md"), []byte("---\ntitle: To Delete\nstatus: todo\ntype: task\n---\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, repoDir, "add", "-A")
	gitRun(t, repoDir, "commit", "-m", "add bean")

	mgr := NewManager(repoDir, beansDir, "main")

	wt, err := mgr.Create("delete-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Delete the bean file in the worktree
	wtBeanFile := filepath.Join(wt.Path, ".beans", "beans-del1--to-delete.md")
	if err := os.Remove(wtBeanFile); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	gitRun(t, wt.Path, "add", "-A")
	gitRun(t, wt.Path, "commit", "-m", "delete bean")

	ids := mgr.DetectBeanIDs(wt.Path)

	// Should NOT include the deleted bean
	if len(ids) != 0 {
		t.Errorf("expected 0 IDs for deleted bean, got %d: %v", len(ids), ids)
	}
}

// gitRun runs a git command in the given directory, failing the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s: %v", args, out, err)
	}
}
