package util

import (
	"errors"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/doyensec/safeurl"
)

func TestRemoteSyncFileClientReusesTheConnectionPolicy(t *testing.T) {
	expected := reflect.ValueOf(federatedFileHTTPConfig).Pointer()
	var callers sync.WaitGroup
	for range 8 {
		callers.Add(1)
		go func() {
			defer callers.Done()
			for range 8 {
				config, err := federatedMediaHTTPConfig("https://peer.example:8443/api/v1/list/one")
				if err != nil {
					t.Error(err)
					return
				}
				client := newRemoteSyncFileClient(config).(*safeurl.WrappedClient)
				if remoteFileClientConfigPointer(client) != expected || client.Client.Timeout != RemoteAttachmentTimeout || config.InTestMode {
					t.Error("sync client changed the shared policy")
				}
				client.CloseIdleConnections()
			}
		}()
	}
	callers.Wait()
}

func TestRemoteSyncFileRealConnectionPolicy(t *testing.T) {
	config, err := federatedMediaHTTPConfig("https://peer.example:8443/api/v1/list/one")
	if err != nil {
		t.Fatal(err)
	}
	client := newRemoteSyncFileClient(config).(*safeurl.WrappedClient)
	defer client.CloseIdleConnections()
	for _, port := range []int{80, 443, 8080, 65535} {
		t.Run(strconv.Itoa(port), func(t *testing.T) {
			if err := remoteFileDialPolicy(t, client, net.JoinHostPort("8.8.8.8", strconv.Itoa(port))); err != nil {
				t.Fatalf("public sync-file port refused: %v", err)
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

func TestRemoteSyncFileDownloadUsesRealClientPolicy(t *testing.T) {
	factory := newRemoteSyncFileClient
	defer func() { newRemoteSyncFileClient = factory }()
	for _, port := range []int{8080, 65535} {
		attempts := []string{}
		newRemoteSyncFileClient = func(config *safeurl.Config) HTTPDoer {
			client := factory(config).(*safeurl.WrappedClient)
			if remoteFileClientConfigPointer(client) != reflect.ValueOf(federatedFileHTTPConfig).Pointer() {
				t.Error("download did not use the shared configuration")
			}
			return remoteFileClientWithoutNetwork(t, client, "", &attempts)
		}
		file, cleanup, err := DownloadRemoteSyncFile(t.Context(), "https://8.8.8.8:"+strconv.Itoa(port)+"/file", "https://peer.example:8443/api/v1/trail/one", "route.gpx", 8)
		if err != nil || file == nil || string(attachmentReadAll(t, file)) != "file" || len(attempts) != 1 {
			cleanup()
			t.Fatalf("public sync file failed: %v, attempts %v", err, attempts)
		}
		cleanup()
	}
}

func TestRemoteSyncFileRedirectsRetainConnectionPolicy(t *testing.T) {
	config, err := federatedMediaHTTPConfig("https://peer.example:8443/api/v1/list/one")
	if err != nil {
		t.Fatal(err)
	}
	for _, port := range []int{80, 443, 8080, 65535} {
		for _, host := range remoteFileBlockedAddresses {
			t.Run(net.JoinHostPort(host, strconv.Itoa(port)), func(t *testing.T) {
				attempts := []string{}
				target := "https://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/file"
				client := remoteFileClientWithoutNetwork(t, newRemoteSyncFileClient(config).(*safeurl.WrappedClient), target, &attempts)
				_, err := client.Do(mustRemoteFileRequest(t, "https://8.8.8.8:443/start"))
				var blocked *safeurl.AllowedIPError
				if !errors.As(err, &blocked) || len(attempts) != 2 {
					t.Fatalf("redirect did not check both connection addresses: %v, attempts %v", err, attempts)
				}
			})
		}
	}
	for _, port := range []int{8080, 65535} {
		attempts := []string{}
		client := remoteFileClientWithoutNetwork(t, newRemoteSyncFileClient(config).(*safeurl.WrappedClient), "https://8.8.4.4:"+strconv.Itoa(port)+"/file", &attempts)
		response, err := client.Do(mustRemoteFileRequest(t, "https://8.8.8.8:443/start"))
		if err != nil || response == nil || len(attempts) != 2 {
			t.Fatalf("public redirect port refused: %v, attempts %v", err, attempts)
		}
		response.Body.Close()
	}
	attempts := []string{}
	client := remoteFileClientWithoutNetwork(t, newRemoteSyncFileClient(config).(*safeurl.WrappedClient), "http://8.8.4.4:65535/file", &attempts)
	_, err = client.Do(mustRemoteFileRequest(t, "https://8.8.8.8:443/start"))
	if err == nil || !strings.Contains(err.Error(), "downgrades") || len(attempts) != 1 {
		t.Fatalf("sync-file redirect downgrade accepted: %v, attempts %v", err, attempts)
	}
}
