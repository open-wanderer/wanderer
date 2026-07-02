package pluginsystem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

const (
	defaultWorkerExportTimeout       = 2 * time.Minute
	defaultWorkerSessionTimeout      = 15 * time.Minute
	defaultWorkerSlotAcquireTimeout  = 30 * time.Second
	defaultWorkerCapturedStderrBytes = 64 * 1024
)

var (
	workerSlotsMu sync.Mutex
	workerSlots   *semaphore.Weighted
)

type WorkerRuntime struct {
	Executable string
}

type RuntimeSessionFatalError struct {
	Err error
}

func (e RuntimeSessionFatalError) Error() string {
	return e.Err.Error()
}

func (e RuntimeSessionFatalError) Unwrap() error {
	return e.Err
}

func IsRuntimeSessionFatalError(err error) bool {
	var fatal RuntimeSessionFatalError
	return errors.As(err, &fatal)
}

func NewWorkerRuntime() WorkerRuntime {
	return WorkerRuntime{}
}

func (r WorkerRuntime) Call(ctx context.Context, plugin LocalPlugin, export string, input []byte, policy RequestPolicyContext) ([]byte, error) {
	session, err := r.OpenSession(ctx, plugin, policy)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = session.Close(context.Background())
	}()
	return session.Call(ctx, export, input)
}

func (r WorkerRuntime) OpenSession(ctx context.Context, plugin LocalPlugin, policy RequestPolicyContext) (RuntimeSession, error) {
	slot, err := acquireWorkerSlot(ctx)
	if err != nil {
		return nil, err
	}
	releaseSlot := true
	defer func() {
		if releaseSlot {
			slot.Release(1)
		}
	}()

	executable := r.Executable
	if executable == "" {
		if configured := strings.TrimSpace(os.Getenv("WANDERER_PLUGIN_WORKER_BIN")); configured != "" {
			executable = configured
		} else {
			var err error
			executable, err = os.Executable()
			if err != nil {
				return nil, err
			}
		}
	}

	cmd := exec.Command(executable, "plugin-worker")
	cmd.Env = childEnvWithout("EXTISM_ENABLE_WASI_OUTPUT")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := &boundedWorkerBuffer{limit: envInt("WANDERER_PLUGIN_WORKER_STDERR_BYTES", defaultWorkerCapturedStderrBytes)}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	session := &workerRuntimeSession{
		plugin:           plugin,
		policy:           policy,
		sessionID:        newWorkerSessionID(plugin.Manifest.ID),
		cmd:              cmd,
		stdin:            stdin,
		stdout:           stdout,
		stderr:           stderr,
		requestMaxBytes:  envInt("WANDERER_PLUGIN_WORKER_REQUEST_BYTES", defaultWorkerRequestMaxBytes),
		responseMaxBytes: envInt("WANDERER_PLUGIN_WORKER_RESPONSE_BYTES", defaultWorkerResponseMaxBytes),
		exportTimeout:    envDuration("WANDERER_PLUGIN_WORKER_EXPORT_TIMEOUT", defaultWorkerExportTimeout),
		slot:             slot,
	}
	session.sessionTimer = time.AfterFunc(envDuration("WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT", defaultWorkerSessionTimeout), func() {
		session.markFatal("worker session timeout")
		session.kill()
	})

	releaseSlot = false
	return session, nil
}

type workerRuntimeSession struct {
	plugin           LocalPlugin
	policy           RequestPolicyContext
	sessionID        string
	cmd              *exec.Cmd
	stdin            io.WriteCloser
	stdout           io.ReadCloser
	stderr           *boundedWorkerBuffer
	requestMaxBytes  int
	responseMaxBytes int
	exportTimeout    time.Duration
	sessionTimer     *time.Timer
	slot             *semaphore.Weighted

	mu       sync.Mutex
	callMu   sync.Mutex
	waitMu   sync.Mutex
	waited   bool
	waitErr  error
	closed   bool
	fatal    bool
	fatalMsg string
}

