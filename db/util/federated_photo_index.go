package util

import (
	"net/url"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/pocketbase/pocketbase/core"
)

// FederatedPhotoAssetIndex is an operation-local index of the legacy copies
// recognized by FederatedPhotoMatchesAsset. Build it after author corrections;
// Replace must be called whenever adoption replaces or mutates a linked asset.
// Positions, not record IDs, preserve the original link order.
type FederatedPhotoAssetIndex struct {
	author  string
	assets  []*core.Record
	keys    [][]federatedPhotoMatchKey
	buckets map[federatedPhotoMatchKey][]int
}

type federatedPhotoMatchKey struct {
	kind               byte
	name, ext          string
	scheme, foldedHost string
}

const (
	federatedPhotoExactName byte = iota
	federatedPhotoCopyName
	federatedPhotoDetectedName
	federatedPhotoLegacyURL
)

func NewFederatedPhotoAssetIndex(assets []*core.Record, author string) *FederatedPhotoAssetIndex {
	index := &FederatedPhotoAssetIndex{
		author: author, assets: slices.Clone(assets),
		keys:    make([][]federatedPhotoMatchKey, len(assets)),
		buckets: make(map[federatedPhotoMatchKey][]int),
	}
	for i, asset := range assets {
		index.Replace(i, asset)
	}
	return index
}

// Replace always refreshes the record, including a different canonical ID.
// It removes stored old keys only when the prepared matching tuple changes.
func (index *FederatedPhotoAssetIndex) Replace(position int, asset *core.Record) {
	keys := federatedPhotoAssetKeys(asset, index.author)
	index.assets[position] = asset
	if slices.Equal(index.keys[position], keys) {
		return
	}
	for _, key := range index.keys[position] {
		bucket := index.buckets[key]
		if i, found := slices.BinarySearch(bucket, position); found {
			bucket = slices.Delete(bucket, i, i+1)
		}
		if len(bucket) == 0 {
			delete(index.buckets, key)
		} else {
			index.buckets[key] = bucket
		}
	}
	index.keys[position] = keys
	for _, key := range index.keys[position] {
		bucket := index.buckets[key]
		if len(bucket) == 0 || bucket[len(bucket)-1] < position {
			index.buckets[key] = append(bucket, position)
			continue
		}
		i, _ := slices.BinarySearch(bucket, position)
		index.buckets[key] = slices.Insert(bucket, i, position)
	}
}

// FirstMatch returns the same first position as a scan of the reference
// predicate, or (-1, nil). No asset metadata, URL or filename is reparsed here.
func (index *FederatedPhotoAssetIndex) FirstMatch(photo FederatedPhoto) (int, *core.Record) {
	// The minimum of the sorted union is the minimum bucket head. Never
	// traverse/materialize all members of an ambiguous matching bucket.
	first := -1
	for _, key := range federatedPhotoLookupKeys(photo) {
		bucket := index.buckets[key]
		if len(bucket) == 0 {
			continue
		}
		if first < 0 || bucket[0] < first {
			first = bucket[0]
		}
	}
	if first < 0 {
		return -1, nil
	}
	return first, index.assets[first]
}

func federatedPhotoAssetKeys(asset *core.Record, author string) []federatedPhotoMatchKey {
	if asset.GetString("author") != author || IsGeneratedRoutePreviewAsset(asset) {
		return nil
	}
	switch asset.GetString("external_provider") {
	case "":
		name := asset.GetString("file")
		keys := []federatedPhotoMatchKey{{kind: federatedPhotoExactName, name: name}}
		// A reference match has a <=100-BYTE base followed by "_" and
		// exactly ten lowercase ASCII letters/digits. Consider every possible
		// delimiter, including bases containing underscores or earlier dots.
		for i := 0; i <= 100 && i+11 <= len(name); i++ {
			if name[i] != '_' || !remoteCopyNameSuffix.MatchString(name[i+1:i+11]) {
				continue
			}
			ext := name[i+11:]
			if ext != "" && !strings.HasPrefix(ext, ".") {
				continue
			}
			keys = append(keys, federatedPhotoMatchKey{kind: federatedPhotoCopyName, name: name[:i], ext: ext})
			if remoteCopyDetectedExt.MatchString(ext) {
				keys = append(keys, federatedPhotoMatchKey{kind: federatedPhotoDetectedName, name: name[:i]})
			}
		}
		return keys
	case FederationAssetProvider:
		legacy, err := url.Parse(asset.GetString("external_id"))
		if err != nil || !strings.Contains(legacy.Path, "/api/v1/files/") && !strings.Contains(legacy.Path, "/api/files/") {
			return nil
		}
		return []federatedPhotoMatchKey{{
			kind: federatedPhotoLegacyURL, name: path.Base(legacy.Path),
			scheme: legacy.Scheme, foldedHost: federatedPhotoFoldHost(legacy.Host),
		}}
	default:
		return nil
	}
}

func federatedPhotoLookupKeys(photo FederatedPhoto) []federatedPhotoMatchKey {
	source := photo.SourceName
	if source == "" {
		return nil
	}
	keys := []federatedPhotoMatchKey{{kind: federatedPhotoExactName, name: source}}
	stable := FederatedSourceFileName(source)
	if stable != source {
		keys = append(keys, federatedPhotoMatchKey{kind: federatedPhotoExactName, name: stable})
	}
	// Keep the reference's last TWO extension components and byte truncation.
	ext := ""
	if dot := strings.LastIndex(source, "."); dot >= 0 {
		if previous := strings.LastIndex(source[:dot], "."); previous >= 0 {
			dot = previous
		}
		ext = source[dot:]
	}
	base := strings.TrimSuffix(source, ext)
	if len(base) > 100 {
		base = base[:100]
	}
	normalized := remoteCopyExtInvalid.ReplaceAllString(ext, "_")
	for _, candidate := range []string{ext, normalized, "." + strings.Trim(ext, "."), "." + strings.Trim(normalized, ".")} {
		key := federatedPhotoMatchKey{kind: federatedPhotoCopyName, name: base, ext: candidate}
		if (candidate == ext || candidate != ".") && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	if ext != "" && strings.Trim(ext, ".") == "" {
		keys = append(keys, federatedPhotoMatchKey{kind: federatedPhotoDetectedName, name: base})
	}
	if modern, err := url.Parse(photo.CanonicalID); err == nil {
		keys = append(keys, federatedPhotoMatchKey{
			kind: federatedPhotoLegacyURL, name: source,
			scheme: modern.Scheme, foldedHost: federatedPhotoFoldHost(modern.Host),
		})
	}
	return keys
}

// EqualFold uses Unicode simple folding, not ToLower (notably for Σ/ς and
// K/K). The smallest rune in each SimpleFold cycle is an exact equivalence key.
func federatedPhotoFoldHost(host string) string {
	return strings.Map(func(r rune) rune {
		smallest := r
		for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
			smallest = min(smallest, folded)
		}
		return smallest
	}, host)
}
