package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// buildBinary builds the kudig binary for testing and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "kudig")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join(".", "..", "..", "cmd", "kudig")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, out)
	}

	return binaryPath
}

func TestCLIVersion(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v\n%s", err, out)
	}

	output := string(out)
	if len(output) == 0 {
		t.Error("--version produced no output")
	}
}

func TestCLIHelp(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\n%s", err, out)
	}

	output := string(out)
	// Verify key commands are listed
	expectedCommands := []string{
		"offline",
		"online",
		"tui",
		"rules",
		"history",
		"completion",
	}
	for _, cmd := range expectedCommands {
		if !contains(output, cmd) {
			t.Errorf("--help output missing command: %s", cmd)
		}
	}
}

func TestCLIListAnalyzers(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "list-analyzers").CombinedOutput()
	if err != nil {
		t.Fatalf("list-analyzers failed: %v\n%s", err, out)
	}

	output := string(out)
	if !contains(output, "Available Analyzers") {
		t.Error("list-analyzers should show 'Available Analyzers'")
	}
}

func TestCLICompletion(t *testing.T) {
	binary := buildBinary(t)

	shells := []string{"bash", "zsh", "fish"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			out, err := exec.Command(binary, "completion", shell).CombinedOutput()
			if err != nil {
				t.Fatalf("completion %s failed: %v\n%s", shell, err, out)
			}
			if len(out) == 0 {
				t.Errorf("completion %s produced no output", shell)
			}
		})
	}
}

func TestCLICompletionInvalidShell(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "completion", "invalid").CombinedOutput()
	if err == nil {
		t.Error("completion with invalid shell should return error")
	}
	_ = out
}

func TestCLIOfflineMissingPath(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "offline").CombinedOutput()
	if err == nil {
		t.Error("offline without path should return error")
	}
	_ = out
}

func TestCLIOfflineInvalidPath(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "offline", "/nonexistent/path/12345").CombinedOutput()
	if err == nil {
		t.Error("offline with nonexistent path should return error")
	}
	_ = out
}

func TestCLIRulesList(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "rules", "--list").CombinedOutput()
	if err != nil {
		t.Fatalf("rules --list failed: %v\n%s", err, out)
	}

	output := string(out)
	if !contains(output, "Available Rules") {
		t.Error("rules --list should show 'Available Rules'")
	}
}

func TestCLIHistoryEmpty(t *testing.T) {
	binary := buildBinary(t)

	// Set HOME to temp dir to avoid polluting real history
	tmpHome := t.TempDir()
	cmd := exec.Command(binary, "history", "list")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("history list failed: %v\n%s", err, out)
	}

	output := string(out)
	if !contains(output, "No history entries found") {
		t.Error("history list on empty should show 'No history entries found'")
	}
}

func TestCLITracenotImplemented(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "trace").CombinedOutput()
	if err == nil {
		t.Error("trace should return error (not implemented)")
	}
	_ = out
}

func TestCLIMulticlusterNotImplemented(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "multicluster").CombinedOutput()
	if err == nil {
		t.Error("multicluster should return error (not implemented)")
	}
	_ = out
}

func TestCLIGrafana(t *testing.T) {
	binary := buildBinary(t)

	out, err := exec.Command(binary, "grafana").CombinedOutput()
	if err != nil {
		t.Fatalf("grafana failed: %v\n%s", err, out)
	}

	output := string(out)
	if !contains(output, "dashboard") && !contains(output, "panels") {
		t.Error("grafana should output dashboard JSON")
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