func (s *workerRuntimeSession) Call(ctx context.Context, export string, input []byte) ([]byte, error) {
	s.callMu.Lock()
	defer s.callMu.Unlock()

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("worker session is closed")
	}
	if s.fatal {
		msg := s.fatalMsg
		s.mu.Unlock()
		return nil, RuntimeSessionFatalError{Err: fmt.Errorf("worker session is invalid: %s", msg)}
	}
	s.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, s.exportTimeout)
	defer cancel()

	result := make(chan workerCallOutcome, 1)
	go func() {
		result <- s.call(callCtx, export, input)
	}()

	select {
	case outcome := <-result:
		if outcome.err != nil {
			return nil, outcome.err
		}
		return outcome.output, nil
	case <-callCtx.Done():
		s.markFatal("worker export timeout")
		s.kill()
		outcome := <-result
		if outcome.err != nil && !errors.Is(outcome.err, io.EOF) {
			return nil, RuntimeSessionFatalError{Err: fmt.Errorf("worker export timeout: %w", outcome.err)}
		}
		return nil, RuntimeSessionFatalError{Err: callCtx.Err()}
	}
}

type workerCallOutcome struct {
	output []byte
	err    error
}

func (s *workerRuntimeSession) call(ctx context.Context, export string, input []byte) workerCallOutcome {
	msg, err := workerMessageWithData(workerMessageCallExport, workerCallExport{
		WASMPath:    s.plugin.WASMPath,
		Export:      export,
		InputBase64: encodeWorkerBytes(input),
		SessionID:   s.sessionID,
	})
	if err != nil {
		return workerCallOutcome{err: err}
	}
	if err := writeWorkerMessage(s.stdin, s.requestMaxBytes, msg); err != nil {
		s.markFatal("write worker call_export failed")
		s.kill()
		return workerCallOutcome{err: RuntimeSessionFatalError{Err: err}}
	}

	for {
		msg, err := readWorkerMessage(s.stdout, s.responseMaxBytes)
		if err != nil {
			s.markFatal("read worker message failed")
			s.kill()
			return workerCallOutcome{err: RuntimeSessionFatalError{Err: s.withStderr(err)}}
		}
		switch msg.Type {
		case workerMessageHostHTTPRequest:
			if err := s.handleHostHTTPRequest(ctx, msg); err != nil {
				s.markFatal("host http rpc failed")
				s.kill()
				return workerCallOutcome{err: RuntimeSessionFatalError{Err: s.withStderr(err)}}
			}
		case workerMessageHostLog:
			s.handleHostLog(msg)
		case workerMessageCallResult:
			result, err := workerData[workerCallResult](msg)
			if err != nil {
				s.markFatal("invalid call_result payload")
				s.kill()
				return workerCallOutcome{err: RuntimeSessionFatalError{Err: err}}
			}
			if result.PluginError != nil {
				return workerCallOutcome{err: PluginCallError{
					PluginID:    s.plugin.Manifest.ID,
					Export:      export,
					PluginError: *result.PluginError,
				}}
			}
			output, err := decodeWorkerBytes(result.OutputBase64)
			if err != nil {
				s.markFatal("invalid call_result output")
				s.kill()
				return workerCallOutcome{err: RuntimeSessionFatalError{Err: err}}
			}
			return workerCallOutcome{output: output}
		case workerMessageError:
			payload, _ := workerData[workerError](msg)
			s.markFatal(payload.Message)
			s.kill()
			if payload.Message == "" {
				payload.Message = "worker returned fatal error"
			}
			return workerCallOutcome{err: RuntimeSessionFatalError{Err: s.withStderr(fmt.Errorf("%s", payload.Message))}}
		default:
			s.markFatal("unexpected worker message")
			s.kill()
			return workerCallOutcome{err: RuntimeSessionFatalError{Err: fmt.Errorf("unexpected worker message %q", msg.Type)}}
		}
	}
}

func (s *workerRuntimeSession) handleHostLog(msg workerMessage) {
	entry, err := workerData[workerHostLog](msg)
	if err != nil {
		log.Printf("plugin log invalid: session %s: %v", s.sessionID, err)
		return
	}
	if entry.SessionID == "" {
		entry.SessionID = s.sessionID
	}
	level, err := normalizeHostLogLevel(entry.Level)
	if err != nil {
		log.Printf("plugin log invalid: session %s: %v", s.sessionID, err)
		return
	}
	message := sanitizeHostLogMessage(entry.Message)
	if message == "" {
		log.Printf("plugin log invalid: session %s: log message is required", s.sessionID)
		return
	}
	log.Printf("plugin log [%s]: session %s: %s", level, entry.SessionID, message)
}

