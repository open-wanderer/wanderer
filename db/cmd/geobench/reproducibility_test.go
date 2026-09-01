package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizedInvocationUsesStableExecutableName(t *testing.T) {
	got := normalizedInvocation([]string{"/tmp/go-build123/geobench", "--profile", "standard"})
	want := []string{"geobench", "--profile", "standard"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invocation = %v, want %v", got, want)
	}
}

func TestDigestFile(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, directory, "geobench", "binary")
	digest, err := digestFile(filepath.Join(directory, "geobench"))
	if err != nil {
		t.Fatalf("digest file: %v", err)
	}
	if digest != sha256Digest([]byte("binary")) {
		t.Fatalf("digest = %q", digest)
	}
}

func TestDigestHarnessDirectoryTracksBuildInputsOnly(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, directory, "go.mod", "module example.test/geobench\n")
	writeTestFile(t, directory, "go.sum", "example.test/dependency v1.0.0 h1:digest\n")
	writeTestFile(t, directory, "z.go", "package main\nconst z = 1\n")
	writeTestFile(t, directory, "nested/a.go", "package nested\nconst a = 1\n")
	writeTestFile(t, directory, "z_test.go", "package main\n")
	writeTestFile(t, directory, "README.md", "first\n")

	first, err := digestHarnessDirectory(directory)
	if err != nil {
		t.Fatalf("digest first harness: %v", err)
	}
	if !strings.HasPrefix(first, "sha256:") || len(first) != len("sha256:")+64 {
		t.Fatalf("harness digest = %q, want prefixed SHA-256", first)
	}

	writeTestFile(t, directory, "README.md", "documentation-only change\n")
	writeTestFile(t, directory, "z_test.go", "package main\n// test-only change\n")
	second, err := digestHarnessDirectory(directory)
	if err != nil {
		t.Fatalf("digest harness after non-build changes: %v", err)
	}
	if second != first {
		t.Fatalf("non-build files changed digest: %s != %s", second, first)
	}

	writeTestFile(t, directory, "nested/a.go", "package nested\nconst a = 2\n")
	third, err := digestHarnessDirectory(directory)
	if err != nil {
		t.Fatalf("digest changed harness: %v", err)
	}
	if third == first {
		t.Fatal("source change did not change harness digest")
	}
}

func TestDigestDatasetIsSemanticAndDeterministic(t *testing.T) {
	dataset := Dataset{
		Seed: 42,
		Trails: []Trail{{
			ID:       "trail",
			Scenario: "test",
			Parts:    [][]Coordinate{{{Lat: 46.8, Lon: 8.2}, {Lat: 46.9, Lon: 8.3}}},
		}},
		QueryPoints: []QueryPoint{{ID: "query", Scenario: "test", Point: Coordinate{Lat: 46.85, Lon: 8.25}}},
	}
	first, err := digestDataset(dataset)
	if err != nil {
		t.Fatalf("digest first dataset: %v", err)
	}
	second, err := digestDataset(dataset)
	if err != nil {
		t.Fatalf("digest second dataset: %v", err)
	}
	if first != second {
		t.Fatalf("same dataset digests differ: %s != %s", first, second)
	}

	dataset.Trails[0].Parts[0][1].Lat += 0.0001
	changed, err := digestDataset(dataset)
	if err != nil {
		t.Fatalf("digest changed dataset: %v", err)
	}
	if changed == first {
		t.Fatal("geometry change did not change dataset digest")
	}
}

func TestParseDockerImageInspectReturnsStableRepoDigests(t *testing.T) {
	imageID, digests, err := parseDockerImageInspect([]byte(`[{"Id":"sha256:local","RepoDigests":["repo@sha256:b","repo@sha256:a","repo@sha256:a"]}]`))
	if err != nil {
		t.Fatalf("parse Docker inspect: %v", err)
	}
	if imageID != "sha256:local" {
		t.Fatalf("image ID = %q", imageID)
	}
	want := []string{"repo@sha256:a", "repo@sha256:b"}
	if !reflect.DeepEqual(digests, want) {
		t.Fatalf("repo digests = %v, want %v", digests, want)
	}
	if _, _, err := parseDockerImageInspect([]byte(`[]`)); err == nil {
		t.Fatal("empty Docker inspect result succeeded")
	}
}

func TestParseHostMetadata(t *testing.T) {
	osRelease := parseOSRelease([]byte("ID=ubuntu\nVERSION_ID='24.04'\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"\n"))
	if osRelease != "Ubuntu 24.04 LTS" {
		t.Fatalf("OS release = %q", osRelease)
	}

	cpuInfo := []byte("processor: 0\nphysical id: 0\ncore id: 0\nmodel name: Example CPU\n\n" +
		"processor: 1\nphysical id: 0\ncore id: 0\nmodel name: Example CPU\n\n" +
		"processor: 2\nphysical id: 0\ncore id: 1\nmodel name: Example CPU\n")
	model, physicalCores := parseCPUInfo(cpuInfo)
	if model != "Example CPU" || physicalCores != 2 {
		t.Fatalf("CPU metadata = (%q, %d), want (Example CPU, 2)", model, physicalCores)
	}

	memoryBytes, ok := parseMemTotalBytes([]byte("MemFree: 10 kB\nMemTotal: 32768 kB\n"))
	if !ok || memoryBytes != 32*1024*1024 {
		t.Fatalf("memory total = (%d, %v), want (%d, true)", memoryBytes, ok, 32*1024*1024)
	}
}

func TestDefaultMeilisearchMatrixIncludesLatestStable(t *testing.T) {
	want := []string{
		"getmeili/meilisearch:v1.36.0",
		"getmeili/meilisearch:v1.44.0",
		"getmeili/meilisearch:v1.53.1",
	}
	if got := splitList(defaultMeilisearchImages); !reflect.DeepEqual(got, want) {
		t.Fatalf("default Meilisearch images = %v, want %v", got, want)
	}
}

func TestMarkdownReportIncludesReproducibilityMetadata(t *testing.T) {
	report := BenchmarkReport{
		Reproducibility: ReproducibilityReport{
			Invocation:       []string{"geobench", "--profile", "standard"},
			ExecutableSHA256: "sha256:executable",
			HarnessSHA256:    "sha256:harness",
			DatasetSHA256:    "sha256:dataset",
			DockerImages: []DockerImageReport{{
				Reference:   "example/image:v1",
				ImageID:     "sha256:image",
				RepoDigests: []string{"example/image@sha256:repo"},
			}},
		},
	}
	markdown := markdownReport(report)
	for _, wanted := range []string{"## Reproducibility", `["geobench","--profile","standard"]`, "sha256:executable", "sha256:harness", "sha256:dataset", "example/image@sha256:repo"} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("Markdown report does not contain %q:\n%s", wanted, markdown)
		}
	}
}

func writeTestFile(t *testing.T, directory, relative, contents string) {
	t.Helper()
	path := filepath.Join(directory, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o640); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}
