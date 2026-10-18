package regions

import (
	"errors"
	"strings"
	"testing"
)

func TestPolySourceURLs(t *testing.T) {
	t.Run("returns exactly two URLs for a valid input", func(t *testing.T) {
		urls, err := PolySourceURLs("2528fbb91977201cf6d16b1b01ebf27eea342e85", "Abkhazia")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(urls) != 2 {
			t.Fatalf("got %d URLs, want 2", len(urls))
		}
	})

	t.Run("index 0 is GitHub, index 1 is Codeberg (fallback order)", func(t *testing.T) {
		urls, err := PolySourceURLs("2528fbb91977201cf6d16b1b01ebf27eea342e85", "Abkhazia")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(urls[0], "https://raw.githubusercontent.com/comaps/comaps/") {
			t.Errorf("urls[0] = %q, want GitHub mirror prefix", urls[0])
		}
		if !strings.HasPrefix(urls[1], "https://codeberg.org/comaps/comaps/raw/commit/") {
			t.Errorf("urls[1] = %q, want Codeberg canonical prefix", urls[1])
		}
	})

	t.Run("verified live-200 URL forms for Abkhazia at the pinned commit", func(t *testing.T) {
		const commit = "2528fbb91977201cf6d16b1b01ebf27eea342e85"
		urls, err := PolySourceURLs(commit, "Abkhazia")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantGitHub := "https://raw.githubusercontent.com/comaps/comaps/2528fbb91977201cf6d16b1b01ebf27eea342e85/data/borders/Abkhazia.poly"
		wantCodeberg := "https://codeberg.org/comaps/comaps/raw/commit/2528fbb91977201cf6d16b1b01ebf27eea342e85/data/borders/Abkhazia.poly"
		if urls[0] != wantGitHub {
			t.Errorf("urls[0] = %q, want %q", urls[0], wantGitHub)
		}
		if urls[1] != wantCodeberg {
			t.Errorf("urls[1] = %q, want %q", urls[1], wantCodeberg)
		}
	})

	t.Run("escapes a space-bearing id as %20 in both URLs", func(t *testing.T) {
		urls, err := PolySourceURLs("2528fbb91977201cf6d16b1b01ebf27eea342e85", "Antigua and Barbuda")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, u := range urls {
			if !strings.HasSuffix(u, "Antigua%20and%20Barbuda.poly") {
				t.Errorf("URL %q does not end with the verified escaped form", u)
			}
		}
	})

	assertRejected := func(t *testing.T, commit, id string) {
		t.Helper()
		urls, err := PolySourceURLs(commit, id)
		if err == nil {
			t.Fatalf("PolySourceURLs(%q, %q) succeeded, want error", commit, id)
		}
		if urls != nil {
			t.Errorf("PolySourceURLs(%q, %q) returned non-nil URLs on error: %v", commit, id, urls)
		}
	}

	t.Run("rejects non-hex commit ref 'main' (SSRF via unpinned ref)", func(t *testing.T) {
		assertRejected(t, "main", "Abkhazia")
	})
	t.Run("rejects path-traversal commit value", func(t *testing.T) {
		assertRejected(t, "../../etc", "Abkhazia")
	})
	t.Run("rejects uppercase hex commit (allow-list is lowercase-only)", func(t *testing.T) {
		assertRejected(t, "2528FBB9", "Abkhazia")
	})
	t.Run("rejects empty commit", func(t *testing.T) {
		assertRejected(t, "", "Abkhazia")
	})
	t.Run("rejects path-traversal id", func(t *testing.T) {
		assertRejected(t, "2528fbb91977201cf6d16b1b01ebf27eea342e85", "../etc/passwd")
	})
	t.Run("rejects id containing a path separator", func(t *testing.T) {
		assertRejected(t, "2528fbb91977201cf6d16b1b01ebf27eea342e85", "a/b")
	})
	t.Run("rejects bare '..' id", func(t *testing.T) {
		assertRejected(t, "2528fbb91977201cf6d16b1b01ebf27eea342e85", "..")
	})
	t.Run("rejects id with a leading space", func(t *testing.T) {
		assertRejected(t, "2528fbb91977201cf6d16b1b01ebf27eea342e85", " Abkhazia")
	})
	t.Run("rejects empty id", func(t *testing.T) {
		assertRejected(t, "2528fbb91977201cf6d16b1b01ebf27eea342e85", "")
	})
}

func TestDescribePolyFetchFailure(t *testing.T) {
	t.Run("message names both upstream hosts that were tried", func(t *testing.T) {
		urls := []string{
			"https://raw.githubusercontent.com/comaps/comaps/2528fbb91977201cf6d16b1b01ebf27eea342e85/data/borders/Abkhazia.poly",
			"https://codeberg.org/comaps/comaps/raw/commit/2528fbb91977201cf6d16b1b01ebf27eea342e85/data/borders/Abkhazia.poly",
		}
		sentinel := errors.New("connection refused")
		err := describePolyFetchFailure(urls, sentinel)

		msg := err.Error()
		if !strings.Contains(msg, "raw.githubusercontent.com") {
			t.Errorf("failure message %q does not name the GitHub upstream", msg)
		}
		if !strings.Contains(msg, "codeberg.org") {
			t.Errorf("failure message %q does not name the Codeberg upstream", msg)
		}
	})

	t.Run("wrapped sentinel error is still found by errors.Is", func(t *testing.T) {
		sentinel := errors.New("connection refused")
		err := describePolyFetchFailure([]string{"https://raw.githubusercontent.com/x", "https://codeberg.org/x"}, sentinel)

		if !errors.Is(err, sentinel) {
			t.Fatalf("errors.Is(err, sentinel) = false, want true")
		}
	})
}
