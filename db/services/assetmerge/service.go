package assetmerge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	assetservice "pocketbase/services/assets"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const (
	reasonContentHash       = "content_hash"
	reasonExternalReference = "external_reference"
	reasonLegacySourceFile  = "legacy_source_file"
	reasonTimeLocation      = "time_location"

	targetReasonMostLinks             = "most_links"
	targetReasonLocalCopy             = "local_copy"
	targetReasonMostCompleteMetadata  = "most_complete_metadata"
	targetReasonOldestAsset           = "oldest_asset"
	targetReasonDeterministicFallback = "deterministic_fallback"
)

var (
	ErrMissingUserID             = errors.New("asset_merge_missing_user_id")
	ErrMissingTargetAssetID      = errors.New("asset_merge_missing_target_asset_id")
	ErrMissingSourceAssetID      = errors.New("asset_merge_missing_source_asset_id")
	ErrSameSourceAndTargetAsset  = errors.New("asset_merge_same_source_target")
	ErrRequiresMultipleAssets    = errors.New("asset_merge_requires_multiple_assets")
	ErrTargetAssetNotFound       = errors.New("asset_merge_target_not_found")
	ErrSourceAssetNotFound       = errors.New("asset_merge_source_not_found")
	ErrTargetAssetAuthorMismatch = errors.New("asset_merge_target_author_mismatch")
	ErrSourceAssetAuthorMismatch = errors.New("asset_merge_source_author_mismatch")
)

type LinkCounts struct {
	Trails     int `json:"trails"`
	Waypoints  int `json:"waypoints"`
	SummitLogs int `json:"summitLogs"`
	Total      int `json:"total"`
}

type AssetSummary struct {
	ID               string     `json:"id"`
	CollectionID     string     `json:"collectionId"`
	CollectionName   string     `json:"collectionName"`
	Created          string     `json:"created,omitempty"`
	Updated          string     `json:"updated,omitempty"`
	Type             string     `json:"type"`
	File             string     `json:"file,omitempty"`
	StorageMode      string     `json:"storageMode,omitempty"`
	RemoteStatus     string     `json:"remoteStatus,omitempty"`
	ExternalProvider string     `json:"externalProvider,omitempty"`
	ExternalID       string     `json:"externalId,omitempty"`
	TakenAt          string     `json:"takenAt,omitempty"`
	Lat              *float64   `json:"lat,omitempty"`
	Lon              *float64   `json:"lon,omitempty"`
	OriginalFileName string     `json:"originalFileName"`
	ThumbnailURL     string     `json:"thumbnailUrl"`
	Links            LinkCounts `json:"links"`
}

type SuggestGroup struct {
	GroupID       string         `json:"groupId"`
	AssetIDs      []string       `json:"assetIds"`
	TargetAssetID string         `json:"targetAssetId"`
	Reason        string         `json:"reason"`
	MatchReason   string         `json:"matchReason"`
	Score         float64        `json:"score"`
	Assets        []AssetSummary `json:"assets"`
}

type SuggestResponse struct {
	Groups []SuggestGroup `json:"groups"`
}

type MergeResponse struct {
	Acknowledged    bool     `json:"acknowledged"`
	TargetAssetID   string   `json:"targetAssetId"`
	MergedAssetIDs  []string `json:"mergedAssetIds"`
	DeletedAssetIDs []string `json:"deletedAssetIds"`
	ReassignedLinks int      `json:"reassignedLinks"`
}

type assetInfo struct {
	record      *core.Record
	metadata    map[string]any
	links       LinkCounts
	contentHash string
	takenAt     time.Time
	hasTakenAt  bool
	lat         float64
	lon         float64
	hasLat      bool
	hasLon      bool
}

type matchReason struct {
	reason   string
	priority int
}

type linkCollection struct {
	name  string
	field string
}

var assetLinkCollections = []linkCollection{
	{name: "trail_assets", field: "trail"},
	{name: "waypoint_assets", field: "waypoint"},
	{name: "summit_log_assets", field: "summit_log"},
}