func (s *workerRuntimeSession) handleHostHTTPRequest(ctx context.Context, msg workerMessage) error {
	request, err := workerData[workerHostHTTPRequest](msg)
	if err != nil {
		return err
	}
	requestBytes, err := decodeWorkerBytes(request.RequestBase64)
	if err != nil {
		return err
	}
	response := executeHostHTTPRequest(ctx, s.plugin.Manifest, s.policy, requestBytes)
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return err
	}
	reply, err := workerMessageWithData(workerMessageHostHTTPResponse, workerHostHTTPResponse{
		ResponseBase64: encodeWorkerBytes(responseBytes),
	})
	if err != nil {
		return err
	}
	return writeWorkerMessage(s.stdin, s.responseMaxBytes, reply)
}

func (s *workerRuntimeSession) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	fatal := s.fatal
	s.mu.Unlock()

	if s.sessionTimer != nil {
		s.sessionTimer.Stop()
	}
	if !fatal {
		_ = writeWorkerMessage(s.stdin, s.requestMaxBytes, workerMessage{Type: workerMessageShutdown})
	}
	_ = s.stdin.Close()

	wait := make(chan error, 1)
	go func() {
		wait <- s.wait()
	}()

	select {
	case err := <-wait:
		s.slot.Release(1)
		if err != nil && !fatal {
			return s.withStderr(err)
		}
		return nil
	case <-ctx.Done():
		s.kill()
		err := <-wait
		s.slot.Release(1)
		if err != nil {
			return s.withStderr(err)
		}
		return ctx.Err()
	}
}

func (s *workerRuntimeSession) wait() error {
	s.waitMu.Lock()
	defer s.waitMu.Unlock()
	if s.waited {
		return s.waitErr
	}
	s.waited = true
	s.waitErr = s.cmd.Wait()
	return s.waitErr
}

func (s *workerRuntimeSession) markFatal(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fatal = true
	if s.fatalMsg == "" {
		s.fatalMsg = msg
	}
}

func (s *workerRuntimeSession) kill() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.stdin.Close()
	_ = s.stdout.Close()
}

func (s *workerRuntimeSession) withStderr(err error) error {
	if err == nil {
		return nil
	}
	stderr := strings.TrimSpace(s.stderr.String())
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w: worker stderr: %s", err, stderr)
}

func acquireWorkerSlot(ctx context.Context) (*semaphore.Weighted, error) {
	timeout := envDuration("WANDERER_PLUGIN_WORKER_SLOT_TIMEOUT", defaultWorkerSlotAcquireTimeout)
	acquireCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	slot := workerSemaphore()
	if err := slot.Acquire(acquireCtx, 1); err != nil {
		return nil, fmt.Errorf("acquire plugin worker slot: %w", err)
	}
	return slot, nil
}

func workerSemaphore() *semaphore.Weighted {
	limit := int64(envInt("WANDERER_PLUGIN_WORKER_MAX", maxInt(2, runtime.NumCPU())))
	workerSlotsMu.Lock()
	defer workerSlotsMu.Unlock()
	if workerSlots == nil {
		workerSlots = semaphore.NewWeighted(limit)
	}
	return workerSlots
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if value, err := time.ParseDuration(raw); err == nil && value > 0 {
		return value
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func childEnvWithout(keys ...string) []string {
	blocked := map[string]bool{}
	for _, key := range keys {
		blocked[key] = true
	}
	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key := entry
		if idx := strings.IndexByte(entry, '='); idx >= 0 {
			key = entry[:idx]
		}
		if blocked[key] {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func newWorkerSessionID(pluginID string) string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return pluginID + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return pluginID + "-" + hex.EncodeToString(random[:])
}

type boundedWorkerBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func (b *boundedWorkerBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit <= 0 || len(b.data) >= b.limit {
		return len(p), nil
	}
	remaining := b.limit - len(b.data)
	if len(p) > remaining {
		b.data = append(b.data, p[:remaining]...)
		return len(p), nil
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *boundedWorkerBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
