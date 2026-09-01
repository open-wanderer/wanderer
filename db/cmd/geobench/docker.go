package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type meiliContainer struct {
	name        string
	image       string
	version     string
	url         string
	dataDir     string
	httpClient  *http.Client
	statsClient *http.Client
}

func startMeiliContainer(ctx context.Context, image, workRoot, suffix string) (*meiliContainer, error) {
	return startMeiliContainerWithThreads(ctx, image, workRoot, suffix, 0)
}

func startMeiliContainerWithThreads(ctx context.Context, image, workRoot, suffix string, indexingThreads int) (*meiliContainer, error) {
	statsClient, err := dockerStatsClient(ctx)
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("wanderer-geobench-%d-%d-%s", os.Getpid(), time.Now().UnixNano(), sanitizeName(suffix))
	dataDir, err := os.MkdirTemp(workRoot, sanitizeName(suffix)+"-")
	if err != nil {
		return nil, fmt.Errorf("create Meilisearch data directory: %w", err)
	}

	uidGID := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	args := []string{
		"run", "--detach", "--name", name,
		"--user", uidGID,
		"--publish", "127.0.0.1::7700",
		"--volume", dataDir + ":/meili_data",
		"--env", "MEILI_ENV=development",
		"--env", "MEILI_NO_ANALYTICS=true",
		"--env", "MEILI_LOG_LEVEL=ERROR",
		"--env", "MEILI_DB_PATH=/meili_data/data.ms",
	}
	if indexingThreads > 0 {
		args = append(args, "--env", "MEILI_MAX_INDEXING_THREADS="+strconv.Itoa(indexingThreads))
	}
	args = append(args, image)
	command := exec.CommandContext(ctx, "docker", args...)
	if output, err := command.CombinedOutput(); err != nil {
		if cleanupErr := removeContainer(name); cleanupErr != nil {
			return nil, fmt.Errorf("start %s: %w: %s; cleanup: %v", image, err, strings.TrimSpace(string(output)), cleanupErr)
		}
		return nil, fmt.Errorf("start %s: %w: %s", image, err, strings.TrimSpace(string(output)))
	}
	port, err := publishedContainerPort(ctx, name)
	if err != nil {
		if cleanupErr := removeContainer(name); cleanupErr != nil {
			return nil, fmt.Errorf("%w; cleanup: %v", err, cleanupErr)
		}
		return nil, err
	}

	container := &meiliContainer{
		name:        name,
		image:       image,
		url:         fmt.Sprintf("http://127.0.0.1:%d", port),
		dataDir:     dataDir,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		statsClient: statsClient,
	}
	if err := container.waitReady(ctx); err != nil {
		logs := container.logs()
		container.close()
		return nil, fmt.Errorf("wait for %s: %w\n%s", image, err, logs)
	}
	if _, err := container.memoryBytes(); err != nil {
		container.close()
		return nil, fmt.Errorf("read Meilisearch container RAM: %w", err)
	}
	return container, nil
}

func dockerStatsClient(ctx context.Context) (*http.Client, error) {
	dockerHost := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	if dockerHost == "" {
		output, err := exec.CommandContext(ctx, "docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}").Output()
		if err != nil {
			return nil, fmt.Errorf("inspect Docker endpoint: %w", err)
		}
		dockerHost = strings.TrimSpace(string(output))
	}
	if !strings.HasPrefix(dockerHost, "unix://") {
		return nil, fmt.Errorf("benchmark requires a local Unix Docker endpoint for bind mounts and RAM metrics, got %q", dockerHost)
	}
	socketPath := strings.TrimPrefix(dockerHost, "unix://")
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &http.Client{Transport: transport, Timeout: 2 * time.Second}, nil
}

func publishedContainerPort(ctx context.Context, name string) (int, error) {
	output, err := exec.CommandContext(ctx, "docker", "port", name, "7700/tcp").Output()
	if err != nil {
		return 0, fmt.Errorf("inspect Meilisearch port: %w", err)
	}
	line := strings.Split(strings.TrimSpace(string(output)), "\n")[0]
	_, portValue, err := net.SplitHostPort(line)
	if err != nil {
		return 0, fmt.Errorf("parse Meilisearch port %q: %w", line, err)
	}
	port, err := strconv.Atoi(portValue)
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("invalid Meilisearch port %q", portValue)
	}
	return port, nil
}

func (container *meiliContainer) memoryBytes() (int64, error) {
	endpoint := "http://docker/containers/" + url.PathEscape(container.name) + "/stats?stream=false&one-shot=true"
	response, err := container.statsClient.Get(endpoint)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Docker stats returned %s", response.Status)
	}
	var stats struct {
		Memory struct {
			Usage uint64            `json:"usage"`
			Stats map[string]uint64 `json:"stats"`
		} `json:"memory_stats"`
	}
	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return 0, err
	}
	usage := stats.Memory.Usage
	// This matches Docker's working-set convention on cgroup v2 and avoids
	// counting reclaimable file cache as resident index memory.
	inactiveFile, ok := stats.Memory.Stats["total_inactive_file"]
	if !ok {
		inactiveFile = stats.Memory.Stats["inactive_file"]
	}
	if inactiveFile < usage {
		usage -= inactiveFile
	}
	return int64(usage), nil
}

func (container *meiliContainer) waitReady(ctx context.Context) error {
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, container.url+"/version", nil)
		if err != nil {
			return err
		}
		response, err := container.httpClient.Do(request)
		if err == nil {
			var version struct {
				PkgVersion string `json:"pkgVersion"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&version)
			response.Body.Close()
			if response.StatusCode == http.StatusOK && decodeErr == nil {
				container.version = version.PkgVersion
				return nil
			}
		}
		if exited, state := container.exited(ctx); exited {
			return fmt.Errorf("container stopped before becoming ready: %s", state)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("startup timeout")
		case <-ticker.C:
		}
	}
}

func (container *meiliContainer) exited(parent context.Context) (bool, string) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(
		ctx,
		"docker",
		"inspect",
		"--format",
		"{{.State.Status}} exit={{.State.ExitCode}} error={{.State.Error}}",
		container.name,
	).Output()
	if err != nil {
		return false, ""
	}
	state := strings.TrimSpace(string(output))
	return strings.HasPrefix(state, "exited ") || strings.HasPrefix(state, "dead "), state
}

func (container *meiliContainer) logs() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, _ := exec.CommandContext(ctx, "docker", "logs", "--tail", "100", container.name).CombinedOutput()
	return strings.TrimSpace(string(output))
}

func (container *meiliContainer) close() {
	container.httpClient.CloseIdleConnections()
	container.statsClient.CloseIdleConnections()
	if err := removeContainer(container.name); err != nil {
		log.Printf("warning   remove container %s: %v", container.name, err)
	}
}

func removeContainer(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", "rm", "--force", name).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if strings.Contains(strings.ToLower(message), "no such container") {
			return nil
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}

func sanitizeName(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-_")
}
