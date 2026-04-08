package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPrimeOutputsPromptWithoutBeansProject(t *testing.T) {
	oldBeansPath, oldConfigPath := beansPath, configPath
	beansPath, configPath = "", ""
	t.Cleanup(func() {
		beansPath, configPath = oldBeansPath, oldConfigPath
	})

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", tmpDir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldCwd)
	})

	cmd := *primeCmd
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.RunE(&cmd, nil); err != nil {
		t.Fatalf("prime RunE() error = %v", err)
	}

	if stdout.Len() == 0 {
		t.Fatal("expected prime to write prompt output, got empty output")
	}

	if !strings.Contains(stdout.String(), "# Beans Usage Guide for Agents") {
		t.Fatalf("expected prompt heading in output, got %q", stdout.String())
	}
}
