package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initStatusTestRepo creates a test repo with a tracked README.md file.
func initStatusTestRepo(t *testing.T) string {
	t.Helper()
	dir := initTestRepo(t)

	// Add a tracked file so we can test modifications
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", "add readme")

	return dir
}

// gitRun runs a git command in the given directory.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, out)
	}
}

func TestFileChanges_Unstaged(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Modify tracked file
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\nline 2\nline 3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := FileChanges(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}

	c := changes[0]
	if c.Path != "README.md" {
		t.Errorf("expected path README.md, got %s", c.Path)
	}
	if c.Status != "modified" {
		t.Errorf("expected status modified, got %s", c.Status)
	}
	if c.Staged {
		t.Error("expected unstaged")
	}
	if c.Additions != 2 {
		t.Errorf("expected 2 additions, got %d", c.Additions)
	}
}

func TestFileChanges_Staged(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Create and stage a new file
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "new.txt")

	changes, err := FileChanges(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}

	c := changes[0]
	if c.Path != "new.txt" {
		t.Errorf("expected path new.txt, got %s", c.Path)
	}
	if !c.Staged {
		t.Error("expected staged")
	}
	if c.Additions != 2 {
		t.Errorf("expected 2 additions, got %d", c.Additions)
	}
}

func TestFileChanges_Untracked(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Create untracked file
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := FileChanges(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}

	c := changes[0]
	if c.Path != "untracked.txt" {
		t.Errorf("expected path untracked.txt, got %s", c.Path)
	}
	if c.Status != "untracked" {
		t.Errorf("expected status untracked, got %s", c.Status)
	}
	if c.Staged {
		t.Error("expected not staged")
	}
}

