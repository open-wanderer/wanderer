package util

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

var errConnectSuppressed = errors.New("connect suppressed")

func TestDeliveryClientIdleConnectionLimits(t *testing.T) {
	client := SafeDeliveryClient(10 * time.Second)
	t.Cleanup(client.CloseIdleConnections)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 5 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 5", transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %s, want %s", transport.IdleConnTimeout, 90*time.Second)
	}
}

// memoryDNS is an in-memory DNS server reached over net.Pipe. The n-th A query
// (0-based) is answered with answers(n); optional AAAA answers are fixed.
type memoryDNS struct {
	aQueries    atomic.Int32
	aaaaQueries atomic.Int32
	answers     func(n int) []net.IP
	aaaaAnswers []net.IP
}

func (d *memoryDNS) resolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			client, server := net.Pipe()
			go d.serve(server)
			return client, nil
		},
	}
}

func (d *memoryDNS) serve(conn net.Conn) {
	defer conn.Close()
	for {
		var lenBuf [2]byte
		if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
			return
		}
		msg := make([]byte, binary.BigEndian.Uint16(lenBuf[:]))
		if _, err := io.ReadFull(conn, msg); err != nil {
			return
		}
		var parser dnsmessage.Parser
		header, err := parser.Start(msg)
		if err != nil {
			return
		}
		question, err := parser.Question()
		if err != nil {
			return
		}
		response := dnsmessage.Message{
			Header: dnsmessage.Header{
				ID:                 header.ID,
				Response:           true,
				Authoritative:      true,
				RecursionDesired:   header.RecursionDesired,
				RecursionAvailable: true,
				RCode:              dnsmessage.RCodeSuccess,
			},
			Questions: []dnsmessage.Question{question},
		}
		switch question.Type {
		case dnsmessage.TypeA:
			n := int(d.aQueries.Add(1)) - 1
			for _, ip := range d.answers(n) {
				ip4 := ip.To4()
				if ip4 == nil {
					continue
				}
				var a [4]byte
				copy(a[:], ip4)
				response.Answers = append(response.Answers, dnsmessage.Resource{
					Header: dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 0},
					Body:   &dnsmessage.AResource{A: a},
				})
			}
		case dnsmessage.TypeAAAA:
			d.aaaaQueries.Add(1)
			for _, ip := range d.aaaaAnswers {
				var aaaa [16]byte
				copy(aaaa[:], ip.To16())
				response.Answers = append(response.Answers, dnsmessage.Resource{
					Header: dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeAAAA, Class: dnsmessage.ClassINET, TTL: 0},
					Body:   &dnsmessage.AAAAResource{AAAA: aaaa},
				})
			}
		}
		packed, err := response.Pack()
		if err != nil {
			return
		}
		out := make([]byte, 2+len(packed))
		binary.BigEndian.PutUint16(out, uint16(len(packed)))
		copy(out[2:], packed)
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}

func installMemoryDNS(t *testing.T, answers func(n int) []net.IP) *memoryDNS {
	t.Helper()
	dns := &memoryDNS{answers: answers}
	previous := net.DefaultResolver
	net.DefaultResolver = dns.resolver()
	t.Cleanup(func() { net.DefaultResolver = previous })
	return dns
}

// recordingDialer records every address the dialer is about to connect to and
// suppresses the connect itself, so no packet leaves the host.
func recordingDialer() (*net.Dialer, func() []string) {
	var mu sync.Mutex
	var dialed []string
	dialer := &net.Dialer{
		Timeout: time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			mu.Lock()
			dialed = append(dialed, address)
			mu.Unlock()
			return errConnectSuppressed
		},
	}
	return dialer, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), dialed...)
	}
}

