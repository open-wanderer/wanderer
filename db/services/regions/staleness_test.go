package regions

import (
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNeedsVectorRebuild(t *testing.T) {
	cases := []struct {
		name       string
		storedDate string
		currDate   string
		maxAgeDays int
		want       bool
	}{
		{"one day newer -> still fresh", "20260720", "20260721", 7, false},
		{"six days newer -> still fresh", "20260715", "20260721", 7, false},
		{"seven days newer -> stale", "20260714", "20260721", 7, true},
		{"across a month boundary", "20260628", "20260705", 7, true},
		{"daily rebuilds when configured", "20260720", "20260721", 1, true},
		{"same date -> fresh", "20260721", "20260721", 1, false},
		{"never built (empty stored date) -> must build", "", "20260721", 7, true},
		{"unreadable stored date -> must build", "2026-07-21", "20260721", 7, true},
		{"unreadable current date -> keep", "20260701", "", 7, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := needsVectorRebuild(c.storedDate, c.currDate, c.maxAgeDays); got != c.want {
				t.Errorf("needsVectorRebuild(%q, %q, %d) = %v, want %v", c.storedDate, c.currDate, c.maxAgeDays, got, c.want)
			}
		})
	}
}

func TestRebuildDays(t *testing.T) {
	for value, want := range map[string]int{"": 7, "14": 14, "1": 1, "0": 7, "-3": 7, "weekly": 7} {
		t.Setenv("REGION_ARCHIVE_REBUILD_DAYS", value)
		if got := rebuildDays(); got != want {
			t.Errorf("rebuildDays() with %q = %d, want %d", value, got, want)
		}
	}
}

func TestURLExistsGivesUpOnAHungServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Accept and never answer.
			defer conn.Close()
		}
	}()

	previous := protomapsHTTPClient
	protomapsHTTPClient = &http.Client{Timeout: 200 * time.Millisecond}
	defer func() { protomapsHTTPClient = previous }()

	done := make(chan bool, 1)
	go func() { done <- urlExists("http://" + listener.Addr().String() + "/20260721.pmtiles") }()
	select {
	case exists := <-done:
		if exists {
			t.Fatal("urlExists = true for a server that never answers")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("urlExists did not return for a server that never answers")
	}
}

func TestBboxChanged(t *testing.T) {
	configBbox := [4]float64{10.0, 47.0, 13.0, 50.0}

	t.Run("identical bbox -> false", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.0, 13.0, 50.0); got != false {
			t.Errorf("bboxChanged = %v, want false", got)
		}
	})

	t.Run("min_lon differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.5, 47.0, 13.0, 50.0); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})

	t.Run("min_lat differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.5, 13.0, 50.0); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})

	t.Run("max_lon differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.0, 13.5, 50.0); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})

	t.Run("max_lat differs -> true", func(t *testing.T) {
		if got := bboxChanged(configBbox, 10.0, 47.0, 13.0, 50.5); got != true {
			t.Errorf("bboxChanged = %v, want true", got)
		}
	})
}

// TestValidProtomapsDateAndURLFormat asserts only the format contract, which
// holds regardless of whether the today or yesterday branch runs (network
// access is not mocked here — see staleness.go's doc comment).
func TestValidProtomapsDateAndURLFormat(t *testing.T) {
	date, url := ValidProtomapsDateAndURL()

	if len(date) != 8 {
		t.Fatalf("date %q has length %d, want 8", date, len(date))
	}
	for _, r := range date {
		if r < '0' || r > '9' {
			t.Fatalf("date %q contains non-digit character %q", date, r)
		}
	}

	if !strings.HasPrefix(url, "https://build.protomaps.com/") {
		t.Fatalf("url %q does not have expected prefix", url)
	}
	if !strings.HasSuffix(url, date+".pmtiles") {
		t.Fatalf("url %q does not have suffix %q", url, date+".pmtiles")
	}
}
