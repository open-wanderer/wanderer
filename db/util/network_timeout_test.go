package util

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func timeoutTestClient(t *testing.T, total, headers, handshake time.Duration) *http.Client {
	t.Helper()
	client := safeHTTPClientWithTimeouts(total, headers, handshake)
	transport := client.Transport.(*http.Transport)
	// Only the test dialer reaches a local server. Production continues to vet
	// DNS answers and dial the resolved public address.
	transport.DialContext = (&net.Dialer{}).DialContext
	t.Cleanup(transport.CloseIdleConnections)
	return client
}

func TestSafeHTTPClientStopsStalledHeadersAndBody(t *testing.T) {
	for _, stage := range []string{"headers", "body"} {
		t.Run(stage, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if stage == "body" {
					io.WriteString(w, `{"partial":`)
					w.(http.Flusher).Flush()
				}
				<-r.Context().Done()
			}))
			t.Cleanup(server.Close)
			total := 80 * time.Millisecond
			if stage == "headers" {
				total = 3 * time.Second
			}
			client := timeoutTestClient(t, total, 30*time.Millisecond, 30*time.Millisecond)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				response, err := client.Do(req)
				if err == nil {
					defer response.Body.Close()
					_, err = io.ReadAll(response.Body)
				}
				done <- err
			}()
			select {
			case err := <-done:
				var netErr net.Error
				if !errors.As(err, &netErr) || !netErr.Timeout() {
					t.Fatalf("stalled %s failed without timeout: %v", stage, err)
				}
			case <-time.After(time.Second):
				cancel()
				<-done
				t.Fatalf("stalled %s exceeded client budget", stage)
			}
		})
	}
}

func TestSafeHTTPClientStopsStalledTLSHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	stop := make(chan struct{})
	defer close(stop)
	accepted := make(chan net.Conn, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		accepted <- connection
		<-stop
		connection.Close()
	}()
	client := timeoutTestClient(t, 3*time.Second, 3*time.Second, 30*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+listener.Addr().String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		response, err := client.Do(req)
		if response != nil {
			response.Body.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		var netErr net.Error
		if !errors.As(err, &netErr) || !netErr.Timeout() {
			t.Fatalf("handshake failed without timeout: %v", err)
		}
	case <-time.After(time.Second):
		cancel()
		<-done
		t.Fatal("TLS handshake exceeded transport budget")
	}
	select {
	case connection := <-accepted:
		defer connection.Close()
	case <-time.After(time.Second):
		t.Fatal("TLS server did not receive a connection")
	}
}

func TestSafeHTTPClientHonorsShorterCallerDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	t.Cleanup(server.Close)
	client := timeoutTestClient(t, time.Second, time.Second, time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("caller deadline not preserved: %v", err)
	}
}

func TestSafeHTTPClientInstallsFiniteDefaultBudgets(t *testing.T) {
	client := SafeHTTPClient()
	transport := client.Transport.(*http.Transport)
	if client.Timeout <= 0 || client.Timeout > 30*time.Second || transport.ResponseHeaderTimeout <= 0 || transport.ResponseHeaderTimeout > 10*time.Second || transport.TLSHandshakeTimeout <= 0 || transport.TLSHandshakeTimeout > 10*time.Second {
		t.Fatalf("unbounded client: total=%s headers=%s TLS=%s", client.Timeout, transport.ResponseHeaderTimeout, transport.TLSHandshakeTimeout)
	}
}
