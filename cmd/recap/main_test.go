package main

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func findProjectRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("failed to locate project root containing go.mod from %q", wd)
		}

		current = parent
	}
}

func TestGoModModulePathIsRecap(t *testing.T) {
	projectRoot := findProjectRoot(t)
	content, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	if !strings.Contains(string(content), "module github.com/glieske/recap") {
		t.Fatalf("expected go.mod to contain module github.com/glieske/recap")
	}
}

func TestCmdRecapDirectoryExists(t *testing.T) {
	projectRoot := findProjectRoot(t)
	info, err := os.Stat(filepath.Join(projectRoot, "cmd", "recap"))
	if err != nil {
		t.Fatalf("expected cmd/recap directory to exist: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("expected cmd/recap to be a directory")
	}
	if info.Name() != "recap" {
		t.Fatalf("expected directory name %q, got %q", "recap", info.Name())
	}
}

func TestNoGoFileContainsOldImportPath(t *testing.T) {
	projectRoot := findProjectRoot(t)
	const oldImportPath = "github.com/greg/" + "notes"

	var offenders []string
	err := filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(b), oldImportPath) {
			rel, err := filepath.Rel(projectRoot, path)
			if err != nil {
				rel = path
			}
			offenders = append(offenders, rel)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed walking Go files: %v", err)
	}

	if len(offenders) != 0 {
		t.Fatalf("found old import path %q in files: %v", oldImportPath, offenders)
	}
}

func TestMainVersionStringUsesRecapNotNotes(t *testing.T) {
	projectRoot := findProjectRoot(t)
	content, err := os.ReadFile(filepath.Join(projectRoot, "cmd", "recap", "main.go"))
	if err != nil {
		t.Fatalf("failed to read cmd/recap/main.go: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "recap version") {
		t.Fatalf("expected main.go to contain %q", "recap version")
	}
	if strings.Contains(text, "notes version") {
		t.Fatalf("did not expect main.go to contain %q", "notes version")
	}
}

func TestInitKeepsDevVersionForLocalDevelBuild(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatalf("expected debug.ReadBuildInfo to succeed in test binary")
	}

	if info.Main.Version != "(devel)" {
		t.Skipf("test expects local test build with Main.Version=(devel), got %q", info.Main.Version)
	}

	if version != "dev" {
		t.Fatalf("expected package version to remain %q for local (devel) build, got %q", "dev", version)
	}
}

func TestInitVersionIsAlwaysNonEmpty(t *testing.T) {
	if version == "" {
		t.Fatalf("expected version to be non-empty")
	}
}

func TestRunUIStub_PrintsExpectedMessageAndExitsWithCode1(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessRunUIStub")
	cmd.Env = append(os.Environ(),
		"RECAP_HELPER_PROCESS=1",
		"RECAP_HELPER_MODE=run_ui_stub",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected helper process to exit with non-zero code")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}

	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}

	const wantStderr = "GUI not available — rebuild with: go build -tags gui\n"
	if stderr.String() != wantStderr {
		t.Fatalf("unexpected stderr\nwant: %q\n got: %q", wantStderr, stderr.String())
	}
}

func TestHelperProcessRunUIStub(t *testing.T) {
	if os.Getenv("RECAP_HELPER_PROCESS") != "1" || os.Getenv("RECAP_HELPER_MODE") != "run_ui_stub" {
		return
	}

	runUI(nil, nil, nil, "", "test")
	t.Fatalf("runUI should have exited with code 1")
}

func TestMainWithoutUISubcommand_DoesNotInvokeUIStub(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcessMainVersionPath")
	cmd.Env = append(os.Environ(),
		"RECAP_HELPER_PROCESS=1",
		"RECAP_HELPER_MODE=main_version",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected helper process to exit successfully, got error: %v", err)
	}

	if !strings.Contains(stdout.String(), "recap version ") {
		t.Fatalf("expected stdout to contain %q, got %q", "recap version ", stdout.String())
	}

	if strings.Contains(stdout.String(), "GUI not available") {
		t.Fatalf("did not expect UI stub message in stdout: %q", stdout.String())
	}

	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestHelperProcessMainVersionPath(t *testing.T) {
	if os.Getenv("RECAP_HELPER_PROCESS") != "1" || os.Getenv("RECAP_HELPER_MODE") != "main_version" {
		return
	}

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	os.Args = []string{"recap", "--version"}
	main()
	os.Exit(0)
}
