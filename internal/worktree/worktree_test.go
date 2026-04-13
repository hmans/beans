package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hmans/beans/internal/testutil"
)

func TestParsePorcelain(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		worktreesDir string
		want         int
		id           string
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
			name:         "detached HEAD during rebase",
			worktreesDir: "/home/user/.beans/worktrees/project",
			input: `worktree /home/user/project
HEAD abc123
branch refs/heads/main

worktree /home/user/.beans/worktrees/project/beans-rebasing
HEAD def456
detached

`,
			want: 1,
			id:   "beans-rebasing",
		},
		{
			name:         "detached HEAD without worktreesDir is skipped",
			worktreesDir: "",
			input: `worktree /some/path
HEAD def456
detached

`,
			want: 0,
			id:   "",
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
			got := parsePorcelain(tt.input, tt.worktreesDir)
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
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

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

	expectedPath := filepath.Join(wtRoot, wt.ID)
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
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

	_, err := mgr.Create("")
	if err == nil {
		t.Fatal("expected error creating worktree with empty name")
	}
}

func TestRemove(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

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

func TestRemoveDirtyWorktree(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

	wt, err := mgr.Create("dirty-wt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Create an uncommitted change in the worktree
	if err := os.WriteFile(filepath.Join(wt.Path, "dirty.txt"), []byte("dirty"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, wt.Path, "add", "dirty.txt")

	// Remove should succeed even with uncommitted changes (uses --force)
	if err := mgr.Remove(wt.ID); err != nil {
		t.Fatalf("Remove dirty worktree: %v", err)
	}

	// Directory should be gone
	if _, err := os.Stat(wt.Path); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists after remove")
	}
}

func TestRemoveStaleWorktree(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

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
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

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
	repoDir, _, wtRoot := testutil.InitTestRepo(t)

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
	mgr := NewManager(repoDir, wtRoot, "other", "")
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

func TestFetchTimeoutDefault(t *testing.T) {
	mgr := NewManager("", "", "main", "")
	if mgr.fetchTimeout != DefaultFetchTimeout {
		t.Errorf("default fetchTimeout = %v, want %v", mgr.fetchTimeout, DefaultFetchTimeout)
	}
}

func TestFetchTimeoutCustom(t *testing.T) {
	mgr := NewManager("", "", "main", "", WithFetchTimeout(30*time.Second))
	if mgr.fetchTimeout != 30*time.Second {
		t.Errorf("fetchTimeout = %v, want 30s", mgr.fetchTimeout)
	}
}

func TestFetchTimeoutZeroSkipsFetch(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)

	// Create a manager with fetch timeout = 0 (disables fetch)
	mgr := NewManager(repoDir, wtRoot, "main", "", WithFetchTimeout(0))

	// Create should succeed without attempting to fetch
	wt, err := mgr.Create("skip-fetch-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if wt == nil {
		t.Fatal("expected worktree, got nil")
	}
}

func TestFetchBaseRefTimesOut(t *testing.T) {
	repoDir, _, _ := testutil.InitTestRepo(t)

	// Configure git to use a "fetch" command that just sleeps, simulating a hanging remote.
	// GIT_SSH_COMMAND is used by git when fetching over SSH.
	cmd := exec.Command("git", "remote", "add", "hangremote", "git@localhost:nonexistent.git")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("add remote: %s: %v", out, err)
	}

	// Override the SSH command to just sleep forever
	gitConfigCmd := exec.Command("git", "config", "core.sshCommand", "sleep 60")
	gitConfigCmd.Dir = repoDir
	if out, err := gitConfigCmd.CombinedOutput(); err != nil {
		t.Fatalf("config sshCommand: %s: %v", out, err)
	}

	mgr := NewManager(repoDir, "", "hangremote/main", "",
		WithFetchTimeout(1*time.Second),
	)

	// fetchBaseRef should complete within the timeout (not hang for 60s)
	start := time.Now()
	mgr.fetchBaseRef()
	elapsed := time.Since(start)

	// Should complete within ~3s (1s timeout + process cleanup overhead), not 60s
	if elapsed > 5*time.Second {
		t.Errorf("fetchBaseRef took %v, expected it to time out within ~1s", elapsed)
	}
}

func TestSubscription(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

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
	repoDir, beansDir, wtRoot := testutil.InitTestRepo(t)

	// Commit a file in .beans so the directory exists on main
	if err := os.WriteFile(filepath.Join(beansDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, repoDir, "add", ".beans/.gitkeep")
	gitRun(t, repoDir, "commit", "-m", "add .beans dir")

	mgr := NewManager(repoDir, wtRoot, "main", "")

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
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "main", "")

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
	repoDir, beansDir, wtRoot := testutil.InitTestRepo(t)

	// Commit a file in .beans so the directory exists on main
	if err := os.WriteFile(filepath.Join(beansDir, ".gitkeep"), []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, repoDir, "add", ".beans/.gitkeep")
	gitRun(t, repoDir, "commit", "-m", "add .beans dir")

	mgr := NewManager(repoDir, wtRoot, "main", "")

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
	repoDir, beansDir, wtRoot := testutil.InitTestRepo(t)

	// Create a bean on main
	if err := os.WriteFile(filepath.Join(beansDir, "beans-del1--to-delete.md"), []byte("---\ntitle: To Delete\nstatus: todo\ntype: task\n---\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, repoDir, "add", "-A")
	gitRun(t, repoDir, "commit", "-m", "add bean")

	mgr := NewManager(repoDir, wtRoot, "main", "")

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

func TestUpdateDescription(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

	// Create a worktree
	wt, err := mgr.Create("desc-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Subscribe to get notified of the update
	ch := mgr.Subscribe()
	defer mgr.Unsubscribe(ch)

	// Update the description
	if err := mgr.UpdateDescription(wt.ID, "Fix auth token refresh bug"); err != nil {
		t.Fatalf("UpdateDescription: %v", err)
	}

	// Should have notified
	select {
	case <-ch:
	default:
		t.Error("expected notification after UpdateDescription")
	}

	// List should return the description
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	if wts[0].Description != "Fix auth token refresh bug" {
		t.Errorf("Description = %q, want %q", wts[0].Description, "Fix auth token refresh bug")
	}

	// The name should still be preserved
	if wts[0].Name != "desc-test" {
		t.Errorf("Name = %q, want %q", wts[0].Name, "desc-test")
	}
}

func TestCreateRunsSetupCommand(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)

	// Use a setup command that creates a marker file
	mgr := NewManager(repoDir, wtRoot, "", "touch .setup-done")

	// Track setup completion via callback
	done := make(chan bool, 1)
	mgr.SetOnSetupDone(func(id string, success bool, errMsg string) {
		done <- success
	})

	wt, err := mgr.Create("setup-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Setup status should be running immediately after Create
	if wt.Setup != SetupRunning {
		t.Errorf("expected setup status 'running', got %q", wt.Setup)
	}

	// Wait for async setup to complete
	select {
	case success := <-done:
		if !success {
			t.Error("setup command failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for setup command")
	}

	// The setup command should have created the marker file in the worktree
	markerPath := filepath.Join(wt.Path, ".setup-done")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Error("setup command did not run: .setup-done file not found")
	}

	// Verify setup status is updated in List()
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, w := range wts {
		if w.ID == wt.ID && w.Setup != SetupDone {
			t.Errorf("expected setup status 'done' in List(), got %q", w.Setup)
		}
	}
}

func TestCreateNoSetupCommand(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)

	// No setup command — should still create fine
	mgr := NewManager(repoDir, wtRoot, "", "")

	wt, err := mgr.Create("no-setup-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Just verify the worktree was created
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Error("worktree directory not created")
	}
}

func TestTouchLastActive(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

	// Create a worktree
	wt, err := mgr.Create("touch-test")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Subscribe to get notified
	ch := mgr.Subscribe()
	defer mgr.Unsubscribe(ch)

	// Touch the last active timestamp
	if err := mgr.TouchLastActive(wt.ID); err != nil {
		t.Fatalf("TouchLastActive: %v", err)
	}

	// Should have notified subscribers
	select {
	case <-ch:
	default:
		t.Error("expected notification after TouchLastActive")
	}

	// List should return the updated timestamp
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	if wts[0].LastActiveAt.IsZero() {
		t.Error("expected LastActiveAt to be set after TouchLastActive")
	}

	// The name should still be preserved
	if wts[0].Name != "touch-test" {
		t.Errorf("Name = %q, want %q", wts[0].Name, "touch-test")
	}
}

func TestListKeepsCreationOrder(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

	// Create three worktrees in a specific order
	_, err := mgr.Create("wt-alpha")
	if err != nil {
		t.Fatalf("Create alpha: %v", err)
	}
	_, err = mgr.Create("wt-beta")
	if err != nil {
		t.Fatalf("Create beta: %v", err)
	}
	_, err = mgr.Create("wt-gamma")
	if err != nil {
		t.Fatalf("Create gamma: %v", err)
	}

	// Touch them in a different order — should NOT affect listing order
	time.Sleep(10 * time.Millisecond)
	if err := mgr.TouchLastActive("wt-gamma"); err != nil {
		t.Fatalf("TouchLastActive gamma: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := mgr.TouchLastActive("wt-alpha"); err != nil {
		t.Fatalf("TouchLastActive alpha: %v", err)
	}

	// List should preserve creation order: alpha, beta, gamma
	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(wts))
	}

	if wts[0].ID != "wt-alpha" {
		t.Errorf("wts[0].ID = %q, want %q (created first)", wts[0].ID, "wt-alpha")
	}
	if wts[1].ID != "wt-beta" {
		t.Errorf("wts[1].ID = %q, want %q (created second)", wts[1].ID, "wt-beta")
	}
	if wts[2].ID != "wt-gamma" {
		t.Errorf("wts[2].ID = %q, want %q (created third)", wts[2].ID, "wt-gamma")
	}
}

func TestCreateSetsLastActiveAt(t *testing.T) {
	repoDir, _, wtRoot := testutil.InitTestRepo(t)
	mgr := NewManager(repoDir, wtRoot, "", "")

	before := time.Now().UTC().Add(-time.Second)
	_, err := mgr.Create("new-wt")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wts, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(wts) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(wts))
	}
	if wts[0].LastActiveAt.Before(before) {
		t.Errorf("expected LastActiveAt >= %v, got %v", before, wts[0].LastActiveAt)
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
