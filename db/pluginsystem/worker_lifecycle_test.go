package pluginsystem

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/semaphore"
)

type workerTimerTestInput struct{ closes atomic.Int32 }

func (*workerTimerTestInput) Write(p []byte) (int, error) { return len(p), nil }
func (w *workerTimerTestInput) Close() error {
	w.closes.Add(1)
	return nil
}

type workerTimerTestOutput struct {
	closes atomic.Int32
	closed chan struct{}
}

func (*workerTimerTestOutput) Read([]byte) (int, error) { return 0, io.EOF }
func (r *workerTimerTestOutput) Close() error {
	if r.closes.Add(1) == 1 && r.closed != nil {
		close(r.closed)
	}
	return nil
}

func newWorkerTimerTestSession(t *testing.T) *workerRuntimeSession {
	t.Helper()
	session := &workerRuntimeSession{
		stdin:            &workerTimerTestInput{},
		stdout:           &workerTimerTestOutput{},
		stderr:           &boundedWorkerBuffer{},
		requestMaxBytes:  defaultWorkerRequestMaxBytes,
		responseMaxBytes: defaultWorkerResponseMaxBytes,
		exportTimeout:    time.Hour,
		sessionTimeout:   time.Hour,
		openedAt:         time.Now(),
	}
	session.resetSessionTimerLocked()
	t.Cleanup(func() {
		session.mu.Lock()
		defer session.mu.Unlock()
		session.stopSessionTimerLocked()
	})
	return session
}

func TestWorkerSessionTimerGenerationAfterRenewal(t *testing.T) {
	session := newWorkerTimerTestSession(t)
	oldGeneration := session.timerGeneration
	openedAt := session.openedAt
	if err := session.Renew(t.Context()); err != nil {
		t.Fatal(err)
	}
	currentGeneration := session.timerGeneration
	if currentGeneration == oldGeneration || !session.openedAt.Equal(openedAt) {
		t.Fatal("renewal must advance the idle generation, not the worker's age")
	}

	// Invoke the callback directly: model an old AfterFunc that already started
	// and was waiting for mu when Renew stopped its timer. No sleep/race needed.
	session.expireSession(oldGeneration)
	stdin := session.stdin.(*workerTimerTestInput)
	stdout := session.stdout.(*workerTimerTestOutput)
	if session.fatal || stdin.closes.Load() != 0 || stdout.closes.Load() != 0 {
		t.Fatal("stale idle callback ended a renewed session")
	}
	session.expireSession(currentGeneration)
	if !session.fatal || stdin.closes.Load() != 1 || stdout.closes.Load() != 1 {
		t.Fatal("current idle callback did not invalidate and kill the session")
	}
	if err := session.Renew(t.Context()); !IsRuntimeSessionFatalError(err) {
		t.Fatalf("expired session was revived: %v", err)
	}
	session.expireSession(currentGeneration)
	if stdin.closes.Load() != 1 || stdout.closes.Load() != 1 {
		t.Fatal("fatal session was killed a second time by an idle callback")
	}
}

func TestWorkerSessionIdleTimerActuallyExpires(t *testing.T) {
	session := newWorkerTimerTestSession(t)
	stdout := session.stdout.(*workerTimerTestOutput)
	stdout.closed = make(chan struct{})
	session.mu.Lock()
	session.sessionTimeout = 20 * time.Millisecond
	session.resetSessionTimerLocked()
	session.mu.Unlock()

	// Exercise the real AfterFunc wiring, not the direct callback test seam.
	// Only an outer deadline is asserted, never an exact scheduler interval.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-stdout.closed:
	case <-ctx.Done():
		t.Fatal("the real idle timer did not close the worker pipes")
	}
	session.mu.Lock()
	fatal := session.fatal
	session.mu.Unlock()
	if !fatal || session.stdin.(*workerTimerTestInput).closes.Load() == 0 || stdout.closes.Load() == 0 {
		t.Fatal("idle expiry did not invalidate and kill the session")
	}
	if err := session.Renew(t.Context()); !IsRuntimeSessionFatalError(err) {
		t.Fatalf("idle expiry was masked by an age-based recycle request: %v", err)
	}
}

