package regions

import (
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
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

// protomapsBuildServer answers the Protomaps build lookup from a local server
// that publishes the dates published reports as live and answers 404 to the
// rest. probes returns every request it saw, as "METHOD /path".
func protomapsBuildServer(t *testing.T, published func(date string) bool) (probes func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		date := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".pmtiles")
		if published(date) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(SetProtomapsBuildsForTesting(server.URL, server.Client()))
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}
}

// TestNewestProtomapsBuildSkipsBuildsNotPublishedYet is GH-1531: Protomaps
// publishes about two days late, so today's and yesterday's builds answer 404.
func TestNewestProtomapsBuildSkipsBuildsNotPublishedYet(t *testing.T) {
	probes := protomapsBuildServer(t, func(date string) bool {
		return date == "20261118" || date == "20261117"
	})

	date, url, ok := newestProtomapsBuild(time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC))

	if !ok || date != "20261118" || url != protomapsBuildBaseURL+"/20261118.pmtiles" {
		t.Errorf("got (%q, %q, %v), want the 20261118 build", date, url, ok)
	}
	want := []string{"HEAD /20261120.pmtiles", "HEAD /20261119.pmtiles", "HEAD /20261118.pmtiles"}
	if got := probes(); !slices.Equal(got, want) {
		t.Errorf("probes = %v, want %v", got, want)
	}
}

func TestNewestProtomapsBuildLookbackWindow(t *testing.T) {
	cases := []struct {
		name      string
		now       time.Time
		published func(string) bool
		wantDate  string
		wantOK    bool
		wantProbe []string
	}{
		{
			name:      "today published",
			now:       time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC),
			published: func(string) bool { return true },
			wantDate:  "20261120", wantOK: true,
			wantProbe: []string{"HEAD /20261120.pmtiles"},
		},
		{
			name:      "across a year boundary",
			now:       time.Date(2027, 1, 1, 1, 0, 0, 0, time.UTC),
			published: func(d string) bool { return d == "20261230" },
			wantDate:  "20261230", wantOK: true,
			wantProbe: []string{"HEAD /20270101.pmtiles", "HEAD /20261231.pmtiles", "HEAD /20261230.pmtiles"},
		},
		{
			name:      "nothing published",
			now:       time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC),
			published: func(string) bool { return false },
			wantOK:    false,
			wantProbe: []string{
				"HEAD /20261120.pmtiles", "HEAD /20261119.pmtiles", "HEAD /20261118.pmtiles",
				"HEAD /20261117.pmtiles", "HEAD /20261116.pmtiles", "HEAD /20261115.pmtiles",
				"HEAD /20261114.pmtiles",
			},
		},
		{
			name:      "build just outside the window",
			now:       time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC),
			published: func(d string) bool { return d == "20261113" },
			wantOK:    false,
			wantProbe: []string{
				"HEAD /20261120.pmtiles", "HEAD /20261119.pmtiles", "HEAD /20261118.pmtiles",
				"HEAD /20261117.pmtiles", "HEAD /20261116.pmtiles", "HEAD /20261115.pmtiles",
				"HEAD /20261114.pmtiles",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			probes := protomapsBuildServer(t, c.published)

			date, url, ok := newestProtomapsBuild(c.now)

			if ok != c.wantOK || date != c.wantDate {
				t.Errorf("got (%q, %v), want (%q, %v)", date, ok, c.wantDate, c.wantOK)
			}
			if c.wantOK && url != protomapsBuildBaseURL+"/"+c.wantDate+".pmtiles" {
				t.Errorf("url = %q", url)
			}
			if !c.wantOK && url != "" {
				t.Errorf("url = %q, want empty", url)
			}
			if got := probes(); !slices.Equal(got, c.wantProbe) {
				t.Errorf("probes = %v, want %v", got, c.wantProbe)
			}
			if len(c.wantProbe) > protomapsLookbackDays {
				t.Errorf("case expects more probes than the lookback window")
			}
		})
	}
}

func TestNewestProtomapsBuildGivesUpWhenTheHostIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	t.Cleanup(SetProtomapsBuildsForTesting(url, &http.Client{Timeout: 2 * time.Second}))

	date, buildURL, ok := newestProtomapsBuild(time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC))

	if ok || date != "" || buildURL != "" {
		t.Errorf("got (%q, %q, %v), want no build", date, buildURL, ok)
	}
}

// TestValidProtomapsDateAndURLFormat asserts the format contract against a
// local server that publishes every date.
func TestValidProtomapsDateAndURLFormat(t *testing.T) {
	if protomapsBuildBaseURL != "https://build.protomaps.com" {
		t.Fatalf("default base URL = %q", protomapsBuildBaseURL)
	}
	protomapsBuildServer(t, func(string) bool { return true })

	date, url, ok := ValidProtomapsDateAndURL()

	if !ok {
		t.Fatal("no build found on a server that publishes every date")
	}
	if len(date) != 8 {
		t.Fatalf("date %q has length %d, want 8", date, len(date))
	}
	for _, r := range date {
		if r < '0' || r > '9' {
			t.Fatalf("date %q contains non-digit character %q", date, r)
		}
	}
	if want := protomapsBuildBaseURL + "/" + date + ".pmtiles"; url != want {
		t.Fatalf("url = %q, want %q", url, want)
	}
}