// Rebinding: first answer public, every later answer loopback. The delivery
// dial must connect to the vetted public address only.
func TestDeliveryDialsOnlyTheCheckedAddress(t *testing.T) {
	dns := installMemoryDNS(t, func(n int) []net.IP {
		if n == 0 {
			return []net.IP{net.ParseIP("93.184.216.34")}
		}
		return []net.IP{net.ParseIP("127.0.0.1")}
	})
	dialer, dialed := recordingDialer()
	_, err := publicDialContext(dialer, false)(context.Background(), "tcp", "inbox.example:443")
	t.Logf("dialed=%v aQueries=%d aaaaQueries=%d err=%v", dialed(), dns.aQueries.Load(), dns.aaaaQueries.Load(), err)
	if err == nil || !errors.Is(err, errConnectSuppressed) {
		t.Fatalf("expected the suppressed connect, got %v", err)
	}
	got := dialed()
	if len(got) == 0 {
		t.Fatal("no connect attempt recorded")
	}
	for _, address := range got {
		if address != "93.184.216.34:443" {
			t.Fatalf("dialed %s, want only the vetted 93.184.216.34:443 (all: %v)", address, got)
		}
	}
}

// Same through the rate-limited actor-fetch variant (SafeHTTPClient path).
func TestActorFetchDialsOnlyTheCheckedAddress(t *testing.T) {
	previousLimiter := ActivityPubRateLimiter
	ActivityPubRateLimiter = NewRateLimiter(1000, time.Hour)
	t.Cleanup(func() { ActivityPubRateLimiter = previousLimiter })
	dns := installMemoryDNS(t, func(n int) []net.IP {
		if n == 0 {
			return []net.IP{net.ParseIP("93.184.216.34")}
		}
		return []net.IP{net.ParseIP("169.254.169.254")}
	})
	dialer, dialed := recordingDialer()
	_, err := publicDialContext(dialer, true)(context.Background(), "tcp", "actor.example:443")
	t.Logf("dialed=%v aQueries=%d err=%v", dialed(), dns.aQueries.Load(), err)
	got := dialed()
	if len(got) == 0 {
		t.Fatal("no connect attempt recorded")
	}
	for _, address := range got {
		if address != "93.184.216.34:443" {
			t.Fatalf("dialed %s, want only the vetted 93.184.216.34:443 (all: %v)", address, got)
		}
	}
}

// Mixed answer: one public and one private address must be refused before any
// connect.
func TestDeliveryRejectsAMixedAddressAnswer(t *testing.T) {
	installMemoryDNS(t, func(int) []net.IP {
		return []net.IP{net.ParseIP("93.184.216.34"), net.ParseIP("127.0.0.1")}
	})
	dialer, dialed := recordingDialer()
	_, err := publicDialContext(dialer, false)(context.Background(), "tcp", "mixed.example:443")
	t.Logf("mixed dialed=%v err=%v", dialed(), err)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected the policy refusal, got %v", err)
	}
	if got := dialed(); len(got) != 0 {
		t.Fatalf("connect attempted to %v despite a private answer", got)
	}
}

func TestDeliveryTriesVettedAddressesInResolverOrder(t *testing.T) {
	dns := installMemoryDNS(t, func(int) []net.IP {
		return []net.IP{net.ParseIP("9.9.9.9"), net.ParseIP("1.1.1.1")}
	})
	dialer, dialed := recordingDialer()
	_, err := publicDialContext(dialer, false)(context.Background(), "tcp", "multi.example:443")
	if !errors.Is(err, errConnectSuppressed) {
		t.Fatalf("expected the suppressed connect, got %v", err)
	}
	got := dialed()
	if len(got) != 2 || got[0] != "9.9.9.9:443" || got[1] != "1.1.1.1:443" {
		t.Fatalf("dialed %v, want both vetted addresses in resolver order", got)
	}
	if dns.aQueries.Load() != 1 {
		t.Fatalf("resolved again during fallback: A queries %d", dns.aQueries.Load())
	}
}

