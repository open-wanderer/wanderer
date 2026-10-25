package util

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/doyensec/safeurl"
)

var remoteFileBlockedAddresses = []string{
	"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254",
	"::1", "fc00::1", "fe80::1", "::7f00:1", "::808:808",
}

func remoteFileClientConfigPointer(client *safeurl.WrappedClient) uintptr {
	return reflect.ValueOf(client).Elem().FieldByName("config").Pointer()
}

// The real transport's net.Dialer.Control runs before connecting a Unix
// socket too. IP:port is a valid relative socket name, so a missing name
// gives ENOENT only after the unchanged IP and port policy accepted it.
// No IP connection, DNS lookup or substitute policy is needed.
func remoteFileDialPolicy(t *testing.T, client *safeurl.WrappedClient, address string) error {
	t.Helper()
	transport, ok := client.Client.Transport.(*http.Transport)
	if !ok || transport.DialContext == nil {
		t.Fatal("client has no production dialer")
	}
	return remoteFileTransportDialPolicy(t, transport, address)
}

func remoteFileTransportDialPolicy(t *testing.T, transport *http.Transport, address string) error {
	t.Helper()
	if _, err := os.Lstat(address); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("policy probe needs an absent Unix socket %q: %v", address, err)
	}
	conn, err := transport.DialContext(t.Context(), "unix", address)
	if conn != nil {
		conn.Close()
		t.Fatal("policy probe unexpectedly connected")
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// HTTP is synthetic only after each hop passed the original DialContext and
// Control. The real WrappedClient and http.Client still validate the initial
// URL and redirects. The shared safeurl.Config is never changed.
func remoteFileClientWithoutNetwork(t *testing.T, client *safeurl.WrappedClient, redirect string, attempts *[]string) HTTPDoer {
	t.Helper()
	originalTransport := client.Client.Transport.(*http.Transport)
	client.Client.Transport = attachmentRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		*attempts = append(*attempts, request.URL.String())
		port := request.URL.Port()
		if port == "" {
			port = "80"
			if request.URL.Scheme == "https" {
				port = "443"
			}
		}
		// Read the actual constructor's transport, rather than rebuilding a
		// policy from fields that a test author believes should be present.
		err := remoteFileTransportDialPolicy(t, originalTransport, net.JoinHostPort(request.URL.Hostname(), port))
		if err != nil {
			return nil, err
		}
		status, body, header := http.StatusOK, "file", make(http.Header)
		if request.URL.Path == "/start" && redirect != "" {
			status, body = http.StatusFound, ""
			header.Set("Location", redirect)
		}
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	return client
}

func TestRemoteAttachmentClientSharesAllPortConfiguration(t *testing.T) {
	config := federatedFileHTTPConfig
	if len(config.AllowedPorts) != 65535 || config.Timeout != RemoteAttachmentTimeout || !config.IsIPv6Enabled || config.InTestMode || config.AllowSendingCredentials || !slices.Equal(config.AllowedSchemes, []string{"http", "https"}) || config.AllowedIPs != nil || config.AllowedIPsCIDR != nil || config.AllowedHosts != nil {
		t.Fatal("incoming file configuration has unexpected bounds or policy")
	}
	for i, port := range config.AllowedPorts {
		if port != i+1 {
			t.Fatalf("port %d missing", i+1)
		}
	}
	if len(config.BlockedIPsCIDR) != 1 || config.BlockedIPsCIDR[0].String() != "::/96" {
		t.Fatal("missing compatible-address restriction")
	}
	expected := reflect.ValueOf(config).Pointer()
	var callers sync.WaitGroup
	for range 8 {
		callers.Add(1)
		go func() {
			defer callers.Done()
			for range 8 {
				client := newRemoteAttachmentClient(RemoteAttachmentTimeout, publicMediaRedirectPolicy).(*safeurl.WrappedClient)
				if client.Client.Timeout != RemoteAttachmentTimeout || reflect.ValueOf(client.Client.CheckRedirect).Pointer() != reflect.ValueOf(publicMediaRedirectPolicy).Pointer() {
					t.Error("client replaced the request deadline or redirect policy")
				}
				if remoteFileClientConfigPointer(client) != expected {
					t.Error("download rebuilt its configuration")
				}
				client.CloseIdleConnections()
			}
		}()
	}
	callers.Wait()
}

func TestRemoteAttachmentClientRealConnectionPolicy(t *testing.T) {
	client := newRemoteAttachmentClient(RemoteAttachmentTimeout, publicMediaRedirectPolicy).(*safeurl.WrappedClient)
	defer client.CloseIdleConnections()
	for _, port := range []int{80, 443, 8080, 65535} {
		t.Run(strconv.Itoa(port), func(t *testing.T) {
			if err := remoteFileDialPolicy(t, client, net.JoinHostPort("8.8.8.8", strconv.Itoa(port))); err != nil {
				t.Fatalf("public port refused: %v", err)
			}
			for _, host := range remoteFileBlockedAddresses {
				err := remoteFileDialPolicy(t, client, net.JoinHostPort(host, strconv.Itoa(port)))
				var blocked *safeurl.AllowedIPError
				if !errors.As(err, &blocked) {
					t.Errorf("address %s did not reach IP policy: %v", host, err)
				}
			}
		})
	}
}

func TestRemoteAttachmentDownloadRealClientPolicy(t *testing.T) {
	t.Cleanup(SetRemoteAttachmentClientForTesting(nil))
	factory := newRemoteAttachmentClient
	defer func() { newRemoteAttachmentClient = factory }()
	for _, port := range []int{8080, 65535} {
		t.Run(fmt.Sprint(port), func(t *testing.T) {
			attempts := []string{}
			newRemoteAttachmentClient = func(timeout time.Duration, redirect func(*http.Request, []*http.Request) error) HTTPDoer {
				return remoteFileClientWithoutNetwork(t, factory(timeout, redirect).(*safeurl.WrappedClient), "", &attempts)
			}
			file, cleanup, err := DownloadRemoteFile(t.Context(), "https://8.8.8.8:"+strconv.Itoa(port)+"/file", 8)
			defer cleanup()
			if err != nil || file == nil || string(attachmentReadAll(t, file)) != "file" || len(attempts) != 1 {
				t.Fatalf("production attachment request failed: %v, attempts %v", err, attempts)
			}
		})
	}
}

func TestRemoteAttachmentRedirectsRetainConnectionPolicy(t *testing.T) {
	factory := newRemoteAttachmentClient
	for _, port := range []int{80, 443, 8080, 65535} {
		for _, host := range remoteFileBlockedAddresses {
			t.Run(net.JoinHostPort(host, strconv.Itoa(port)), func(t *testing.T) {
				attempts := []string{}
				target := "https://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/file"
				client := remoteFileClientWithoutNetwork(t, factory(RemoteAttachmentTimeout, publicMediaRedirectPolicy).(*safeurl.WrappedClient), target, &attempts)
				_, err := client.Do(mustRemoteFileRequest(t, "https://8.8.8.8:443/start"))
				var blocked *safeurl.AllowedIPError
				if !errors.As(err, &blocked) || len(attempts) != 2 {
					t.Fatalf("redirect did not apply IP policy at both hops: %v, attempts %v", err, attempts)
				}
			})
		}
	}
	for _, port := range []int{8080, 65535} {
		attempts := []string{}
		client := remoteFileClientWithoutNetwork(t, factory(RemoteAttachmentTimeout, publicMediaRedirectPolicy).(*safeurl.WrappedClient), "https://8.8.4.4:"+strconv.Itoa(port)+"/file", &attempts)
		response, err := client.Do(mustRemoteFileRequest(t, "https://8.8.8.8:443/start"))
		if err != nil || response == nil || len(attempts) != 2 {
			t.Fatalf("public redirect port refused: %v, attempts %v", err, attempts)
		}
		response.Body.Close()
	}
	attempts := []string{}
	client := remoteFileClientWithoutNetwork(t, factory(RemoteAttachmentTimeout, publicMediaRedirectPolicy).(*safeurl.WrappedClient), "http://8.8.4.4:8080/file", &attempts)
	_, err := client.Do(mustRemoteFileRequest(t, "https://8.8.8.8:443/start"))
	if err == nil || !strings.Contains(err.Error(), "downgrades") || len(attempts) != 1 {
		t.Fatalf("redirect downgrade accepted: %v, attempts %v", err, attempts)
	}
}

func mustRemoteFileRequest(t *testing.T, raw string) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func TestPluginMediaClientsRetainStandardPorts(t *testing.T) {
	client := NewSafeURLClient(time.Second, publicMediaRedirectPolicy).(*safeurl.WrappedClient)
	defer client.CloseIdleConnections()
	for _, port := range []int{80, 443, 8080, 65535} {
		err := remoteFileDialPolicy(t, client, net.JoinHostPort("8.8.8.8", strconv.Itoa(port)))
		var blocked *safeurl.AllowedPortError
		if (port == 80 || port == 443) && err != nil {
			t.Errorf("standard port refused: %v", err)
		}
		if (port == 8080 || port == 65535) && !errors.As(err, &blocked) {
			t.Errorf("plugin nonstandard port accepted: %v", err)
		}
	}
	factory := newPublicURLClient
	defer func() { newPublicURLClient = factory }()
	for _, port := range []int{8080, 65535} {
		attempts := []string{}
		newPublicURLClient = func(timeout time.Duration, redirect func(*http.Request, []*http.Request) error) HTTPDoer {
			return remoteFileClientWithoutNetwork(t, factory(timeout, redirect).(*safeurl.WrappedClient), "", &attempts)
		}
		_, err := FetchPublicURL(t.Context(), "https://8.8.8.8:"+strconv.Itoa(port)+"/file", 8)
		var blocked *safeurl.AllowedPortError
		if !errors.As(err, &blocked) || len(attempts) != 1 {
			t.Fatalf("plugin fetch expanded its port policy: %v, attempts %v", err, attempts)
		}
	}
	for _, raw := range []string{"ftp://8.8.8.8:8080/file", "https://user:password@8.8.8.8:8080/file"} {
		attempts := []string{}
		incoming := remoteFileClientWithoutNetwork(t, newRemoteAttachmentClient(RemoteAttachmentTimeout, publicMediaRedirectPolicy).(*safeurl.WrappedClient), "", &attempts)
		if _, err := incoming.Do(mustRemoteFileRequest(t, raw)); err == nil || len(attempts) != 0 {
			t.Fatalf("initial URL policy was not applied: %s %v", raw, err)
		}
	}
}