// SuggestGroups returns groups of photo assets that are likely duplicates.
// Local copied assets are matched by file hash; remote/private assets use
// stable external ids, legacy migration metadata, and exact time/location keys.
func SuggestGroups(app core.App, actorID string) (*SuggestResponse, error) {
	if strings.TrimSpace(actorID) == "" {
		return nil, ErrMissingUserID
	}

	records, err := app.FindRecordsByFilter(
		"assets",
		"author={:actor} && type='photo'",
		"",
		-1,
		0,
		dbx.Params{"actor": actorID},
	)
	if err != nil {
		return nil, err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	linkCountsByID, err := linkCountsByAssetID(app, records)
	if err != nil {
		return nil, err
	}

	infos := make(map[string]*assetInfo, len(records))
	groupsByKey := map[string][]string{}
	reasonByKey := map[string]matchReason{}
	ds := newDisjointSet()

	addKey := func(key string, assetID string, reason matchReason) {
		if key == "" || assetID == "" {
			return
		}
		groupsByKey[key] = append(groupsByKey[key], assetID)
		reasonByKey[key] = reason
	}

	for _, record := range records {
		if isGeneratedRoutePreviewAsset(record) {
			continue
		}
		info, err := buildAssetInfo(app, fsys, record, linkCountsByID[record.Id])
		if err != nil {
			return nil, err
		}
		infos[record.Id] = info
		ds.add(record.Id)

		if info.contentHash != "" {
			addKey("content:"+info.contentHash, record.Id, matchReason{reason: reasonContentHash, priority: 100})
		}
		if provider := record.GetString("external_provider"); provider != "" {
			if externalID := record.GetString("external_id"); externalID != "" {
				addKey("external:"+provider+"\x00"+externalID, record.Id, matchReason{reason: reasonExternalReference, priority: 90})
			}
		}
		if legacyKey := legacySourceFileKey(info); legacyKey != "" {
			addKey("legacy:"+legacyKey, record.Id, matchReason{reason: reasonLegacySourceFile, priority: 70})
		}
		if timeLocationKey := timeLocationKey(info); timeLocationKey != "" {
			addKey("time_location:"+timeLocationKey, record.Id, matchReason{reason: reasonTimeLocation, priority: 60})
		}
	}

	for _, assetIDs := range groupsByKey {
		assetIDs = util.UniqueNonEmptyStrings(assetIDs)
		if len(assetIDs) < 2 {
			continue
		}
		first := assetIDs[0]
		for _, assetID := range assetIDs[1:] {
			ds.union(first, assetID)
		}
	}

	componentReason := map[string]matchReason{}
	for key, assetIDs := range groupsByKey {
		if len(util.UniqueNonEmptyStrings(assetIDs)) < 2 {
			continue
		}
		root := ds.find(assetIDs[0])
		reason := reasonByKey[key]
		if existing, ok := componentReason[root]; !ok || reason.priority > existing.priority {
			componentReason[root] = reason
		}
	}

	components := map[string][]*assetInfo{}
	for assetID, info := range infos {
		root := ds.find(assetID)
		components[root] = append(components[root], info)
	}

	groups := make([]SuggestGroup, 0, len(components))
	for root, component := range components {
		if len(component) < 2 {
			continue
		}
		sortAssetInfos(component)
		target := chooseTargetAsset(component)
		assets := make([]AssetSummary, 0, len(component))
		assetIDs := make([]string, 0, len(component))
		for _, info := range component {
			assets = append(assets, summarizeAsset(info))
			assetIDs = append(assetIDs, info.record.Id)
		}
		reason := componentReason[root]
		groups = append(groups, SuggestGroup{
			GroupID:       root,
			AssetIDs:      assetIDs,
			TargetAssetID: target.record.Id,
			Reason:        targetReason(component, target),
			MatchReason:   reason.reason,
			Score:         float64(reason.priority) + float64(len(component))/100,
			Assets:        assets,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Score != groups[j].Score {
			return groups[i].Score > groups[j].Score
		}
		if len(groups[i].Assets) != len(groups[j].Assets) {
			return len(groups[i].Assets) > len(groups[j].Assets)
		}
		return groups[i].GroupID < groups[j].GroupID
	})

	return &SuggestResponse{Groups: groups}, nil
}

// Merge reassigns all links from the source assets to the target asset and
// removes the now-duplicate source asset records.
func Merge(app core.App, actorID string, targetAssetID string, sourceAssetIDs []string) (*MergeResponse, error) {
	return MergeWithContext(context.Background(), app, actorID, targetAssetID, sourceAssetIDs)
}

func MergeWithContext(ctx context.Context, app core.App, actorID string, targetAssetID string, sourceAssetIDs []string) (*MergeResponse, error) {
	if strings.TrimSpace(actorID) == "" {
		return nil, ErrMissingUserID
	}
	if strings.TrimSpace(targetAssetID) == "" {
		return nil, ErrMissingTargetAssetID
	}
	sourceAssetIDs = util.UniqueNonEmptyStrings(sourceAssetIDs)
	if len(sourceAssetIDs) == 0 {
		return nil, ErrMissingSourceAssetID
	}
	if len(sourceAssetIDs) == 1 && sourceAssetIDs[0] == targetAssetID {
		return nil, ErrRequiresMultipleAssets
	}

	response := &MergeResponse{
		Acknowledged:    true,
		TargetAssetID:   targetAssetID,
		MergedAssetIDs:  make([]string, 0, len(sourceAssetIDs)),
		DeletedAssetIDs: make([]string, 0, len(sourceAssetIDs)),
	}

	err := app.RunInTransaction(func(txApp core.App) error {
		target, err := txApp.FindRecordById("assets", targetAssetID)
		if err != nil {
			return ErrTargetAssetNotFound
		}
		if target.GetString("author") != actorID {
			return ErrTargetAssetAuthorMismatch
		}

		for _, sourceAssetID := range sourceAssetIDs {
			if sourceAssetID == targetAssetID {
				return ErrSameSourceAndTargetAsset
			}
			source, err := txApp.FindRecordById("assets", sourceAssetID)
			if err != nil {
				return ErrSourceAssetNotFound
			}
			if source.GetString("author") != actorID {
				return ErrSourceAssetAuthorMismatch
			}

			reassigned, err := reassignAssetLinks(ctx, txApp, sourceAssetID, targetAssetID)
			if err != nil {
				return err
			}
			response.ReassignedLinks += reassigned

			metadataChanged, err := mergeSourceAssetMetadata(target, source)
			if err != nil {
				return err
			}

			if err := txApp.Delete(source); err != nil {
				return err
			}
			if metadataChanged {
				if err := txApp.Save(target); err != nil {
					return err
				}
			}
			response.MergedAssetIDs = append(response.MergedAssetIDs, sourceAssetID)
			response.DeletedAssetIDs = append(response.DeletedAssetIDs, sourceAssetID)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

func mergeSourceAssetMetadata(target *core.Record, source *core.Record) (bool, error) {
	changed := false

	if target.GetDateTime("taken_at").IsZero() {
		if taken := source.GetDateTime("taken_at"); !taken.IsZero() {
			target.Set("taken_at", taken.Time())
			changed = true
		}
	}

	sourceLat := optionalFloat(source, "lat")
	sourceLon := optionalFloat(source, "lon")
	if sourceLat != nil && sourceLon != nil && (optionalFloat(target, "lat") == nil || optionalFloat(target, "lon") == nil) {
		target.Set("lat", *sourceLat)
		target.Set("lon", *sourceLon)
		changed = true
	}

	if target.GetString("external_provider") == "" && source.GetString("external_provider") != "" {
		target.Set("external_provider", source.GetString("external_provider"))
		changed = true
	}
	if target.GetString("external_id") == "" && source.GetString("external_id") != "" {
		target.Set("external_id", source.GetString("external_id"))
		changed = true
	}

	targetMetadata, err := metadataMap(target)
	if err != nil {
		return false, err
	}
	sourceMetadata, err := metadataMap(source)
	if err != nil {
		return false, err
	}

	for key, value := range sourceMetadata {
		if _, exists := targetMetadata[key]; !exists {
			targetMetadata[key] = value
		}
	}
	targetMetadata["merged_assets"] = appendMergedAssetSnapshot(
		targetMetadata["merged_assets"],
		sourceAssetMetadataSnapshot(source, sourceMetadata),
	)
	target.Set("metadata", targetMetadata)
	changed = true

	return changed, nil
}

func appendMergedAssetSnapshot(raw any, snapshot map[string]any) []any {
	if raw == nil {
		return []any{snapshot}
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return []any{raw, snapshot}
	}
	var existing []any
	if err := json.Unmarshal(rawBytes, &existing); err != nil {
		return []any{raw, snapshot}
	}
	return append(existing, snapshot)
}

func sourceAssetMetadataSnapshot(source *core.Record, metadata map[string]any) map[string]any {
	snapshot := map[string]any{
		"asset_id": source.Id,
	}
	for _, field := range []string{
		"type",
		"file",
		"storage_mode",
		"remote_status",
		"external_provider",
		"external_id",
	} {
		if value := strings.TrimSpace(source.GetString(field)); value != "" {
			snapshot[field] = value
		}
	}
	if taken := source.GetDateTime("taken_at"); !taken.IsZero() {
		snapshot["taken_at"] = taken.Time().Format(time.RFC3339)
	}
	if lat := optionalFloat(source, "lat"); lat != nil {
		snapshot["lat"] = *lat
	}
	if lon := optionalFloat(source, "lon"); lon != nil {
		snapshot["lon"] = *lon
	}
	if len(metadata) > 0 {
		snapshot["metadata"] = metadata
	}
	return snapshot
}

func buildAssetInfo(app core.App, fsys *filesystem.System, record *core.Record, links LinkCounts) (*assetInfo, error) {
	metadata, err := metadataMap(record)
	if err != nil {
		return nil, err
	}
	info := &assetInfo{
		record:   record,
		metadata: metadata,
		links:    links,
	}
	if hash := metadataContentHash(metadata); hash != "" {
		info.contentHash = hash
	} else if hash, ok := recordContentHash(fsys, record); ok {
		info.contentHash = hash
		if err := persistAssetContentHash(app, record, hash); err != nil {
			app.Logger().Warn("failed to persist asset content hash", "asset", record.Id, "error", err)
		}
	}
	if taken := record.GetDateTime("taken_at"); !taken.IsZero() {
		info.takenAt = taken.Time()
		info.hasTakenAt = true
	}
	if lat := optionalFloat(record, "lat"); lat != nil {
		info.lat = *lat
		info.hasLat = true
	}
	if lon := optionalFloat(record, "lon"); lon != nil {
		info.lon = *lon
		info.hasLon = true
	}
	return info, nil
}

func persistAssetContentHash(app core.App, record *core.Record, hash string) error {
	current, err := app.FindRecordById("assets", record.Id)
	if err != nil {
		return err
	}
	if current.GetString("file") != record.GetString("file") {
		return nil
	}
	metadata, err := metadataMap(current)
	if err != nil {
		return err
	}
	metadata["content_hash"] = hash
	current.Set("metadata", metadata)
	current.IgnoreUnchangedFields(true)
	return app.UnsafeWithoutHooks().Save(current)
}

func reassignAssetLinks(ctx context.Context, app core.App, sourceAssetID string, targetAssetID string) (int, error) {
	reassigned := 0
	for _, collection := range assetLinkCollections {
		links, err := app.FindRecordsByFilter(
			collection.name,
			"asset={:asset}",
			"",
			-1,
			0,
			dbx.Params{"asset": sourceAssetID},
		)
		if err != nil {
			return reassigned, err
		}
		for _, link := range links {
			targetID := link.GetString(collection.field)
			if _, err := assetservice.EnsurePublicTrailSafeAssetLink(ctx, app, collection.name, collection.field, targetID, targetAssetID); err != nil {
				return reassigned, err
			}
			if err := app.Delete(link); err != nil {
				return reassigned, err
			}
			reassigned++
		}
	}
	return reassigned, nil
}

func linkCountsByAssetID(app core.App, records []*core.Record) (map[string]LinkCounts, error) {
	counts := make(map[string]LinkCounts, len(records))
	if len(records) == 0 {
		return counts, nil
	}

	assetIDs := make([]any, 0, len(records))
	for _, record := range records {
		assetIDs = append(assetIDs, record.Id)
		counts[record.Id] = LinkCounts{}
	}

	type linkCountRow struct {
		Asset string `db:"asset"`
		Count int    `db:"count"`
	}

	for _, collection := range assetLinkCollections {
		rows := []linkCountRow{}
		err := app.DB().
			Select("asset", "COUNT(*) AS count").
			From(collection.name).
			Where(dbx.In("asset", assetIDs...)).
			GroupBy("asset").
			All(&rows)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			linkCounts := counts[row.Asset]
			switch collection.name {
			case "trail_assets":
				linkCounts.Trails = row.Count
			case "waypoint_assets":
				linkCounts.Waypoints = row.Count
			case "summit_log_assets":
				linkCounts.SummitLogs = row.Count
			}
			linkCounts.Total += row.Count
			counts[row.Asset] = linkCounts
		}
	}

	return counts, nil
}

func summarizeAsset(info *assetInfo) AssetSummary {
	record := info.record
	collectionName := ""
	collectionID := ""
	if collection := record.Collection(); collection != nil {
		collectionName = collection.Name
		collectionID = collection.Id
	}
	return AssetSummary{
		ID:               record.Id,
		CollectionID:     collectionID,
		CollectionName:   collectionName,
		Created:          util.RecordDateTimeRFC3339(record, "created"),
		Updated:          util.RecordDateTimeRFC3339(record, "updated"),
		Type:             record.GetString("type"),
		File:             record.GetString("file"),
		StorageMode:      record.GetString("storage_mode"),
		RemoteStatus:     record.GetString("remote_status"),
		ExternalProvider: record.GetString("external_provider"),
		ExternalID:       record.GetString("external_id"),
		TakenAt:          util.RecordDateTimeRFC3339(record, "taken_at"),
		Lat:              optionalFloat(record, "lat"),
		Lon:              optionalFloat(record, "lon"),
		OriginalFileName: util.AssetFilename(record, info.metadata),
		ThumbnailURL:     util.AssetPublicMediaURL(record, ""),
		Links:            info.links,
	}
}

func sortAssetInfos(infos []*assetInfo) {
	sort.Slice(infos, func(i, j int) bool {
		return betterTarget(infos[i], infos[j])
	})
}

func chooseTargetAsset(infos []*assetInfo) *assetInfo {
	candidates := append([]*assetInfo(nil), infos...)
	sort.Slice(candidates, func(i, j int) bool {
		return betterTarget(candidates[i], candidates[j])
	})
	return candidates[0]
}

func betterTarget(a, b *assetInfo) bool {
	if hasLocalCopy(a.record) != hasLocalCopy(b.record) {
		return hasLocalCopy(a.record)
	}
	if a.links.Total != b.links.Total {
		return a.links.Total > b.links.Total
	}
	if metadataCompleteness(a) != metadataCompleteness(b) {
		return metadataCompleteness(a) > metadataCompleteness(b)
	}
	aCreated := a.record.GetDateTime("created").Time()
	bCreated := b.record.GetDateTime("created").Time()
	if !aCreated.Equal(bCreated) {
		return aCreated.Before(bCreated)
	}
	return a.record.Id < b.record.Id
}

func targetReason(component []*assetInfo, target *assetInfo) string {
	for _, info := range component {
		if info.record.Id == target.record.Id {
			continue
		}
		if hasLocalCopy(target.record) && !hasLocalCopy(info.record) {
			return targetReasonLocalCopy
		}
	}
	for _, info := range component {
		if info.record.Id == target.record.Id {
			continue
		}
		if target.links.Total > info.links.Total {
			return targetReasonMostLinks
		}
	}
	for _, info := range component {
		if info.record.Id == target.record.Id {
			continue
		}
		if metadataCompleteness(target) > metadataCompleteness(info) {
			return targetReasonMostCompleteMetadata
		}
	}
	for _, info := range component {
		if info.record.Id == target.record.Id {
			continue
		}
		if target.record.GetDateTime("created").Time().Before(info.record.GetDateTime("created").Time()) {
			return targetReasonOldestAsset
		}
	}
	return targetReasonDeterministicFallback
}

func metadataCompleteness(info *assetInfo) int {
	score := 0
	if info.record.GetString("external_provider") != "" && info.record.GetString("external_id") != "" {
		score += 4
	}
	if info.hasTakenAt {
		score += 2
	}
	if info.hasLat && info.hasLon {
		score += 2
	}
	if info.record.GetString("remote_status") == "available" {
		score++
	}
	return score
}

func hasLocalCopy(record *core.Record) bool {
	return record.GetString("file") != "" && (record.GetString("storage_mode") == "" || record.GetString("storage_mode") == "copy")
}

func metadataContentHash(metadata map[string]any) string {
	hash := strings.TrimSpace(util.AssetMetadataString(metadata, "content_hash"))
	if len(hash) != sha256.Size*2 {
		return ""
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return ""
	}
	return strings.ToLower(hash)
}

func recordContentHash(fsys *filesystem.System, record *core.Record) (string, bool) {
	fileName := record.GetString("file")
	if fileName == "" {
		return "", false
	}
	reader, err := fsys.GetReader(record.BaseFilesPath() + "/" + fileName)
	if err != nil {
		return "", false
	}
	defer reader.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", false
	}
	return hex.EncodeToString(hash.Sum(nil)), true
}

func legacySourceFileKey(info *assetInfo) string {
	sourceFile := strings.ToLower(strings.TrimSpace(util.AssetMetadataString(info.metadata, "source_file")))
	if sourceFile == "" {
		return ""
	}
	if info.hasLat && info.hasLon {
		return fmt.Sprintf("%s:%s:%0.4f:%0.4f", util.AssetMetadataString(info.metadata, "source_collection"), sourceFile, roundFloat(info.lat, 4), roundFloat(info.lon, 4))
	}
	sourceRecord := util.AssetMetadataString(info.metadata, "source_record")
	if sourceRecord == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s", util.AssetMetadataString(info.metadata, "source_collection"), sourceRecord, sourceFile)
}

func timeLocationKey(info *assetInfo) string {
	if !info.hasTakenAt || !info.hasLat || !info.hasLon {
		return ""
	}
	return fmt.Sprintf("%d:%0.4f:%0.4f", info.takenAt.Unix(), roundFloat(info.lat, 4), roundFloat(info.lon, 4))
}

func roundFloat(value float64, precision int) float64 {
	scale := math.Pow10(precision)
	return math.Round(value*scale) / scale
}

func metadataMap(record *core.Record) (map[string]any, error) {
	raw := record.Get("metadata")
	if raw == nil {
		return map[string]any{}, nil
	}
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var metadata map[string]any
	if err := json.Unmarshal(rawBytes, &metadata); err != nil {
		return nil, err
	}
	if metadata == nil {
		return map[string]any{}, nil
	}
	return metadata, nil
}

func isGeneratedRoutePreviewAsset(record *core.Record) bool {
	return util.IsGeneratedRoutePreviewAsset(record)
}

func optionalFloat(record *core.Record, field string) *float64 {
	raw := record.Get(field)
	if raw == nil {
		return nil
	}
	switch value := raw.(type) {
	case float64:
		if value == 0 {
			return nil
		}
		return &value
	case float32:
		v := float64(value)
		if v == 0 {
			return nil
		}
		return &v
	case int:
		v := float64(value)
		if v == 0 {
			return nil
		}
		return &v
	case int64:
		v := float64(value)
		if v == 0 {
			return nil
		}
		return &v
	case json.Number:
		if parsed, err := value.Float64(); err == nil {
			if parsed == 0 {
				return nil
			}
			return &parsed
		}
	}
	return nil
}

type disjointSet struct {
	parent map[string]string
	rank   map[string]int
}

func newDisjointSet() *disjointSet {
	return &disjointSet{
		parent: map[string]string{},
		rank:   map[string]int{},
	}
}

func (d *disjointSet) add(value string) {
	if _, ok := d.parent[value]; !ok {
		d.parent[value] = value
	}
}

func (d *disjointSet) find(value string) string {
	parent, ok := d.parent[value]
	if !ok {
		d.add(value)
		return value
	}
	if parent != value {
		d.parent[value] = d.find(parent)
	}
	return d.parent[value]
}

func (d *disjointSet) union(a, b string) {
	rootA := d.find(a)
	rootB := d.find(b)
	if rootA == rootB {
		return
	}
	if d.rank[rootA] < d.rank[rootB] {
		d.parent[rootA] = rootB
		return
	}
	if d.rank[rootA] > d.rank[rootB] {
		d.parent[rootB] = rootA
		return
	}
	d.parent[rootB] = rootA
	d.rank[rootA]++
}