func TestDeliveryAddressFallbackUsesOneDialBudget(t *testing.T) {
	previousLimiter := ActivityPubRateLimiter
	ActivityPubRateLimiter = NewRateLimiter(1, time.Minute)
	t.Cleanup(func() { ActivityPubRateLimiter = previousLimiter })
	dns := installMemoryDNS(t, func(int) []net.IP {
		return []net.IP{net.ParseIP("9.9.9.9"), net.ParseIP("1.1.1.1")}
	})
	dialer, dialed := recordingDialer()
	dial := publicDialContext(dialer, true)
	if _, err := dial(context.Background(), "tcp", "multi.example:443"); !errors.Is(err, errConnectSuppressed) {
		t.Fatalf("first dial failed: %v", err)
	}
	if got := dialed(); len(got) != 2 {
		t.Fatalf("dial budget refused a fallback address: %v", got)
	}
	if _, err := dial(context.Background(), "tcp", "multi.example:443"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("second dial did not consume the single budget: %v", err)
	}
	if got := dialed(); len(got) != 2 || dns.aQueries.Load() != 1 {
		t.Fatalf("refused dial opened a connection or resolved again: %v, A queries %d", got, dns.aQueries.Load())
	}
}

func TestDeliveryAddressFallbackReturnsFirstErrorAndStopsOnCancellation(t *testing.T) {
	for _, cancelFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelFirst), func(t *testing.T) {
			installMemoryDNS(t, func(int) []net.IP {
				return []net.IP{net.ParseIP("9.9.9.9"), net.ParseIP("1.1.1.1")}
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			firstErr, laterErr := errors.New("first connection failed"), errors.New("later connection failed")
			var dialed []string
			dialer := &net.Dialer{Timeout: time.Second, ControlContext: func(_ context.Context, _ string, address string, _ syscall.RawConn) error {
				dialed = append(dialed, address)
				if len(dialed) == 1 {
					if cancelFirst {
						cancel()
					}
					return firstErr
				}
				return laterErr
			}}
			_, err := publicDialContext(dialer, false)(ctx, "tcp", "multi.example:443")
			if !errors.Is(err, firstErr) || errors.Is(err, laterErr) {
				t.Fatalf("expected the first connection error, got %v", err)
			}
			want := 2
			if cancelFirst {
				want = 1
			}
			if len(dialed) != want {
				t.Fatalf("connection attempts %v, want %d", dialed, want)
			}
		})
	}
}

func TestDeliveryAddressFallbackDividesRemainingTime(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		dialTimeout, contextTimeout time.Duration
		shares                      []time.Duration
	}{
		{"dial-timeout", 8 * time.Second, 20 * time.Second, []time.Duration{4 * time.Second, 8 * time.Second}},
		{"context-timeout", 10 * time.Second, 3 * time.Second, []time.Duration{2 * time.Second, 3 * time.Second}},
		{"three-addresses", 10 * time.Second, 3 * time.Second, []time.Duration{2 * time.Second, 2 * time.Second, 3 * time.Second}},
		{"short-context", 10 * time.Second, time.Second, []time.Duration{time.Second, time.Second}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installMemoryDNS(t, func(int) []net.IP {
				ips := []net.IP{net.ParseIP("9.9.9.9"), net.ParseIP("1.1.1.1")}
				if len(tc.shares) == 3 {
					ips = append(ips, net.ParseIP("8.8.8.8"))
				}
				return ips
			})
			ctx, cancel := context.WithTimeout(context.Background(), tc.contextTimeout)
			defer cancel()
			var deadlines []time.Time
			dialer := &net.Dialer{Timeout: tc.dialTimeout, ControlContext: func(attempt context.Context, _, _ string, _ syscall.RawConn) error {
				deadline, ok := attempt.Deadline()
				if !ok {
					t.Error("connection attempt has no deadline")
				}
				deadlines = append(deadlines, deadline)
				return errConnectSuppressed
			}}
			started := time.Now()
			_, err := publicDialContext(dialer, false)(ctx, "tcp", "multi.example:443")
			if !errors.Is(err, errConnectSuppressed) || len(deadlines) != len(tc.shares) {
				t.Fatalf("connection attempts %v, error %v", deadlines, err)
			}
			for i, want := range tc.shares {
				got := deadlines[i].Sub(started)
				if got < want-200*time.Millisecond || got > want+200*time.Millisecond {
					t.Fatalf("attempt %d deadline after %v, want about %v", i, got, want)
				}
			}
		})
	}
}

