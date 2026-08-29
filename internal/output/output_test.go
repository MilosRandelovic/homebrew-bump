package output

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/MilosRandelovic/bump-core/v2/shared"
)

func TestPrintUpdatePromptPreservesMinimumAge(t *testing.T) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStdout := os.Stdout
	os.Stdout = writePipe

	PrintUpdatePrompt(true, shared.Options{Semver: true, EnforceMinimumReleaseAge: true})
	os.Stdout = originalStdout
	if err := writePipe.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(output), "bump --update --semver --minimum-age") {
		t.Fatalf("prompt did not preserve options: %q", output)
	}
}

func TestPrintProgressBarUsesPerFileProgress(t *testing.T) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	os.Stderr = writePipe

	PrintProgressBar(shared.Progress{FilePath: "package.json", FileCurrent: 1, FileTotal: 2, Current: 7, Total: 20})
	os.Stderr = originalStderr
	if err := writePipe.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(output), "package.json") || !strings.Contains(string(output), "1/2 50%") {
		t.Fatalf("progress did not use per-file counts: %q", output)
	}
	if strings.Contains(string(output), "7/20") {
		t.Fatalf("progress used global counts: %q", output)
	}
}
