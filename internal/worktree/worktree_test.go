package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a temporary git repo with an initial commit.
func initTestRepo(t *testing.T) string {
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

	return dir
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
	repoDir := initTestRepo(t)
	mgr := NewManager(repoDir)

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

	expectedPath := filepath.Join(filepath.Dir(repoDir), filepath.Base(repoDir)+"-beans-test1")
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
	repoDir := initTestRepo(t)
	mgr := NewManager(repoDir)

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
	repoDir := initTestRepo(t)
	mgr := NewManager(repoDir)

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

func TestSubscription(t *testing.T) {
	repoDir := initTestRepo(t)
	mgr := NewManager(repoDir)

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
