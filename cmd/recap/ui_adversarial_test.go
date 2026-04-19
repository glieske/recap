package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type adversarialRunResult struct {
	stdout   string
	stderr   string
	exitCode int
	timedOut bool
}

func runMainAdversarial(t *testing.T, args []string, env map[string]string) adversarialRunResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcessUIAdversarialMain")
	cmd.Env = append(os.Environ(),
		"RECAP_HELPER_PROCESS=1",
		"RECAP_HELPER_MODE=ui_adversarial_main",
		"RECAP_TEST_ARGS="+strings.Join(args, "\n"),
	)

	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := adversarialRunResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.timedOut = true
		return result
	}

	if err == nil {
		result.exitCode = 0
		return result
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T (err=%v)", err, err)
	}

	result.exitCode = exitErr.ExitCode()
	return result
}

func TestUIAdversarial_DoubleUISubcommand_InvokesUIStub(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	tmp := t.TempDir()
	result := runMainAdversarial(t, []string{"ui", "ui"}, map[string]string{
		"XDG_CONFIG_HOME": filepath.Join(tmp, "xdg"),
		"HOME":            filepath.Join(tmp, "home"),
	})

	if result.timedOut {
		t.Fatalf("expected helper process to complete without timeout")
	}
	if result.exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.exitCode)
	}
	if !strings.Contains(result.stderr, "GUI not available") {
		t.Fatalf("expected stderr to contain UI stub message, got %q", result.stderr)
	}
}

func TestUIAdversarial_UIWithNonexistentConfig_ExitsOnConfigError(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	result := runMainAdversarial(t, []string{"ui", "--config", "/nonexistent"}, nil)

	if result.timedOut {
		t.Fatalf("expected helper process to complete without timeout")
	}
	if result.exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.exitCode)
	}
	if !strings.Contains(result.stderr, "Error loading config:") {
		t.Fatalf("expected config load error in stderr, got %q", result.stderr)
	}
	if strings.Contains(result.stderr, "GUI not available") {
		t.Fatalf("did not expect UI stub message when config load fails, got %q", result.stderr)
	}
}

func TestUIAdversarial_NoArgs_DoesNotInvokeUIStub(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "xdg-blocker")
	if err := os.WriteFile(blocked, []byte("block"), 0o600); err != nil {
		t.Fatalf("failed creating blocker file: %v", err)
	}

	result := runMainAdversarial(t, nil, map[string]string{
		"XDG_CONFIG_HOME": blocked,
	})

	if result.timedOut {
		t.Fatalf("expected helper process to complete without timeout")
	}
	if result.exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.exitCode)
	}
	if strings.Contains(result.stderr, "GUI not available") {
		t.Fatalf("did not expect UI stub message for no-args path, got %q", result.stderr)
	}
}

func TestUIAdversarial_NewThenUI_NewShouldTakePriority(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	tmp := t.TempDir()
	result := runMainAdversarial(t, []string{"new", "ui"}, map[string]string{
		"XDG_CONFIG_HOME": filepath.Join(tmp, "xdg"),
		"HOME":            filepath.Join(tmp, "home"),
	})

	if result.timedOut {
		t.Fatalf("expected helper process to complete without timeout")
	}
	if result.exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.exitCode)
	}
	if !strings.Contains(result.stderr, "GUI not available") {
		t.Fatalf("expected 'ui' to take priority when both 'new' and 'ui' are present, got %q", result.stderr)
	}
}

func TestUIAdversarial_UIWithManyExtraArgs_StillRoutesToUI(t *testing.T) {
	t.Setenv("RECAP_HELPER_PROCESS", "0")

	extra := make([]string, 0, 12)
	extra = append(extra,
		"--config", "ignored-after-positional",
		"../path-traversal",
		"<script>alert(1)</script>",
		"${injection}",
		"key=value-with-dashes-and_symbols",
		"emoji-😀",
		"zero-width-\u200bspace",
		"long-safe-arg-"+strings.Repeat("x", 1024),
		strings.Repeat("A", 12*1024),
	)
	for i := 0; i < 200; i++ {
		extra = append(extra, "arg-boundary")
	}

	args := append([]string{"ui"}, extra...)
	tmp := t.TempDir()
	result := runMainAdversarial(t, args, map[string]string{
		"XDG_CONFIG_HOME": filepath.Join(tmp, "xdg"),
		"HOME":            filepath.Join(tmp, "home"),
	})

	if result.timedOut {
		t.Fatalf("expected helper process to complete without timeout")
	}
	if result.exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.exitCode)
	}
	if !strings.Contains(result.stderr, "GUI not available") {
		t.Fatalf("expected UI stub message with extra args, got %q", result.stderr)
	}
}

func TestHelperProcessUIAdversarialMain(t *testing.T) {
	if os.Getenv("RECAP_HELPER_PROCESS") != "1" || os.Getenv("RECAP_HELPER_MODE") != "ui_adversarial_main" {
		return
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		_ = os.Setenv("XDG_CONFIG_HOME", xdg)
	}
	if home := os.Getenv("HOME"); home != "" {
		_ = os.Setenv("HOME", home)
	}

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	argPayload := os.Getenv("RECAP_TEST_ARGS")
	var args []string
	if argPayload != "" {
		args = strings.Split(argPayload, "\n")
	}

	os.Args = append([]string{"recap"}, args...)
	main()
	os.Exit(0)
}
