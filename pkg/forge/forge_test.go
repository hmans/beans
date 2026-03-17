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
			PullRequest{ChecksPass: true, ReviewApproved: true, Mergeable: true},
			true,
		},
		{
			"draft PR",
			PullRequest{IsDraft: true, ChecksPass: true, ReviewApproved: true, Mergeable: true},
			false,
		},
		{
			"checks failing",
			PullRequest{ChecksPass: false, ReviewApproved: true, Mergeable: true},
			false,
		},
		{
			"not mergeable",
			PullRequest{ChecksPass: true, ReviewApproved: true, Mergeable: false},
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

func TestChecksPass(t *testing.T) {
	tests := []struct {
		name     string
		checks   []ghStatusCheck
		expected bool
	}{
		{"no checks", nil, true},
		{"empty checks", []ghStatusCheck{}, true},
		{
			"all success",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
			},
			true,
		},
		{
			"neutral and skipped count as pass",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "NEUTRAL"},
				{Status: "COMPLETED", Conclusion: "SKIPPED"},
			},
			true,
		},
		{
			"one failure",
			[]ghStatusCheck{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "FAILURE"},
			},
			false,
		},
		{
			"still in progress",
			[]ghStatusCheck{
				{Status: "IN_PROGRESS", Conclusion: ""},
			},
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := checksPass(tc.checks)
			if got != tc.expected {
				t.Errorf("checksPass() = %v, want %v", got, tc.expected)
			}
		})
	}
}