func TestFileChanges_Mixed(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Unstaged modification
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Changed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Staged new file
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "staged.txt")

	// Untracked file
	if err := os.WriteFile(filepath.Join(dir, "extra.txt"), []byte("extra\n"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := FileChanges(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d: %+v", len(changes), changes)
	}

	// Staged should come first (cached), then unstaged, then untracked
	byPath := make(map[string]FileChange)
	for _, c := range changes {
		byPath[c.Path] = c
	}

	if c, ok := byPath["staged.txt"]; !ok || !c.Staged {
		t.Error("expected staged.txt to be staged")
	}
	if c, ok := byPath["README.md"]; !ok || c.Staged {
		t.Error("expected README.md to be unstaged")
	}
	if c, ok := byPath["extra.txt"]; !ok || c.Status != "untracked" {
		t.Error("expected extra.txt to be untracked")
	}
}

func TestFileChanges_Empty(t *testing.T) {
	dir := initStatusTestRepo(t)

	changes, err := FileChanges(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(changes) != 0 {
		t.Fatalf("expected 0 changes, got %d: %+v", len(changes), changes)
	}
}

func TestFileDiff_Unstaged(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Modify tracked file
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\nline 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	diff, err := FileDiff(dir, "README.md", false)
	if err != nil {
		t.Fatal(err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(diff, "+line 2") {
		t.Errorf("expected diff to contain '+line 2', got:\n%s", diff)
	}
}

func TestFileDiff_Staged(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Create and stage a new file
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "new.txt")

	diff, err := FileDiff(dir, "new.txt", true)
	if err != nil {
		t.Fatal(err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(diff, "+hello") {
		t.Errorf("expected diff to contain '+hello', got:\n%s", diff)
	}
}

func TestFileDiff_Untracked(t *testing.T) {
	dir := initStatusTestRepo(t)

	// Create untracked file
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	diff, err := FileDiff(dir, "untracked.txt", false)
	if err != nil {
		t.Fatal(err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff for untracked file")
	}
	if !strings.Contains(diff, "+data") {
		t.Errorf("expected diff to contain '+data', got:\n%s", diff)
	}
}

// initBranchedTestRepo creates a test repo with a remote "origin" and a feature
// branch that has diverged from main. Returns the worktree directory of the
// feature branch. The main branch has README.md; the feature branch adds
// feature.txt on top.
func initBranchedTestRepo(t *testing.T) string {
	t.Helper()

	// Create the "remote" bare repo
	bare := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = bare
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %s: %v", out, err)
	}

	// Clone it to create a working repo
	dir := t.TempDir()
	cmd = exec.Command("git", "clone", bare, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone failed: %s: %v", out, err)
	}

	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")

	// Create initial commit on main
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", "initial")
	gitRun(t, dir, "push", "-u", "origin", "HEAD")

	// Set origin/HEAD so MergeBase can find the default branch
	gitRun(t, dir, "remote", "set-head", "origin", "--auto")

	// Create a feature branch with a committed change
	gitRun(t, dir, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "feature.txt")
	gitRun(t, dir, "commit", "-m", "add feature")

	return dir
}

func TestMergeBase(t *testing.T) {
	dir := initBranchedTestRepo(t)

	base, ok := MergeBase(dir, "main")
	if !ok {
		t.Fatal("expected MergeBase to succeed")
	}
	if base == "" {
		t.Fatal("expected non-empty merge-base")
	}
}

func TestMergeBase_FallbackToRemote(t *testing.T) {
	dir := initBranchedTestRepo(t)

	// With empty baseRef, should fall back to origin's default branch
	base, ok := MergeBase(dir, "")
	if !ok {
		t.Fatal("expected MergeBase fallback to succeed")
	}
	if base == "" {
		t.Fatal("expected non-empty merge-base")
	}
}

func TestAllChangesVsUpstream_CommittedOnly(t *testing.T) {
	dir := initBranchedTestRepo(t)

	changes, err := AllChangesVsUpstream(dir, "main")
	if err != nil {
		t.Fatal(err)
	}

	// Should show feature.txt as a new file (committed on feature branch)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}
	if changes[0].Path != "feature.txt" {
		t.Errorf("expected feature.txt, got %s", changes[0].Path)
	}
	if changes[0].Status != "added" {
		t.Errorf("expected status added, got %s", changes[0].Status)
	}
}

func TestAllChangesVsUpstream_CommittedPlusUnstaged(t *testing.T) {
	dir := initBranchedTestRepo(t)

	// Add an unstaged modification
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\nmodified\n"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := AllChangesVsUpstream(dir, "main")
	if err != nil {
		t.Fatal(err)
	}

	// Should show feature.txt (committed) + README.md (unstaged)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}

	byPath := make(map[string]FileChange)
	for _, c := range changes {
		byPath[c.Path] = c
	}

	if _, ok := byPath["feature.txt"]; !ok {
		t.Error("expected feature.txt in changes")
	}
	if c, ok := byPath["README.md"]; !ok {
		t.Error("expected README.md in changes")
	} else if c.Status != "modified" {
		t.Errorf("expected README.md status modified, got %s", c.Status)
	}
}

func TestAllChangesVsUpstream_WithUntracked(t *testing.T) {
	dir := initBranchedTestRepo(t)

	// Add an untracked file
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	changes, err := AllChangesVsUpstream(dir, "main")
	if err != nil {
		t.Fatal(err)
	}

	// Should show feature.txt (committed) + untracked.txt
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d: %+v", len(changes), changes)
	}

	byPath := make(map[string]FileChange)
	for _, c := range changes {
		byPath[c.Path] = c
	}

	if _, ok := byPath["feature.txt"]; !ok {
		t.Error("expected feature.txt in changes")
	}
	if c, ok := byPath["untracked.txt"]; !ok {
		t.Error("expected untracked.txt in changes")
	} else if c.Status != "untracked" {
		t.Errorf("expected untracked.txt status untracked, got %s", c.Status)
	}
}

func TestAllFileDiff_CommittedFile(t *testing.T) {
	dir := initBranchedTestRepo(t)

	diff, err := AllFileDiff(dir, "feature.txt", "main")
	if err != nil {
		t.Fatal(err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(diff, "+feature content") {
		t.Errorf("expected diff to contain '+feature content', got:\n%s", diff)
	}
}

func TestAllFileDiff_UntrackedFile(t *testing.T) {
	dir := initBranchedTestRepo(t)

	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	diff, err := AllFileDiff(dir, "untracked.txt", "main")
	if err != nil {
		t.Fatal(err)
	}

	if diff == "" {
		t.Fatal("expected non-empty diff")
	}
	if !strings.Contains(diff, "+data") {
		t.Errorf("expected diff to contain '+data', got:\n%s", diff)
	}
}

func TestParseNumstat(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		staged bool
		want   []FileChange
	}{
		{
			name:  "empty",
			input: "",
			want:  nil,
		},
		{
			name:   "single file",
			input:  "10\t5\tsrc/main.go",
			staged: false,
			want: []FileChange{
				{Path: "src/main.go", Status: "modified", Additions: 10, Deletions: 5, Staged: false},
			},
		},
		{
			name:   "binary file",
			input:  "-\t-\timage.png",
			staged: true,
			want: []FileChange{
				{Path: "image.png", Status: "modified", Additions: 0, Deletions: 0, Staged: true},
			},
		},
		{
			name:   "rename",
			input:  "0\t0\told.txt => new.txt",
			staged: true,
			want: []FileChange{
				{Path: "old.txt => new.txt", Status: "renamed", Additions: 0, Deletions: 0, Staged: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNumstat(tt.input, tt.staged)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d changes, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("change[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
