package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
)

const harnessDigestVersion = "wanderer-geobench-harness-v1"

func initialReproducibility(args []string) ReproducibilityReport {
	report := ReproducibilityReport{Invocation: normalizedInvocation(args)}
	executable, err := os.Executable()
	if err != nil {
		report.Warnings = append(report.Warnings, "executable digest unavailable: "+err.Error())
	} else if digest, digestErr := digestFile(executable); digestErr != nil {
		report.Warnings = append(report.Warnings, "executable digest unavailable: "+digestErr.Error())
	} else {
		report.ExecutableSHA256 = digest
	}

	directory, err := harnessSourceDirectory()
	if err != nil {
		report.Warnings = append(report.Warnings, "harness digest unavailable: "+err.Error())
		return report
	}
	digest, err := digestHarnessDirectory(directory)
	if err != nil {
		report.Warnings = append(report.Warnings, "harness digest unavailable: "+err.Error())
		return report
	}
	report.HarnessSHA256 = digest
	return report
}

func normalizedInvocation(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	invocation := append([]string(nil), args...)
	invocation[0] = filepath.Base(invocation[0])
	return invocation
}

func digestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func addDatasetDigest(report *BenchmarkReport, dataset Dataset) {
	digest, err := digestDataset(dataset)
	if err != nil {
		report.Reproducibility.Warnings = append(report.Reproducibility.Warnings, "dataset digest unavailable: "+err.Error())
		return
	}
	report.Reproducibility.DatasetSHA256 = digest
}

// addDockerImageMetadata runs after the benchmark cells. At that point Docker
// has pulled every successfully started image, so mutable tags can be paired
// with the immutable repository digests that were actually present locally.
func addDockerImageMetadata(report *BenchmarkReport) {
	if len(report.Config.MeilisearchImages) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	report.Reproducibility.DockerImages = inspectDockerImages(ctx, report.Config.MeilisearchImages)
}

func digestDataset(dataset Dataset) (string, error) {
	encoded, err := json.Marshal(dataset)
	if err != nil {
		return "", err
	}
	return sha256Digest(encoded), nil
}

