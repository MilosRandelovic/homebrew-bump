package output

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MilosRandelovic/bump-core/v2/shared"
)

func TestGetDisplayPathUsesExplicitWorkingDirectory(t *testing.T) {
	base := t.TempDir()
	filePath := filepath.Join(base, "packages", "app", "package.json")
	for _, test := range []struct {
		name, filePath, workingDirectory, want string
	}{
		{"relative path", filePath, base, filepath.Join("packages", "app", "package.json")},
		{"absent base", filePath, "", filePath},
		{"incompatible path forms", "package.json", base, "package.json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := getDisplayPath(test.filePath, Config{WorkingDirectory: test.workingDirectory}); got != test.want {
				t.Fatalf("getDisplayPath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPrintUpdatePromptPreservesMinimumAge(t *testing.T) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStdout := os.Stdout
	os.Stdout = writePipe

	PrintUpdatePrompt(true, Config{Semver: true, MinimumAge: true})
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

	PrintProgressBar(shared.Progress{FilePath: "package.json", FileCurrent: 1, FileTotal: 2, Current: 7, Total: 20}, Config{})
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
