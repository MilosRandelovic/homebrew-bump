package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
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

func TestCLIOrchestratesParsedOptions(t *testing.T) {
	if phase := os.Getenv("BUMP_TEST_OPTIONS_PHASE"); phase != "" {
		want := shared.Options{Semver: true, NoCache: true, IncludePeerDependencies: true, Monorepo: true, EnforceMinimumReleaseAge: true}
		checkOutdated = func(ctx context.Context, dependencies []shared.Dependency, registryType shared.RegistryType, options shared.Options, workingDirectory string, progressCallback shared.ProgressFunc, log shared.LogFunc) (*shared.CheckResult, error) {
			foundPeer := false
			for _, dependency := range dependencies {
				if dependency.Name == "peer-example" && dependency.Type == shared.PeerDependencies {
					foundPeer = true
				}
			}
			if !reflect.DeepEqual(options, want) || len(dependencies) != 2 || !foundPeer || progressCallback != nil || log == nil {
				return nil, fmt.Errorf("check received options=%+v, dependencies=%d, progress=%v, verbose=%v", options, len(dependencies), progressCallback != nil, log != nil)
			}
			return &shared.CheckResult{
				Outdated:      []shared.OutdatedDependency{{BaseDependency: shared.BaseDependency{Name: "example", FilePath: "package.json", Type: shared.Dependencies, OriginalVersion: "1.0.0"}, CurrentVersion: "1.0.0", LatestVersion: "1.0.1"}},
				SemverSkipped: []shared.SemverSkipped{{OutdatedDependency: shared.OutdatedDependency{BaseDependency: shared.BaseDependency{Name: "skipped", FilePath: "package.json", Type: shared.Dependencies, OriginalVersion: "^1.0.0"}, LatestVersion: "2.0.0"}, Reason: shared.IncompatibleWithConstraint}},
			}, nil
		}
		updateDependencies = func(ctx context.Context, filePath string, outdated []shared.OutdatedDependency, registryType shared.RegistryType, options shared.Options, workingDirectory string, log shared.LogFunc) error {
			if !reflect.DeepEqual(options, want) || len(outdated) != 1 || log == nil {
				return fmt.Errorf("update received options=%+v, outdated=%d, verbose=%v", options, len(outdated), log != nil)
			}
			fmt.Println("UPDATE_CALLED")
			return nil
		}
		os.Args = []string{"bump", "-vsaCPm"}
		if phase == "update" {
			os.Args = []string{"bump", "-uvsaCPm"}
		}
		main()
		return
	}

	directory := t.TempDir()
	if err := os.WriteFile(directory+"/package.json", []byte("{\n  \"dependencies\": {\n    \"example\": \"1.0.0\"\n  },\n  \"peerDependencies\": {\n    \"peer-example\": \"1.0.0\"\n  }\n}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"check", "update"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIOrchestratesParsedOptions$")
			command.Dir = directory
			command.Env = append(os.Environ(), "BUMP_TEST_OPTIONS_PHASE="+phase)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("CLI orchestration failed: %v\n%s", err, output)
			}
			for _, expected := range []string{"Found 2 dependencies", "Packages skipped due to semver constraints:", "incompatible with constraint"} {
				if !strings.Contains(string(output), expected) {
					t.Errorf("CLI output missing %q:\n%s", expected, output)
				}
			}
			if phase == "check" && !strings.Contains(string(output), "bump --update --semver --minimum-age") {
				t.Errorf("CLI output did not preserve output options:\n%s", output)
			}
			if phase == "update" && !strings.Contains(string(output), "UPDATE_CALLED") {
				t.Errorf("CLI did not call update with parsed options:\n%s", output)
			}
		})
	}
}

func TestCLIRejectsPubPeerFlagBeforeRegistryCheck(t *testing.T) {
	if os.Getenv("BUMP_TEST_PUB_VALIDATION") != "" {
		checkOutdated = func(ctx context.Context, dependencies []shared.Dependency, registryType shared.RegistryType, options shared.Options, workingDirectory string, progressCallback shared.ProgressFunc, log shared.LogFunc) (*shared.CheckResult, error) {
			fmt.Fprintln(os.Stderr, "REGISTRY_CHECK_CALLED")
			return nil, fmt.Errorf("registry check should not run")
		}
		os.Args = []string{"bump", "--include-peers"}
		main()
		return
	}

	directory := t.TempDir()
	if err := os.WriteFile(directory+"/pubspec.yaml", []byte("name: fixture\ndependencies:\n  example: ^1.0.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIRejectsPubPeerFlagBeforeRegistryCheck$")
	command.Dir = directory
	command.Env = append(os.Environ(), "BUMP_TEST_PUB_VALIDATION=1")
	output, err := command.CombinedOutput()
	if err == nil || ctx.Err() != nil {
		t.Fatalf("CLI should reject the Pub peer flag promptly: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "peer dependencies are not supported by pub") || strings.Contains(string(output), "REGISTRY_CHECK_CALLED") {
		t.Fatalf("CLI did not reject the parsed flag before registry work:\n%s", output)
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