func harnessSourceDirectory() (string, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if ok {
		directory := filepath.Dir(sourceFile)
		if isRegularFile(filepath.Join(directory, "go.mod")) {
			return directory, nil
		}
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidates := []string{
		workingDirectory,
		filepath.Join(workingDirectory, "db", "cmd", "geobench"),
	}
	for _, candidate := range candidates {
		if isRegularFile(filepath.Join(candidate, "go.mod")) {
			return candidate, nil
		}
	}
	return "", errors.New("could not locate the geobench module directory")
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// digestHarnessDirectory hashes the build-affecting files owned by this
// module. Tests and documentation are intentionally excluded. Paths and file
// lengths are framed so concatenation cannot create an ambiguous digest.
func digestHarnessDirectory(directory string) (string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != directory && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !entry.Type().IsRegular() || !isHarnessSourceFile(entry.Name()) {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", errors.New("no harness source files found")
	}
	slices.Sort(paths)

	hasher := sha256.New()
	writeDigestField(hasher, harnessDigestVersion)
	for _, relative := range paths {
		contents, err := os.ReadFile(filepath.Join(directory, filepath.FromSlash(relative)))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", relative, err)
		}
		writeDigestField(hasher, relative)
		writeDigestBytes(hasher, contents)
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func isHarnessSourceFile(name string) bool {
	if name == "go.mod" || name == "go.sum" {
		return true
	}
	if strings.HasSuffix(name, "_test.go") {
		return false
	}
	switch filepath.Ext(name) {
	case ".go", ".c", ".cc", ".cpp", ".cxx", ".h", ".hh", ".hpp", ".m", ".s", ".S", ".syso":
		return true
	default:
		return false
	}
}

func writeDigestField(writer io.Writer, value string) {
	writeDigestBytes(writer, []byte(value))
}

func writeDigestBytes(writer io.Writer, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write(value)
}

func sha256Digest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func inspectDockerImages(ctx context.Context, references []string) []DockerImageReport {
	reports := make([]DockerImageReport, 0, len(references))
	for _, reference := range references {
		report := DockerImageReport{Reference: reference}
		output, err := exec.CommandContext(ctx, "docker", "image", "inspect", reference).CombinedOutput()
		if err != nil {
			detail := strings.TrimSpace(string(output))
			if detail == "" {
				detail = err.Error()
			}
			report.Error = detail
			reports = append(reports, report)
			continue
		}
		report.ImageID, report.RepoDigests, err = parseDockerImageInspect(output)
		if err != nil {
			report.Error = err.Error()
		} else if len(report.RepoDigests) == 0 {
			report.Error = "local image has no repository digest"
		}
		reports = append(reports, report)
	}
	return reports
}

func parseDockerImageInspect(encoded []byte) (string, []string, error) {
	var inspected []struct {
		ID          string   `json:"Id"`
		RepoDigests []string `json:"RepoDigests"`
	}
	if err := json.Unmarshal(encoded, &inspected); err != nil {
		return "", nil, fmt.Errorf("decode Docker image metadata: %w", err)
	}
	if len(inspected) != 1 {
		return "", nil, fmt.Errorf("Docker returned metadata for %d images, want 1", len(inspected))
	}
	digests := append([]string(nil), inspected[0].RepoDigests...)
	slices.Sort(digests)
	digests = slices.Compact(digests)
	return inspected[0].ID, digests, nil
}

func collectBenchmarkEnvironment(ctx context.Context) map[string]string {
	environment := map[string]string{
		"arch":       runtime.GOARCH,
		"compiler":   runtime.Compiler,
		"gitDirty":   "unknown",
		"go":         runtime.Version(),
		"goMaxProcs": strconv.Itoa(runtime.GOMAXPROCS(0)),
		"logicalCpu": strconv.Itoa(runtime.NumCPU()),
		"os":         runtime.GOOS,
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "CGO_ENABLED":
				environment["cgoEnabled"] = setting.Value
			case "GOAMD64", "GOARM64", "GO386":
				environment[strings.ToLower(setting.Key)] = setting.Value
			case "vcs.revision":
				if environment["gitCommit"] == "" {
					environment["gitCommit"] = setting.Value
				}
			case "vcs.modified":
				if setting.Value == "true" || setting.Value == "false" {
					environment["gitDirty"] = setting.Value
				}
			}
		}
	}

	if contents, err := os.ReadFile("/etc/os-release"); err == nil {
		if release := parseOSRelease(contents); release != "" {
			environment["osRelease"] = release
		}
	}
	if contents, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		if kernel := strings.TrimSpace(string(contents)); kernel != "" {
			environment["kernel"] = kernel
		}
	}
	if contents, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		model, cores := parseCPUInfo(contents)
		if model != "" {
			environment["cpuModel"] = model
		}
		if cores > 0 {
			environment["physicalCores"] = strconv.Itoa(cores)
		}
	}
	if contents, err := os.ReadFile("/proc/meminfo"); err == nil {
		if totalBytes, ok := parseMemTotalBytes(contents); ok {
			environment["memoryTotalBytes"] = strconv.FormatInt(totalBytes, 10)
		}
	}

	if output, err := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").Output(); err == nil {
		environment["docker"] = strings.TrimSpace(string(output))
	}
	if output, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Driver}}").Output(); err == nil {
		if driver := strings.TrimSpace(string(output)); driver != "" {
			environment["dockerStorageDriver"] = driver
		}
	}
	if directory, err := harnessSourceDirectory(); err == nil {
		if output, err := exec.CommandContext(ctx, "git", "-C", directory, "rev-parse", "HEAD").Output(); err == nil {
			environment["gitCommit"] = strings.TrimSpace(string(output))
		}
		if output, err := exec.CommandContext(ctx, "git", "-C", directory, "status", "--porcelain=v1", "--untracked-files=normal").Output(); err == nil {
			environment["gitDirty"] = strconv.FormatBool(len(strings.TrimSpace(string(output))) > 0)
		}
	}
	return environment
}

func parseOSRelease(contents []byte) string {
	values := make(map[string]string)
	for _, line := range strings.Split(string(contents), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || key == "" {
			continue
		}
		value = strings.TrimSpace(value)
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		} else if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	if values["PRETTY_NAME"] != "" {
		return values["PRETTY_NAME"]
	}
	if values["ID"] == "" {
		return ""
	}
	if values["VERSION_ID"] == "" {
		return values["ID"]
	}
	return values["ID"] + " " + values["VERSION_ID"]
}

func parseCPUInfo(contents []byte) (string, int) {
	model := ""
	cores := make(map[string]struct{})
	for _, block := range strings.Split(strings.TrimSpace(string(contents)), "\n\n") {
		values := make(map[string]string)
		for _, line := range strings.Split(block, "\n") {
			key, value, ok := strings.Cut(line, ":")
			if ok {
				values[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
			}
		}
		if model == "" {
			for _, key := range []string{"model name", "cpu model", "hardware"} {
				if values[key] != "" {
					model = values[key]
					break
				}
			}
		}
		coreID := values["core id"]
		if coreID != "" {
			cores[values["physical id"]+":"+coreID] = struct{}{}
		}
	}
	return model, len(cores)
}

func parseMemTotalBytes(contents []byte) (int64, bool) {
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "MemTotal:" {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || value < 0 {
			return 0, false
		}
		multiplier := int64(1)
		if len(fields) >= 3 {
			switch strings.ToLower(fields[2]) {
			case "kb":
				multiplier = 1024
			case "mb":
				multiplier = 1024 * 1024
			case "gb":
				multiplier = 1024 * 1024 * 1024
			default:
				return 0, false
			}
		}
		if value > (1<<63-1)/multiplier {
			return 0, false
		}
		return value * multiplier, true
	}
	return 0, false
}
