package util

import (
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// These tests use only schema/record values. No App, database, fixture, HTTP
// client, filesystem or listener is constructed.
func photoIndexRecord(author, provider, file, identity string) *core.Record {
	collection := core.NewBaseCollection("assets")
	collection.Fields.Add(
		&core.TextField{Name: "author"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.TextField{Name: "file"},
		&core.JSONField{Name: "metadata"},
		&core.DateField{Name: "taken_at"},
	)
	record := core.NewRecord(collection)
	record.Set("author", author)
	record.Set("external_provider", provider)
	record.Set("external_id", identity)
	record.Set("file", file)
	return record
}

func photoIndexReference(assets []*core.Record, author string, photo FederatedPhoto) []int {
	var positions []int
	for i, asset := range assets {
		if FederatedPhotoMatchesAsset(asset, author, photo) {
			positions = append(positions, i)
		}
	}
	return positions
}

// candidates is only an all-position equivalence oracle for tests. Production
// FirstMatch needs just the minimum bucket head, not this full sorted union.
func (index *FederatedPhotoAssetIndex) candidates(photo FederatedPhoto) []int {
	var buckets [][]int
	for _, key := range federatedPhotoLookupKeys(photo) {
		if bucket := index.buckets[key]; len(bucket) != 0 {
			buckets = append(buckets, bucket)
		}
	}
	var positions []int
	for len(buckets) != 0 {
		first := buckets[0][0]
		for _, bucket := range buckets[1:] {
			first = min(first, bucket[0])
		}
		positions = append(positions, first)
		for i := len(buckets) - 1; i >= 0; i-- {
			if buckets[i][0] == first {
				buckets[i] = buckets[i][1:]
				if len(buckets[i]) == 0 {
					buckets = slices.Delete(buckets, i, i+1)
				}
			}
		}
	}
	return positions
}

func assertPhotoIndexEquivalent(t *testing.T, index *FederatedPhotoAssetIndex, assets []*core.Record, author string, photo FederatedPhoto) {
	t.Helper()
	want := photoIndexReference(assets, author, photo)
	if got := index.candidates(photo); !slices.Equal(got, want) {
		t.Fatalf("photo %+v: indexed positions %v, reference %v", photo, got, want)
	}
	position, asset := index.FirstMatch(photo)
	if len(want) == 0 {
		if position != -1 || asset != nil {
			t.Fatalf("unexpected first match: %d %v", position, asset)
		}
	} else if position != want[0] || asset != assets[want[0]] {
		t.Fatalf("first match %d %p, want %d %p", position, asset, want[0], assets[want[0]])
	}
}

func TestFederatedPhotoIndexLegacyNames(t *testing.T) {
	for _, tc := range []struct {
		source, saved string
		want          bool
	}{
		{"photo.jpg", "photo.jpg", true},
		{"photo.jpg", "photo_ab12cd34ef.jpg", true},
		{"photo", "photo_ab12cd34ef", true},
		{"photo.15+16.jpg", "photo_ab12cd34ef.15_16.jpg", true},
		{"photo.a=b.jpg", "photo_ab12cd34ef.a_b.jpg", true},
		{"photo.*.jpg", "photo_ab12cd34ef._.jpg", true},
		{"photo.a+=b.jpg", "photo_ab12cd34ef.a_b.jpg", true},
		{"photo..jpg", "photo_ab12cd34ef.jpg", true},
		{"photo.jpg.", "photo_ab12cd34ef.jpg", true},
		{"photo.a=b.", "photo_ab12cd34ef.a_b", true},
		{"photo.a=b.", "photo_ab12cd34ef.a=b", true},
		{"photo..", "photo_ab12cd34ef.txt", true},
		{"photo.", "photo_ab12cd34ef.a1b2c", true},
		{"photo..", "photo_ab12cd34ef.png", true},
		{"photo...", "photo._ab12cd34ef.jpg", true},
		{"a.b.c.jpg", "a.b_ab12cd34ef.c.jpg", true},
		{"with_under_score.jpg", "with_under_score_ab12cd34ef.jpg", true},
		{"photo.jpg", "photo_ab12cd34ef.png", false},
		{"photo..", "photo_ab12cd34ef.abcdef", false},
		{"photo..", "photo_ab12cd34ef.JPG", false},
		{"photo..", "photo_ab12cd34ef.jpg.png", false},
		{"photo..", "photo_!!!!!!!!!!.png", false},
		{"photo..", "other_ab12cd34ef.png", false},
		{"photo.a=b.jpg", "photo_ab12cd34ef.a_c.jpg", false},
		{"photo.a=b.jpg", "photo_!!!!!!!!!!.a_b.jpg", false},
		{"photo.jpg", "photo_ab12cd34eF.jpg", false},
		{"photo.jpg", "photo_ab12cd34e.jpg", false},
		{"photo.jpg", "photo_ab12cd34eff.jpg", false},
		{"photo..", "photo_ab12cd34ef", false},
		{"", "", false},
		{strings.Repeat("a", 101) + ".jpg", strings.Repeat("a", 100) + "_ab12cd34ef.jpg", true},
		{strings.Repeat("a", 101) + ".jpg", strings.Repeat("a", 101) + "_ab12cd34ef.jpg", false},
		// Byte 100 cuts a UTF-8 rune: do not replace the reference's byte slice
		// with rune truncation or normalize the resulting invalid UTF-8.
		{strings.Repeat("a", 99) + "雪.jpg", strings.Repeat("a", 99) + "\xe9_ab12cd34ef.jpg", true},
	} {
		t.Run(fmt.Sprintf("%q/%q", tc.source, tc.saved), func(t *testing.T) {
			asset := photoIndexRecord("author", "", tc.saved, "")
			photo := FederatedPhoto{SourceName: tc.source, CanonicalID: "https://remote.example/api/v1/assets/id"}
			if got := FederatedPhotoMatchesAsset(asset, "author", photo); got != tc.want {
				t.Fatalf("reference got %v, expected %v", got, tc.want)
			}
			assets := []*core.Record{asset}
			assertPhotoIndexEquivalent(t, NewFederatedPhotoAssetIndex(assets, "author"), assets, "author", photo)
		})
	}
	for _, source := range []string{"photo.jpg", "a/b.jpg", "a long name.jpg", "雪.jpg", "photo.???", "a." + strings.Repeat("e", 25), "\xff.jpg"} {
		asset := photoIndexRecord("author", "", FederatedSourceFileName(source), "")
		photo := FederatedPhoto{SourceName: source, CanonicalID: "%bad"}
		assets := []*core.Record{asset}
		index := NewFederatedPhotoAssetIndex(assets, "author")
		assertPhotoIndexEquivalent(t, index, assets, "author", photo)
		if _, got := index.FirstMatch(photo); got != asset {
			t.Fatalf("stable SHA name missing for %q", source)
		}
	}
}

func TestFederatedPhotoIndexURLAndEligibility(t *testing.T) {
	modern := "https://REMOTE.example/api/v1/assets/id"
	for _, tc := range []struct {
		name, provider, identity, canonical, source, author string
		want                                                bool
	}{
		{"legacy v1", "federation", "https://remote.example/api/v1/files/trails/id/photo.jpg", modern, "photo.jpg", "author", true},
		{"legacy pb", "federation", "https://remote.example/api/files/trails/id/photo.jpg", modern, "photo.jpg", "author", true},
		{"uppercase scheme", "federation", "HTTPS://remote.example/api/files/id/photo.jpg", modern, "photo.jpg", "author", true},
		{"different scheme", "federation", "http://remote.example/api/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"unicode fold sigma", "federation", "https://Σ.example/api/files/id/photo.jpg", "https://ς.example/api/v1/assets/id", "photo.jpg", "author", true},
		{"unicode fold kelvin", "federation", "https://K.example/api/files/id/photo.jpg", "https://k.example/api/v1/assets/id", "photo.jpg", "author", true},
		{"distinct unicode", "federation", "https://İ.example/api/files/id/photo.jpg", "https://i.example/api/v1/assets/id", "photo.jpg", "author", false},
		{"port not normalized", "federation", "https://remote.example:443/api/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"userinfo ignored", "federation", "https://user:pass@remote.example/api/files/id/photo.jpg", modern, "photo.jpg", "author", true},
		{"decoded path", "federation", "https://remote.example/api/files/id/snow%20photo.jpg?x#y", modern, "snow photo.jpg", "author", true},
		{"substring path", "federation", "https://remote.example/prefix/api/files/id/photo.jpg", modern, "photo.jpg", "author", true},
		{"case sensitive path", "federation", "https://remote.example/API/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"base after trailing slash", "federation", "https://remote.example/api/files/", modern, "files", "author", true},
		{"relative host", "federation", "/api/files/id/photo.jpg", "/api/v1/assets/id", "photo.jpg", "author", true},
		{"protocol relative", "federation", "//REMOTE.example/api/files/id/photo.jpg", "//remote.example/assets/id", "photo.jpg", "author", true},
		{"opaque", "federation", "https:remote.example/api/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"bad legacy URL", "federation", "https://remote.example/api/files/%zz", modern, "photo.jpg", "author", false},
		{"bad modern URL", "federation", "https://remote.example/api/files/id/photo.jpg", "https://remote.example/%zz", "photo.jpg", "author", false},
		{"invalid port", "federation", "https://remote.example:bad/api/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"wrong host", "federation", "https://other.example/api/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"modern not adoptable", "federation", modern, modern, "photo.jpg", "author", false},
		{"different provider", "import", "https://remote.example/api/files/id/photo.jpg", modern, "photo.jpg", "author", false},
		{"different author", "federation", "https://remote.example/api/files/id/photo.jpg", modern, "photo.jpg", "other-author", false},
		{"empty source", "federation", "https://remote.example/api/files/id/photo.jpg", modern, "", "author", false},
		{"file ignores invalid identity", "", "%bad", "%bad", "photo.jpg", "author", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset := photoIndexRecord(tc.author, tc.provider, "photo.jpg", tc.identity)
			photo := FederatedPhoto{SourceName: tc.source, CanonicalID: tc.canonical}
			if got := FederatedPhotoMatchesAsset(asset, "author", photo); got != tc.want {
				t.Fatalf("reference got %v, expected %v", got, tc.want)
			}
			assets := []*core.Record{asset}
			assertPhotoIndexEquivalent(t, NewFederatedPhotoAssetIndex(assets, "author"), assets, "author", photo)
		})
	}
	photo := FederatedPhoto{SourceName: "wanderer-route-preview.png", CanonicalID: modern}
	for _, tc := range []struct {
		name, provider, date string
		metadata             any
		want                 bool
	}{
		{"generated filename", "", "", nil, false},
		{"taken_at ordinary photo", "", "2026-01-01 00:00:00.000Z", nil, true},
		{"generated metadata", "", "", map[string]any{"generated": map[string]any{"kind": "route-preview"}}, false},
		{"malformed metadata", "", "2026-01-01 00:00:00.000Z", "not JSON", true},
		{"federation filename not generated", "federation", "", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset := photoIndexRecord("author", tc.provider, photo.SourceName, "https://remote.example/api/files/id/"+photo.SourceName)
			asset.Set("metadata", tc.metadata)
			asset.Set("taken_at", tc.date)
			assets := []*core.Record{asset}
			if got := FederatedPhotoMatchesAsset(asset, "author", photo); got != tc.want {
				t.Fatalf("reference got %v, expected %v", got, tc.want)
			}
			assertPhotoIndexEquivalent(t, NewFederatedPhotoAssetIndex(assets, "author"), assets, "author", photo)
		})
	}
}

func TestFederatedPhotoIndexFoldHost(t *testing.T) {
	hosts := []string{"EXAMPLE", "example", "Σ.example", "σ.example", "ς.example", "K.example", "K.example", "k.example", "İ.example", "i.example", "ı.example", "ſ.example", "s.example", "\xff", "\xfe", "�", "a:443", "A:443", "a"}
	for _, left := range hosts {
		for _, right := range hosts {
			if got := federatedPhotoFoldHost(left) == federatedPhotoFoldHost(right); got != strings.EqualFold(left, right) {
				t.Fatalf("host key differs from EqualFold for %q and %q", left, right)
			}
		}
	}
	if !strings.EqualFold("Σ.example", "ς.example") || strings.ToLower("Σ.example") == strings.ToLower("ς.example") {
		t.Fatal("Unicode lowercasing counterexample is not exercised")
	}
}

func TestFederatedPhotoIndexOrderAndRekey(t *testing.T) {
	source := strings.Repeat("a", 100) + "_ab12cd34ef.jpg"
	photo := FederatedPhoto{SourceName: source, CanonicalID: "https://remote.example/api/v1/assets/id"}
	assets := []*core.Record{
		photoIndexRecord("other-author", "", source, ""),
		photoIndexRecord("author", "federation", "", "https://remote.example/api/files/id/"+source),
		photoIndexRecord("author", "", FederatedSourceFileName(source), ""),
		photoIndexRecord("author", "", source, ""), // exact AND PB suffix match
		photoIndexRecord("author", "", source, ""),
	}
	index := NewFederatedPhotoAssetIndex(assets, "author")
	for repeat := 0; repeat < 10; repeat++ {
		assertPhotoIndexEquivalent(t, index, assets, "author", photo)
		if got := index.candidates(photo); !slices.Equal(got, []int{1, 2, 3, 4}) {
			t.Fatalf("unordered or duplicate candidates: %v", got)
		}
	}
	// Simulate adoption into an already managed, DIFFERENT record. The
	// earlier legacy copy must still win before that managed shortcut.
	managed := photoIndexRecord("author", "federation", source, photo.CanonicalID)
	managed.Id = "different-id"
	assets[1] = managed
	index.Replace(1, managed)
	assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	if position, _ := index.FirstMatch(photo); position != 2 {
		t.Fatalf("stale legacy URL key after merge: %d", position)
	}
	// In-place provider/identity/author changes must remove the old keys.
	assets[2].Set("external_provider", "federation")
	assets[2].Set("external_id", photo.CanonicalID)
	index.Replace(2, assets[2])
	assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	assets[3].Set("author", "local-independent-author")
	index.Replace(3, assets[3])
	assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	assets[4].Set("metadata", map[string]any{"generated": map[string]any{"kind": "route-preview"}})
	index.Replace(4, assets[4])
	assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	// Build only after author repair; repaired legacy URL is now eligible.
	assets[0].Set("author", "author")
	index.Replace(0, assets[0])
	assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	rebuilt := NewFederatedPhotoAssetIndex(assets, "author")
	if !reflect.DeepEqual(index.buckets, rebuilt.buckets) {
		t.Fatal("rekeyed buckets differ from rebuilt corrected state")
	}
}

func TestFederatedPhotoIndexAmbiguousFirstMatch(t *testing.T) {
	const size = 1000
	photo := FederatedPhoto{SourceName: "photo.jpg", CanonicalID: "https://remote.example/api/v1/assets/id"}
	assets := make([]*core.Record, size)
	for i := range assets {
		assets[i] = photoIndexRecord("author", "", photo.SourceName, "")
	}
	index := NewFederatedPhotoAssetIndex(assets, "author")
	before := make(map[federatedPhotoMatchKey][]int, len(index.buckets))
	for key, bucket := range index.buckets {
		before[key] = slices.Clone(bucket)
	}
	for i := 0; i < size; i++ {
		// An unchanged tuple can still be a different adopted record. Always
		// refresh the record pointer, without shifting ambiguous buckets.
		replacement := photoIndexRecord("author", "", photo.SourceName, "")
		replacement.Id = fmt.Sprintf("replacement-%d", i)
		assets[i] = replacement
		index.Replace(i, replacement)
		if index.assets[i] != replacement {
			t.Fatal("unchanged keys retained a stale record")
		}
		position, asset := index.FirstMatch(photo)
		if position != 0 || asset != assets[0] {
			t.Fatalf("ambiguous lookup %d: first %d %p, want 0 %p", i, position, asset, assets[0])
		}
	}
	if !reflect.DeepEqual(index.buckets, before) {
		t.Fatal("unchanged tuple replacements altered bucket order/membership")
	}
	assertPhotoIndexEquivalent(t, index, assets, "author", photo)

	// Equal empty key sets must refresh ineligible record pointers too.
	managed := photoIndexRecord("author", "federation", "", photo.CanonicalID)
	index = NewFederatedPhotoAssetIndex([]*core.Record{managed}, "author")
	replacement := photoIndexRecord("author", "federation", "", photo.CanonicalID)
	replacement.Id = "different-managed-id"
	index.Replace(0, replacement)
	if index.assets[0] != replacement || len(index.buckets) != 0 {
		t.Fatal("unchanged empty keys failed to refresh the record")
	}
}

func TestFederatedPhotoIndexRandomizedEquivalence(t *testing.T) {
	random := rand.New(rand.NewSource(20261127))
	names := []string{"", "photo.jpg", "photo..", "a.b.c.jpg", "photo.a=b.", "photo.15+16.jpg", "雪.jpg", strings.Repeat("a", 101) + ".jpg", "\xff.jpg"}
	identities := []string{"https://remote.example/api/files/id/photo.jpg", "https://Σ.example/api/v1/files/id/photo.jpg", "https://remote.example/api/v1/assets/id", "/api/files/id/photo..", "%bad", ""}
	var assets []*core.Record
	for i := 0; i < 150; i++ {
		source := names[random.Intn(len(names))]
		file := source
		switch i % 4 {
		case 1:
			file = FederatedSourceFileName(source)
		case 2:
			file = strings.TrimSuffix(source, ".jpg") + "_ab12cd34ef.jpg"
		case 3:
			file = "photo_ab12cd34ef.png"
		}
		assets = append(assets, photoIndexRecord([]string{"author", "other"}[i%2], []string{"", "federation", "import"}[i%3], file, identities[i%len(identities)]))
	}
	index := NewFederatedPhotoAssetIndex(assets, "author")
	for iteration := 0; iteration < 300; iteration++ {
		if iteration%3 == 0 {
			position := random.Intn(len(assets))
			asset := assets[position]
			asset.Set("author", []string{"author", "other"}[random.Intn(2)])
			asset.Set("external_provider", []string{"", "federation", "import"}[random.Intn(3)])
			asset.Set("external_id", identities[random.Intn(len(identities))])
			asset.Set("metadata", map[string]any{"generated": map[string]any{"kind": []string{"", "route-preview"}[random.Intn(2)]}})
			index.Replace(position, asset)
		}
		photo := FederatedPhoto{SourceName: names[random.Intn(len(names))], CanonicalID: identities[random.Intn(len(identities))]}
		assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	}
}

func TestFederatedPhotoIndexLinearCandidateCounts(t *testing.T) {
	const size = 1000
	for _, kind := range []string{"modern", "exact", "hashed", "suffix", "legacy-url", "legacy-miss"} {
		t.Run(kind, func(t *testing.T) {
			assets := make([]*core.Record, size)
			photos := make([]FederatedPhoto, size)
			for i := range assets {
				source := fmt.Sprintf("photo-%04d.jpg", i)
				provider, file, identity := "", source, ""
				switch kind {
				case "modern":
					provider, identity = "federation", fmt.Sprintf("https://remote.example/api/v1/assets/%d", i)
				case "hashed":
					file = FederatedSourceFileName(source)
				case "suffix":
					file = fmt.Sprintf("photo-%04d_ab12cd34ef.jpg", i)
				case "legacy-url":
					provider, identity = "federation", "https://remote.example/api/files/id/"+source
				case "legacy-miss":
					file = "unrelated-" + source
				}
				assets[i] = photoIndexRecord("author", provider, file, identity)
				photos[i] = FederatedPhoto{SourceName: source, CanonicalID: fmt.Sprintf("https://remote.example/api/v1/assets/%d", i)}
			}
			index := NewFederatedPhotoAssetIndex(assets, "author")
			referenceCalls, candidateCount := 0, 0
			for _, photo := range photos {
				want := photoIndexReference(assets, "author", photo)
				referenceCalls += len(assets)
				got := index.candidates(photo)
				candidateCount += len(got)
				if !slices.Equal(got, want) {
					t.Fatalf("candidate mismatch: %v != %v", got, want)
				}
			}
			wantCandidates := size
			if kind == "modern" || kind == "legacy-miss" {
				wantCandidates = 0
			}
			if referenceCalls != size*size || candidateCount != wantCandidates {
				t.Fatalf("reference calls=%d, candidates=%d, want candidates=%d", referenceCalls, candidateCount, wantCandidates)
			}
			t.Logf("E=%d P=%d brute-force predicates=%d indexed candidate positions=%d asset preparations=%d photo preparations=%d", size, size, referenceCalls, candidateCount, len(index.keys), len(photos))
		})
	}
}

func FuzzFederatedPhotoIndexReference(f *testing.F) {
	for _, seed := range [][5]string{
		{"photo..", "photo_ab12cd34ef.png", "", "", "https://example/api/v1/assets/id"},
		{"photo.jpg", "", "federation", "https://Σ.example/api/files/id/photo.jpg", "https://ς.example/api/v1/assets/id"},
		{"photo.a=b.", "photo_ab12cd34ef.a_b", "", "", "%bad"},
		{strings.Repeat("a", 99) + "雪.jpg", strings.Repeat("a", 99) + "\xe9_ab12cd34ef.jpg", "", "", ""},
	} {
		f.Add(seed[0], seed[1], seed[2], seed[3], seed[4])
	}
	f.Fuzz(func(t *testing.T, source, saved, provider, legacy, modern string) {
		assets := []*core.Record{
			photoIndexRecord("author", provider, saved, legacy),
			photoIndexRecord("author", "", source, ""),
			photoIndexRecord("author", "", FederatedSourceFileName(source), ""),
			photoIndexRecord("other", "", saved, legacy),
			photoIndexRecord("author", "federation", saved, legacy),
		}
		photo := FederatedPhoto{SourceName: source, CanonicalID: modern}
		index := NewFederatedPhotoAssetIndex(assets, "author")
		assertPhotoIndexEquivalent(t, index, assets, "author", photo)
		assets[0].Set("external_provider", "federation")
		assets[0].Set("external_id", modern)
		index.Replace(0, assets[0])
		assertPhotoIndexEquivalent(t, index, assets, "author", photo)
	})
}
