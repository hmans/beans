package forge

import (
	"testing"
)

func TestIsGitHub(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"github.com SSH", "git@github.com:org/repo.git", true},
		{"github.com HTTPS", "https://github.com/org/repo.git", true},
		{"GitHub Enterprise SSH", "git@github.corp.co:org/repo.git", true},
		{"GitHub Enterprise HTTPS", "https://github.example.com/org/repo.git", true},
		{"GitLab", "git@gitlab.com:org/repo.git", false},
		{"Bitbucket", "git@bitbucket.org:org/repo.git", false},
		{"empty", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isGitHub(tc.url)
			if got != tc.expected {
				t.Errorf("isGitHub(%q) = %v, want %v", tc.url, got, tc.expected)
			}
		})
	}
}

func TestNormalizeState(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"OPEN", "open"},
		{"CLOSED", "closed"},
		{"MERGED", "merged"},
		{"open", "open"},
		{"something", "something"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeState(tc.input)
			if got != tc.expected {
				t.Errorf("normalizeState(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestFormatPRRef(t *testing.T) {
	pr := &PullRequest{Number: 42}
	got := FormatPRRef(pr)
	if got != "#42" {
		t.Errorf("FormatPRRef() = %q, want %q", got, "#42")
	}
}

func TestCanMerge(t *testing.T) {
	tests := []struct {
		name     string
		pr       PullRequest
		expected bool
	}{
		{
			"all green",
			PullRequest{Checks: CheckStatusPass, ReviewApproved: true, Mergeable: true},
			true,
		},
		{
			"draft PR",
			PullRequest{IsDraft: true, Checks: CheckStatusPass, ReviewApproved: true, Mergeable: true},
			false,
		},
		{
			"checks failing",
			PullRequest{Checks: CheckStatusFail, ReviewApproved: true, Mergeable: true},
			false,
		},
		{
			"checks pending",
			PullRequest{Checks: CheckStatusPending, ReviewApproved: true, Mergeable: true},
			false,
		},
		{
			"not mergeable",
			PullRequest{Checks: CheckStatusPass, ReviewApproved: true, Mergeable: false},
			false,
		},
		{
			"zero value",
			PullRequest{},
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.pr.CanMerge()
			if got != tc.expected {
				t.Errorf("CanMerge() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantOK    bool
	}{
		{"SSH github.com", "git@github.com:owner/repo.git", "owner", "repo", true},
		{"SSH no .git suffix", "git@github.com:owner/repo", "owner", "repo", true},
		{"HTTPS github.com", "https://github.com/owner/repo.git", "owner", "repo", true},
		{"HTTPS no .git suffix", "https://github.com/owner/repo", "owner", "repo", true},
		{"HTTP", "http://github.com/owner/repo.git", "owner", "repo", true},
		{"GitHub Enterprise SSH", "git@github.corp.co:org/project.git", "org", "project", true},
		{"GitHub Enterprise HTTPS", "https://github.example.com/org/project.git", "org", "project", true},
		{"empty", "", "", "", false},
		{"no path", "https://github.com", "", "", false},
		{"single path segment", "https://github.com/owner", "", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, ok := ParseOwnerRepo(tc.url)
			if ok != tc.wantOK {
				t.Fatalf("ParseOwnerRepo(%q) ok = %v, want %v", tc.url, ok, tc.wantOK)
			}
			if owner != tc.wantOwner {
				t.Errorf("ParseOwnerRepo(%q) owner = %q, want %q", tc.url, owner, tc.wantOwner)
			}
			if repo != tc.wantRepo {
				t.Errorf("ParseOwnerRepo(%q) repo = %q, want %q", tc.url, repo, tc.wantRepo)
			}
		})
	}
}

func TestGraphQLCheckToStatusCheck(t *testing.T) {
	tests := []struct {
		name string
		node ghGraphQLCheckNode
		want ghStatusCheck
	}{
		{
			"CheckRun success",
			ghGraphQLCheckNode{TypeName: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"},
			ghStatusCheck{Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
		{
			"CheckRun in progress",
			ghGraphQLCheckNode{TypeName: "CheckRun", Status: "IN_PROGRESS", Conclusion: ""},
			ghStatusCheck{Status: "IN_PROGRESS", Conclusion: ""},
		},
		{
			"StatusContext success",
			ghGraphQLCheckNode{TypeName: "StatusContext", State: "SUCCESS"},
			ghStatusCheck{Status: "COMPLETED", Conclusion: "SUCCESS"},
		},
		{
			"StatusContext pending",
			ghGraphQLCheckNode{TypeName: "StatusContext", State: "PENDING"},
			ghStatusCheck{Status: "IN_PROGRESS", Conclusion: ""},
		},
		{
			"StatusContext failure",
			ghGraphQLCheckNode{TypeName: "StatusContext", State: "FAILURE"},
			ghStatusCheck{Status: "COMPLETED", Conclusion: "FAILURE"},
		},
		{
			"StatusContext error",
			ghGraphQLCheckNode{TypeName: "StatusContext", State: "ERROR"},
			ghStatusCheck{Status: "COMPLETED", Conclusion: "FAILURE"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := graphQLCheckToStatusCheck(tc.node)
			if got != tc.want {
				t.Errorf("graphQLCheckToStatusCheck() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestGraphQLPRToForge(t *testing.T) {
	pr := ghGraphQLPR{
		Number:           42,
		Title:            "Test PR",
		State:            "OPEN",
		URL:              "https://github.com/owner/repo/pull/42",
		IsDraft:          false,
		MergeStateStatus: "CLEAN",
		ReviewDecision:   "APPROVED",
	}
	pr.Commits.Nodes = []struct {
		Commit struct {
			StatusCheckRollup *struct {
				Contexts struct {
					Nodes []ghGraphQLCheckNode `json:"nodes"`
				} `json:"contexts"`
			} `json:"statusCheckRollup"`
		} `json:"commit"`
	}{
		{Commit: struct {
			StatusCheckRollup *struct {
				Contexts struct {
					Nodes []ghGraphQLCheckNode `json:"nodes"`
				} `json:"contexts"`
			} `json:"statusCheckRollup"`
		}{
			StatusCheckRollup: &struct {
				Contexts struct {
					Nodes []ghGraphQLCheckNode `json:"nodes"`
				} `json:"contexts"`
			}{
				Contexts: struct {
					Nodes []ghGraphQLCheckNode `json:"nodes"`
				}{
					Nodes: []ghGraphQLCheckNode{
						{TypeName: "CheckRun", Status: "COMPLETED", Conclusion: "SUCCESS"},
					},
				},
			},
		}},
	}

	got := graphQLPRToForge(pr)
	if got.Number != 42 {
		t.Errorf("Number = %d, want 42", got.Number)
	}
	if got.State != "open" {
		t.Errorf("State = %q, want %q", got.State, "open")
	}
	if got.Checks != CheckStatusPass {
		t.Errorf("Checks = %q, want %q", got.Checks, CheckStatusPass)
	}
	if !got.ReviewApproved {
		t.Error("ReviewApproved = false, want true")
	}
	if !got.Mergeable {
		t.Error("Mergeable = false, want true")
	}
}

func TestComputeCheckStatus(t *testing.T) {
	tests := []struct {
		name     string
		checks   []ghStatusCheck
		expected CheckStatus
	}{
		{"no checks", nil, CheckStatusPass},
		{"empty checks", []ghStatusCheck{}, CheckStatusPass},
		{
			"all success",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
			},
			CheckStatusPass,
		},
		{
			"neutral and skipped count as pass",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "NEUTRAL"},
				{Status: "COMPLETED", Conclusion: "SKIPPED"},
			},
			CheckStatusPass,
		},
		{
			"one failure",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "FAILURE"},
			},
			CheckStatusFail,
		},
		{
			"still in progress",
			[]ghStatusCheck{
				{Status: "IN_PROGRESS", Conclusion: ""},
			},
			CheckStatusPending,
		},
		{
			"mix of completed and in progress",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "IN_PROGRESS", Conclusion: ""},
			},
			CheckStatusPending,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeCheckStatus(tc.checks)
			if got != tc.expected {
				t.Errorf("computeCheckStatus() = %q, want %q", got, tc.expected)
			}
		})
	}
}