func TestDeliveryTimeoutAlsoBoundsNameLookup(t *testing.T) {
	previous := net.DefaultResolver
	release := make(chan struct{})
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		select {
		case <-release:
			return nil, errConnectSuppressed
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	t.Cleanup(func() { net.DefaultResolver = previous })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dialer, dialed := recordingDialer()
	dialer.Timeout = 50 * time.Millisecond
	finished := make(chan error, 1)
	go func() {
		_, err := publicDialContext(dialer, false)(ctx, "tcp", "multi.example:443")
		finished <- err
	}()
	select {
	case err := <-finished:
		close(release)
		if !errors.Is(err, context.DeadlineExceeded) || len(dialed()) != 0 {
			t.Fatalf("name lookup did not honor the whole-dial timeout: connections %v, error %v", dialed(), err)
		}
	case <-time.After(500 * time.Millisecond):
		close(release)
		<-finished
		t.Fatal("name lookup continued beyond the whole-dial timeout")
	}
}

func TestDeliveryAddressFamiliesAlternateInStableOrder(t *testing.T) {
	for _, tc := range []struct {
		name        string
		input, want []string
	}{
		{"ipv6-first", []string{"2001:4860::1", "2001:4860::2", "2001:4860::3", "2001:4860::4", "2001:4860::5", "9.9.9.9"}, []string{"2001:4860::1", "9.9.9.9", "2001:4860::2", "2001:4860::3", "2001:4860::4", "2001:4860::5"}},
		{"ipv4-first", []string{"9.9.9.9", "1.1.1.1", "2001:4860::1"}, []string{"9.9.9.9", "2001:4860::1", "1.1.1.1"}},
		{"ipv4-only", []string{"9.9.9.9", "1.1.1.1"}, []string{"9.9.9.9", "1.1.1.1"}},
		{"ipv6-only", []string{"2001:4860::1", "2001:4860::2"}, []string{"2001:4860::1", "2001:4860::2"}},
		{"empty", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ips []net.IP
			for _, address := range tc.input {
				ips = append(ips, net.ParseIP(address))
			}
			original := append([]net.IP(nil), ips...)
			var got []string
			for _, ip := range interleaveAddressFamilies(ips) {
				got = append(got, ip.String())
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("address order %v, want %v", got, tc.want)
			}
			if !reflect.DeepEqual(ips, original) {
				t.Fatal("resolver answer was changed in place")
			}
		})
	}
}

func TestDeliveryAddressFallbackTriesBothFamiliesEarly(t *testing.T) {
	for _, ipv4 := range [][]string{{"9.9.9.9"}, {"9.9.9.9", "1.1.1.1"}} {
		t.Run(fmt.Sprint(len(ipv4)), func(t *testing.T) {
			dns := installMemoryDNS(t, func(int) []net.IP {
				var ips []net.IP
				for _, address := range ipv4 {
					ips = append(ips, net.ParseIP(address))
				}
				return ips
			})
			for i := 1; i <= 5; i++ {
				dns.aaaaAnswers = append(dns.aaaaAnswers, net.ParseIP(fmt.Sprintf("2001:4860::%d", i)))
			}
			dialer, dialed := recordingDialer()
			_, err := publicDialContext(dialer, false)(t.Context(), "tcp", "multi.example:443")
			if !errors.Is(err, errConnectSuppressed) {
				t.Fatalf("connect error %v", err)
			}
			got := dialed()
			if len(got) != 5+len(ipv4) {
				t.Fatalf("dialed %v, want all vetted addresses", got)
			}
			first, _, err := net.SplitHostPort(got[0])
			if err != nil {
				t.Fatal(err)
			}
			second, _, err := net.SplitHostPort(got[1])
			if err != nil {
				t.Fatal(err)
			}
			if (net.ParseIP(first).To4() != nil) == (net.ParseIP(second).To4() != nil) {
				t.Fatalf("first attempts use the same family: %v", got)
			}
			if dns.aQueries.Load() != 1 || dns.aaaaQueries.Load() != 1 {
				t.Fatalf("repeated lookup: A=%d AAAA=%d", dns.aQueries.Load(), dns.aaaaQueries.Load())
			}
		})
	}
}
