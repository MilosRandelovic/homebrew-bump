package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/MilosRandelovic/bump-core/v2/shared"
	"github.com/MilosRandelovic/homebrew-bump/internal/output"
)

func TestDependencyOptionsMapsEachField(t *testing.T) {
	tests := []struct {
		name  string
		input commandOptions
		want  shared.Options
	}{
		{"semver", commandOptions{Semver: true}, shared.Options{Semver: true}},
		{"no cache", commandOptions{NoCache: true}, shared.Options{NoCache: true}},
		{"peers", commandOptions{IncludePeerDependencies: true}, shared.Options{IncludePeerDependencies: true}},
		{"monorepo", commandOptions{Monorepo: true}, shared.Options{Monorepo: true}},
		{"minimum age", commandOptions{MinimumAge: true}, shared.Options{EnforceMinimumReleaseAge: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.input.dependencyOptions(); got != test.want {
				t.Fatalf("dependencyOptions() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestOutputConfigMapsEachField(t *testing.T) {
	tests := []struct {
		name  string
		input commandOptions
		want  output.Config
	}{
		{"verbose", commandOptions{Verbose: true}, output.Config{Verbose: true}},
		{"semver", commandOptions{Semver: true}, output.Config{Semver: true}},
		{"minimum age", commandOptions{MinimumAge: true}, output.Config{MinimumAge: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.input.outputConfig(); got != test.want {
				t.Fatalf("outputConfig() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestSignalCancellation(t *testing.T) {
	if phase := os.Getenv("BUMP_TEST_SIGNAL_PHASE"); phase != "" {
		if phase == "check" {
			checkOutdated = func(ctx context.Context, dependencies []shared.Dependency, registryType shared.RegistryType, options shared.Options, workingDirectory string, progressCallback shared.ProgressFunc, log shared.LogFunc) (*shared.CheckResult, error) {
				fmt.Fprintln(os.Stderr, "READY")
				<-ctx.Done()
				return nil, ctx.Err()
			}
		} else {
			checkOutdated = func(ctx context.Context, dependencies []shared.Dependency, registryType shared.RegistryType, options shared.Options, workingDirectory string, progressCallback shared.ProgressFunc, log shared.LogFunc) (*shared.CheckResult, error) {
				return &shared.CheckResult{Outdated: []shared.OutdatedDependency{{BaseDependency: shared.BaseDependency{Name: "example", FilePath: "package.json", Type: shared.Dependencies, OriginalVersion: "1.0.0"}, CurrentVersion: "1.0.0", LatestVersion: "1.0.1"}}}, nil
			}
			updateDependencies = func(ctx context.Context, filePath string, outdated []shared.OutdatedDependency, registryType shared.RegistryType, options shared.Options, workingDirectory string, log shared.LogFunc) error {
				fmt.Fprintln(os.Stderr, "READY")
				<-ctx.Done()
				return ctx.Err()
			}
		}
		os.Args = []string{"bump", "--update"}
		main()
		return
	}

	for _, phase := range []string{"check", "update"} {
		t.Run(phase, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(directory+"/package.json", []byte(`{"dependencies":{"example":"1.0.0"}}`), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSignalCancellation$")
			command.Dir = directory
			command.Env = append(os.Environ(), "BUMP_TEST_SIGNAL_PHASE="+phase)
			stderr, err := command.StderrPipe()
			if err != nil {
				t.Fatal(err)
			}
			ready := make(chan struct{}, 1)
			outputDone := make(chan string, 1)
			go func() {
				var diagnostics bytes.Buffer
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					line := scanner.Text()
					fmt.Fprintln(&diagnostics, line)
					if line == "READY" {
						ready <- struct{}{}
					}
				}
				outputDone <- diagnostics.String()
			}()
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			select {
			case <-ready:
			case diagnostics := <-outputDone:
				_ = command.Wait()
				t.Fatalf("phase %s ended before the stalled call: %s", phase, diagnostics)
			case <-ctx.Done():
				_ = command.Wait()
				t.Fatalf("phase %s did not start", phase)
			}
			if err := command.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			var diagnostics string
			select {
			case diagnostics = <-outputDone:
			case <-ctx.Done():
				_ = command.Wait()
				t.Fatalf("phase %s did not cancel promptly", phase)
			}
			if err := command.Wait(); err == nil {
				t.Fatal("expected cancellation failure")
			}
			if ctx.Err() != nil || !strings.Contains(diagnostics, "context canceled") {
				t.Fatalf("phase %s did not cancel promptly: %s", phase, diagnostics)
			}
		})
	}
}
