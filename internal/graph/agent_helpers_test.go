package graph

import (
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/forge"
)

func TestPrPrompt_NoPR(t *testing.T) {
	ctx := actionContext{ForgeCLI: "gh", PullRequest: nil}
	prompt := prPrompt(ctx)

	if !strings.Contains(prompt, "Create a pull request") {
		t.Error("expected 'Create a pull request' in prompt for no-PR state")
	}
	if strings.Contains(prompt, "Merge") {
		t.Error("prompt should not mention merging when no PR exists")
	}
}

func TestPrPrompt_HasLocalChanges(t *testing.T) {
	ctx := actionContext{
		ForgeCLI:   "gh",
		PullRequest: &forge.PullRequest{Checks: forge.CheckStatusPass},
		HasChanges: true,
	}
	prompt := prPrompt(ctx)

	if !strings.Contains(prompt, "Update the existing pull request") {
		t.Error("expected 'Update the existing pull request' in prompt")
	}
	if !strings.Contains(prompt, "Do NOT merge the PR") {
		t.Error("expected 'Do NOT merge' guardrail in update prompt")
	}
}

func TestPrPrompt_HasUnpushedCommits(t *testing.T) {
	ctx := actionContext{
		ForgeCLI:           "gh",
		PullRequest:        &forge.PullRequest{Checks: forge.CheckStatusPass},
		HasUnpushedCommits: true,
	}
	prompt := prPrompt(ctx)

	if !strings.Contains(prompt, "Update the existing pull request") {
		t.Error("expected 'Update the existing pull request' in prompt")
	}
	if !strings.Contains(prompt, "Do NOT merge the PR") {
		t.Error("expected 'Do NOT merge' guardrail in update prompt")
	}
}

func TestPrPrompt_ChecksFailing(t *testing.T) {
	ctx := actionContext{
		ForgeCLI:    "gh",
		PullRequest: &forge.PullRequest{Checks: forge.CheckStatusFail},
	}
	prompt := prPrompt(ctx)

	if !strings.Contains(prompt, "CI checks on this PR are failing") {
		t.Error("expected failing checks prompt")
	}
	if !strings.Contains(prompt, "Do NOT merge the PR") {
		t.Error("expected 'Do NOT merge' guardrail in fix-tests prompt")
	}
}

func TestPrPrompt_Mergeable(t *testing.T) {
	ctx := actionContext{
		ForgeCLI:    "gh",
		PullRequest: &forge.PullRequest{Checks: forge.CheckStatusPass, Mergeable: true},
	}
	prompt := prPrompt(ctx)

	if !strings.Contains(prompt, "Merge this pull request") {
		t.Error("expected 'Merge this pull request' in prompt")
	}
}

func TestPrPrompt_UsesForgeCLI(t *testing.T) {
	for _, cli := range []string{"gh", "glab"} {
		ctx := actionContext{ForgeCLI: cli, PullRequest: nil}
		prompt := prPrompt(ctx)
		if !strings.Contains(prompt, cli+" pr create") {
			t.Errorf("expected prompt to use %s CLI", cli)
		}
	}
}
