package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/internal/version"
)

func TestRunVersion(t *testing.T) {
	restore := setVersionForTest("v0.1.0", "7abe2887e2d6", "2026-03-24T10:00:00Z")
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run([]string{"version"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run() exit code = %d, want 0", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	got := stdout.String()
	if !strings.Contains(got, "go-stencil version v0.1.0 (7abe288, 2026-03-24T10:00:00Z)") {
		t.Fatalf("stdout = %q, want version details", got)
	}
}

func TestRunWithoutArgsShowsUsage(t *testing.T) {
	restore := setVersionForTest("dev", "unknown", "unknown")
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("run() exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stdout.String(), "Version: dev") {
		t.Fatalf("stdout = %q, want banner with dev version", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: stencil <command> [arguments]") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func setVersionForTest(v, commit, date string) func() {
	previousVersion := version.Version
	previousCommit := version.Commit
	previousDate := version.Date

	version.Version = v
	version.Commit = commit
	version.Date = date

	return func() {
		version.Version = previousVersion
		version.Commit = previousCommit
		version.Date = previousDate
	}
}