func TestWorkerSessionRenewRequestsCooperativeRecycle(t *testing.T) {
	session := newWorkerTimerTestSession(t)
	session.openedAt = time.Now().Add(-2 * session.sessionTimeout)
	generation := session.timerGeneration
	for range 2 {
		if err := session.Renew(t.Context()); !errors.Is(err, ErrRuntimeSessionRecycleRequired) || IsRuntimeSessionFatalError(err) {
			t.Fatalf("aged healthy session must request recycling, not fail fatally: %v", err)
		}
	}
	if session.fatal || session.timerGeneration != generation || session.stdin.(*workerTimerTestInput).closes.Load() != 0 {
		t.Fatal("recycle request must not kill the worker or renew its idle timer")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := session.Renew(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("renewal ignored caller cancellation: %v", err)
	}
}

func TestWorkerSessionCloseInvalidatesTimerGeneration(t *testing.T) {
	session := newWorkerTimerTestSession(t)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	session.cmd = exec.Command(executable, "-test.run=^$")
	session.cmd.Env = append(os.Environ(), "GORACE=atexit_sleep_ms=0")
	if err := session.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	session.slot = semaphore.NewWeighted(1)
	if err := session.slot.Acquire(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	oldGeneration := session.timerGeneration
	if err := session.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	session.expireSession(oldGeneration)
	session.expireSession(session.timerGeneration)
	if session.timerGeneration == oldGeneration || session.sessionTimer != nil || session.fatal || session.stdout.(*workerTimerTestOutput).closes.Load() != 0 {
		t.Fatal("idle callback killed a closed session")
	}
	if !session.slot.TryAcquire(1) {
		t.Fatal("Close did not release the worker slot")
	}
	session.slot.Release(1)
}

func startWorkerPipeCall(t *testing.T, session *workerRuntimeSession, ctx context.Context) (<-chan workerCallOutcome, *io.PipeWriter) {
	t.Helper()
	requestReader, requestWriter := io.Pipe()
	responseReader, responseWriter := io.Pipe()
	session.stdin, session.stdout = requestWriter, responseReader
	t.Cleanup(func() {
		_ = requestReader.Close()
		_ = requestWriter.Close()
		_ = responseReader.Close()
		_ = responseWriter.Close()
	})
	result := make(chan workerCallOutcome, 1)
	go func() {
		output, err := session.Call(ctx, "detail", nil, RuntimeCallOptions{})
		result <- workerCallOutcome{output: output, err: err}
	}()
	// Reading the request proves Call has acquired both locks and suspended the
	// timer, while its result is still held back by our response pipe.
	if _, err := readWorkerMessage(requestReader, defaultWorkerRequestMaxBytes); err != nil {
		t.Fatal(err)
	}
	return result, responseWriter
}

func TestWorkerCallSuspendsIdleTimerIncludingConcurrentRenew(t *testing.T) {
	session := newWorkerTimerTestSession(t)
	session.openedAt = time.Now().Add(-2 * session.sessionTimeout)
	oldGeneration := session.timerGeneration
	result, response := startWorkerPipeCall(t, session, t.Context())
	session.mu.Lock()
	activeGeneration, active, timer := session.timerGeneration, session.activeCall, session.sessionTimer
	session.mu.Unlock()
	if !active || timer != nil || activeGeneration == oldGeneration {
		t.Fatal("Call did not suspend and invalidate its idle timer")
	}
	for range 2 {
		if err := session.Renew(t.Context()); err != nil {
			t.Fatalf("concurrent renewal must not request recycling during an export: %v", err)
		}
	}
	session.expireSession(oldGeneration)
	session.expireSession(activeGeneration)
	session.mu.Lock()
	protected := session.activeCall && !session.fatal && session.sessionTimer == nil && session.timerGeneration == activeGeneration
	session.mu.Unlock()
	if !protected {
		t.Fatal("concurrent renewal/callback armed the idle timer or killed an active export")
	}
	reply, err := workerMessageWithData(workerMessageCallResult, workerCallResult{OutputBase64: encodeWorkerBytes([]byte(`{"ok":true}`))})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeWorkerMessage(response, defaultWorkerResponseMaxBytes, reply); err != nil {
		t.Fatal(err)
	}
	outcome := <-result
	if outcome.err != nil || string(outcome.output) != `{"ok":true}` {
		t.Fatalf("active call was interrupted: %s %v", outcome.output, outcome.err)
	}
	if session.activeCall || session.sessionTimer == nil || session.timerGeneration == activeGeneration {
		t.Fatal("healthy completion did not restart the idle timer")
	}
	session.expireSession(activeGeneration)
	if session.fatal {
		t.Fatal("callback invalidated during Call killed the now-idle session")
	}
	if err := session.Renew(t.Context()); !errors.Is(err, ErrRuntimeSessionRecycleRequired) {
		t.Fatalf("completed old worker did not request recycling: %v", err)
	}
	session.expireSession(session.timerGeneration)
	if !session.fatal {
		t.Fatal("current post-call idle generation could not expire")
	}
}

func TestWorkerCallStillHonorsExportTimeoutAndCancellation(t *testing.T) {
	for _, cause := range []string{"export timeout", "context cancellation"} {
		t.Run(cause, func(t *testing.T) {
			session := newWorkerTimerTestSession(t)
			if cause == "export timeout" {
				session.exportTimeout = 250 * time.Millisecond
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			result, _ := startWorkerPipeCall(t, session, ctx)
			if cause == "context cancellation" {
				cancel()
			}
			select {
			case outcome := <-result:
				if !IsRuntimeSessionFatalError(outcome.err) || !strings.Contains(outcome.err.Error(), "worker export timeout") && !errors.Is(outcome.err, context.DeadlineExceeded) {
					t.Fatalf("call was not bounded by %s: %v", cause, outcome.err)
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("%s did not interrupt the export", cause)
			}
			if session.activeCall || !session.fatal || session.sessionTimer != nil {
				t.Fatal("failed export restarted its idle timer or left a call active")
			}
		})
	}
}
