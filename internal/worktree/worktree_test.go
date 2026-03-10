package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit
// and a .beans directory inside it.
func initTestRepo(t *testing.T) (repoDir, beansDir string) {
	t.Helper()
	dir := t.TempDir()

	commands := [][]string{
		{"git", "init"},
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
		name   string
		input  string
		want   int
		beanID string
	}{
		{
			name:   "empty",
			input:  "",
			want:   0,
			beanID: "",
		},
		{
			name: "main worktree only",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

`,
			want:   0,
			beanID: "",
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
			want:   1,
			beanID: "beans-a1b2",
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
			want:   1,
			beanID: "beans-x1y2",
		},
		{
			name: "no trailing newline",
			input: `worktree /tmp/repo
HEAD abc
branch refs/heads/main

worktree /tmp/repo-beans-foo
HEAD def
branch refs/heads/beans/beans-foo`,
			want:   1,
			beanID: "beans-foo",
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
			want:   1,
			beanID: "beans-good",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePorcelain(tt.input)
			if len(got) != tt.want {
				t.Fatalf("got %d worktrees, want %d", len(got), tt.want)
			}
			if tt.want > 0 && got[0].BeanID != tt.beanID {
				t.Errorf("got BeanID %q, want %q", got[0].BeanID, tt.beanID)
			}
		})
	}
}

func TestCreateAndList(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir)

	// List should be empty initially
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 0 {
		t.Fatalf("expected 0 worktrees, got %d", len(wts))
	}

	// Create a worktree
	wt, err := mgr.Create("beans-test1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if wt.BeanID != "beans-test1" {
		t.Errorf("BeanID = %q, want %q", wt.BeanID, "beans-test1")
	}
	if wt.Branch != "beans/beans-test1" {
		t.Errorf("Branch = %q, want %q", wt.Branch, "beans/beans-test1")
	}

	expectedPath := filepath.Join(beansDir, ".worktrees", "beans-test1")
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
	if wts[0].BeanID != "beans-test1" {
		t.Errorf("listed BeanID = %q, want %q", wts[0].BeanID, "beans-test1")
	}
}

func TestCreateDuplicate(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir)

	_, err := mgr.Create("beans-dup")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = mgr.Create("beans-dup")
	if err == nil {
		t.Fatal("expected error creating duplicate worktree")
	}
}

func TestRemove(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir)

	wt, err := mgr.Create("beans-rm")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mgr.Remove("beans-rm"); err != nil {
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
	mgr := NewManager(repoDir, beansDir)

	// Create a worktree, then delete its directory out from under git
	wt, err := mgr.Create("beans-stale")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := os.RemoveAll(wt.Path); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	// Remove should handle the stale entry gracefully via prune
	if err := mgr.Remove("beans-stale"); err != nil {
		t.Fatalf("Remove (stale): %v", err)
	}

	// List should be empty after pruning
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 0 {
		t.Fatalf("expected 0 worktrees after stale remove, got %d", len(wts))
	}
}

func TestCreateReusesExistingBranch(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir)

	// Create and then remove a worktree, leaving the branch behind
	_, err := mgr.Create("beans-reuse")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mgr.Remove("beans-reuse"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// The branch beans/beans-reuse still exists; creating again should succeed
	wt, err := mgr.Create("beans-reuse")
	if err != nil {
		t.Fatalf("Create (reuse): %v", err)
	}
	if wt.BeanID != "beans-reuse" {
		t.Errorf("BeanID = %q, want %q", wt.BeanID, "beans-reuse")
	}
	if _, err := os.Stat(wt.Path); err != nil {
		t.Errorf("worktree directory does not exist: %v", err)
	}
}

func TestSubscription(t *testing.T) {
	repoDir, beansDir := initTestRepo(t)
	mgr := NewManager(repoDir, beansDir)

	ch := mgr.Subscribe()
	defer mgr.Unsubscribe(ch)

	// Create should notify
	_, err := mgr.Create("beans-sub")
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
	if err := mgr.Remove("beans-sub"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	select {
	case <-ch:
		// Got notification
	default:
		t.Error("expected notification after Remove")
	}
}
