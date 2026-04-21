package trailmerge

import (
	"bytes"
	"errors"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"io"
	"slices"
	"strings"

	"pocketbase/federation"
	"pocketbase/util"
)

const (
	SuggestModeManualSelection         = "manual-selection"
	SuggestModeAutoDiscovery           = "auto-discovery"
	SuggestModeMaintenance             = "maintenance-groups"
	maintenanceLocationBucketDegrees   = 0.003
	maintenanceLocationToleranceMeters = 500.0
)

var (
	ErrUnknownSuggestMode       = errors.New("trail_merge_unknown_suggest_mode")
	ErrMissingActor             = errors.New("trail_merge_missing_actor")
	ErrMissingTrailID           = errors.New("trail_merge_missing_trail_id")
	ErrSameSourceAndTargetTrail = errors.New("trail_merge_same_source_target")
	ErrRequiresMultipleTrails   = errors.New("trail_merge_requires_multiple_trails")
	ErrMissingSourceTrailID     = errors.New("trail_merge_missing_source_trail_id")
	ErrSourceActorMismatch      = errors.New("trail_merge_source_actor_mismatch")
)

type MergeSettings struct {
	SummitLog bool `json:"summitLog"`
	Photos    bool `json:"photos"`
	Comments  bool `json:"comments"`
	Delete    bool `json:"delete"`
	Tags      bool `json:"tags"`
	Likes     bool `json:"likes"`
}

type IntegrationAutoMergeSettings struct {
	Enabled bool `json:"enabled"`
}

type SuggestRequest struct {
	Mode          string   `json:"mode"`
	TrailIDs      []string `json:"trailIds"`
	SourceTrailID string   `json:"sourceTrailId"`
}

type SuggestCandidate struct {
	TrailID    string   `json:"trailId"`
	Score      float64  `json:"score"`
	Reason     string   `json:"reason"`
	Warnings   []string `json:"warnings"`
	Selectable bool     `json:"selectable"`
}

type SuggestResponse struct {
	TargetTrailID string             `json:"targetTrailId"`
	Reason        string             `json:"reason"`
	Warnings      []string           `json:"warnings"`
	Candidates    []SuggestCandidate `json:"candidates"`
}

type SuggestGroup struct {
	GroupID       string   `json:"groupId"`
	TrailIDs      []string `json:"trailIds"`
	TargetTrailID string   `json:"targetTrailId"`
	Reason        string   `json:"reason"`
	Score         float64  `json:"score"`
	Indirect      bool     `json:"indirect"`
}

type SuggestGroupsResponse struct {
	Groups []SuggestGroup `json:"groups"`
}

type mergeContext struct {
	App      core.App
	Client   meilisearch.ServiceManager
	Actor    *core.Record
	ActorID  string
	Target   *core.Record
	Source   *core.Record
	Settings MergeSettings
}

type mergeSideEffects struct {
	CreatedSummitLogIDs []string
	CreatedCommentIDs   []string
	TargetTrailID       string
}

type maintenanceTrailCandidate struct {
	Trail    *core.Record
	Coords   [][2]float64
	StartLat float64
	StartLon float64
	EndLat   float64
	EndLon   float64
	Distance float64
}

func DefaultIntegrationAutoMergeSettings() IntegrationAutoMergeSettings {
	return IntegrationAutoMergeSettings{
		Enabled: false,
	}
}

func DefaultIntegrationAutoMergeMergeSettings() MergeSettings {
	return MergeSettings{
		SummitLog: true,
		Photos:    true,
		Comments:  false,
		Delete:    true,
		Tags:      false,
		Likes:     false,
	}
}

func Suggest(app core.App, actorID string, request SuggestRequest) (*SuggestResponse, error) {
	switch request.Mode {
	case SuggestModeManualSelection:
		return suggestForManualSelection(app, actorID, request.TrailIDs)
	case SuggestModeAutoDiscovery:
		return suggestForAutoDiscovery(app, actorID, request.SourceTrailID)
	default:
		return nil, ErrUnknownSuggestMode
	}
}

func SuggestGroups(app core.App, actorID string, request SuggestRequest) (*SuggestGroupsResponse, error) {
	switch request.Mode {
	case SuggestModeMaintenance:
		return suggestMaintenanceGroups(app, actorID)
	default:
		return nil, ErrUnknownSuggestMode
	}
}

func Merge(app core.App, client meilisearch.ServiceManager, actor *core.Record, sourceTrailID string, targetTrailID string, settings MergeSettings) error {
	if actor == nil {
		return ErrMissingActor
	}
	if sourceTrailID == "" || targetTrailID == "" {
		return ErrMissingTrailID
	}
	if sourceTrailID == targetTrailID {
		return ErrSameSourceAndTargetTrail
	}

	var effects mergeSideEffects
	err := app.RunInTransaction(func(txApp core.App) error {
		source, err := txApp.FindRecordById("trails", sourceTrailID)
		if err != nil {
			return err
		}
		target, err := txApp.FindRecordById("trails", targetTrailID)
		if err != nil {
			return err
		}

		ctx := mergeContext{
			App:      txApp,
			Client:   client,
			Actor:    actor,
			ActorID:  actor.Id,
			Target:   target,
			Source:   source,
			Settings: settings,
		}

		sideEffects, err := mergeTrailIntoTarget(ctx)
		if err != nil {
			return err
		}
		effects = sideEffects

		return nil
	})
	if err != nil {
		return err
	}

	target, err := app.FindRecordById("trails", effects.TargetTrailID)
	if err != nil {
		return err
	}
	if err := util.IndexTrails(app, []*core.Record{target}, client); err != nil {
		return err
	}

	for _, summitLogID := range effects.CreatedSummitLogIDs {
		record, err := app.FindRecordById("summit_logs", summitLogID)
		if err != nil {
			return err
		}
		logAuthor, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
		if err != nil {
			return err
		}
		if err := federation.CreateSummitLogActivity(app, logAuthor, record, pub.CreateType); err != nil {
			return err
		}
	}

	for _, commentID := range effects.CreatedCommentIDs {
		record, err := app.FindRecordById("comments", commentID)
		if err != nil {
			return err
		}
		if err := federation.CreateCommentActivity(app, actor, record, pub.CreateType); err != nil {
			return err
		}
	}

	return nil
}

func CanMerge(app core.App, actorID string, source *core.Record, target *core.Record, deleteSource bool) bool {
	if source == nil || target == nil {
		return false
	}
	if source.Id == target.Id {
		return false
	}
	if !canEditTrail(app, target, actorID) {
		return false
	}
	if deleteSource && !canDeleteTrail(source, actorID) {
		return false
	}

	return true
}

func suggestForManualSelection(app core.App, actorID string, trailIDs []string) (*SuggestResponse, error) {
	if len(trailIDs) < 2 {
		return nil, ErrRequiresMultipleTrails
	}

	trails := make([]*core.Record, 0, len(trailIDs))
	for _, id := range trailIDs {
		trail, err := app.FindRecordById("trails", id)
		if err != nil {
			return nil, err
		}
		trails = append(trails, trail)
	}

	suggestedIndex := 0
	reason := "first_selected"
	for i, trail := range trails {
		logCount, err := app.CountRecords("summit_logs", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
		if err != nil {
			return nil, err
		}
		if logCount > 0 {
			suggestedIndex = i
			reason = "existing_summit_logs"
			break
		}
	}

	candidates := make([]SuggestCandidate, 0, len(trails))
	for i, candidate := range trails {
		warnings := make([]string, 0)
		for j, other := range trails {
			if i == j {
				continue
			}
			warnings = appendUniqueStrings(warnings, geometryWarnings(app, other, candidate)...)
		}

		candidates = append(candidates, SuggestCandidate{
			TrailID:    candidate.Id,
			Score:      candidateScore(i == suggestedIndex, len(warnings)),
			Reason:     reasonForCandidate(i == suggestedIndex, candidate.Id == trails[suggestedIndex].Id, reason),
			Warnings:   warnings,
			Selectable: canEditTrail(app, candidate, actorID),
		})
	}

	selectedCandidate := candidates[suggestedIndex]
	return &SuggestResponse{
		TargetTrailID: selectedCandidate.TrailID,
		Reason:        reason,
		Warnings:      selectedCandidate.Warnings,
		Candidates:    candidates,
	}, nil
}

func suggestForAutoDiscovery(app core.App, actorID string, sourceTrailID string) (*SuggestResponse, error) {
	if sourceTrailID == "" {
		return nil, ErrMissingSourceTrailID
	}

	source, err := app.FindRecordById("trails", sourceTrailID)
	if err != nil {
		return nil, err
	}
	if source.GetString("author") != actorID {
		return nil, ErrSourceActorMismatch
	}

	candidateTrails, err := app.FindRecordsByFilter(
		"trails",
		"author={:actor} && id!={:id} && gpx!=''",
		"",
		-1,
		0,
		dbx.Params{
			"actor": actorID,
			"id":    sourceTrailID,
		},
	)
	if err != nil {
		return nil, err
	}

	candidates := make([]SuggestCandidate, 0)
	for _, candidate := range candidateTrails {
		metrics, err := util.TrailGeometrySimilarity(app, source, candidate)
		if err != nil {
			continue
		}
		if !isStrongGeometryMatch(metrics) {
			continue
		}

		candidates = append(candidates, SuggestCandidate{
			TrailID:    candidate.Id,
			Score:      geometryScore(metrics),
			Reason:     "geometry_match",
			Warnings:   []string{},
			Selectable: canEditTrail(app, candidate, actorID),
		})
	}

	slices.SortFunc(candidates, func(a, b SuggestCandidate) int {
		switch {
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		default:
			return strings.Compare(a.TrailID, b.TrailID)
		}
	})

	response := &SuggestResponse{
		Candidates: candidates,
		Reason:     "no_geometry_match",
		Warnings:   []string{},
	}
	if len(candidates) > 0 {
		response.TargetTrailID = candidates[0].TrailID
		response.Reason = candidates[0].Reason
	}

	return response, nil
}

func suggestMaintenanceGroups(app core.App, actorID string) (*SuggestGroupsResponse, error) {
	trails, err := findMaintenanceCandidateTrails(app, actorID)
	if err != nil {
		return nil, err
	}

	if len(trails) < 2 {
		return &SuggestGroupsResponse{Groups: []SuggestGroup{}}, nil
	}

	preparedTrails := make([]maintenanceTrailCandidate, 0, len(trails))
	for _, trail := range trails {
		candidate, ok, err := prepareMaintenanceTrailCandidate(app, trail)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		preparedTrails = append(preparedTrails, candidate)
	}

	if len(preparedTrails) < 2 {
		return &SuggestGroupsResponse{Groups: []SuggestGroup{}}, nil
	}

	adjacency := make(map[string][]string, len(preparedTrails))
	trailByID := make(map[string]*core.Record, len(preparedTrails))
	trailScores := make(map[string]float64, len(preparedTrails))
	edgeCounts := make(map[string]int, len(preparedTrails))
	startBuckets := groupMaintenanceTrailsByLocation(preparedTrails, false)
	endBuckets := groupMaintenanceTrailsByLocation(preparedTrails, true)

	for _, trail := range preparedTrails {
		trailByID[trail.Trail.Id] = trail.Trail
	}

	for i := range preparedTrails {
		candidates := findMaintenanceComparisonCandidates(preparedTrails[i], startBuckets, endBuckets)
		for _, j := range candidates {
			if i >= j {
				continue
			}

			if !maintenanceDistanceCompatible(preparedTrails[i], preparedTrails[j]) {
				continue
			}

			metrics, err := util.CompareTrailCoordinates(preparedTrails[i].Coords, preparedTrails[j].Coords)
			if err != nil || !isStrongGeometryMatch(metrics) {
				continue
			}

			score := geometryScore(metrics)
			leftID := preparedTrails[i].Trail.Id
			rightID := preparedTrails[j].Trail.Id
			adjacency[leftID] = append(adjacency[leftID], rightID)
			adjacency[rightID] = append(adjacency[rightID], leftID)
			trailScores[leftID] += score
			trailScores[rightID] += score
			edgeCounts[leftID]++
			edgeCounts[rightID]++
		}
	}

	visited := make(map[string]bool, len(trails))
	groups := make([]SuggestGroup, 0)

	for _, trail := range preparedTrails {
		if visited[trail.Trail.Id] || len(adjacency[trail.Trail.Id]) == 0 {
			continue
		}

		componentIDs := collectConnectedTrailIDs(trail.Trail.Id, adjacency, visited)
		if len(componentIDs) < 2 {
			continue
		}

		componentTrails := make([]*core.Record, 0, len(componentIDs))
		groupScore := 0.0
		for _, id := range componentIDs {
			record, ok := trailByID[id]
			if !ok {
				continue
			}
			componentTrails = append(componentTrails, record)
			if edgeCounts[id] > 0 {
				groupScore += trailScores[id] / float64(edgeCounts[id])
			}
		}

		if len(componentTrails) < 2 {
			continue
		}

		slices.SortFunc(componentTrails, func(a, b *core.Record) int {
			nameCompare := strings.Compare(a.GetString("name"), b.GetString("name"))
			if nameCompare != 0 {
				return nameCompare
			}
			return strings.Compare(a.Id, b.Id)
		})

		targetTrailID, reason, err := chooseSuggestedGroupTarget(app, componentTrails, trailScores)
		if err != nil {
			return nil, err
		}

		trailIDs := make([]string, 0, len(componentTrails))
		for _, componentTrail := range componentTrails {
			trailIDs = append(trailIDs, componentTrail.Id)
		}

		groups = append(groups, SuggestGroup{
			GroupID:       strings.Join(trailIDs, ":"),
			TrailIDs:      trailIDs,
			TargetTrailID: targetTrailID,
			Reason:        reason,
			Score:         groupScore / float64(len(componentTrails)),
			Indirect:      isIndirectMaintenanceGroup(componentIDs, adjacency),
		})
	}

	slices.SortFunc(groups, func(a, b SuggestGroup) int {
		switch {
		case len(a.TrailIDs) > len(b.TrailIDs):
			return -1
		case len(a.TrailIDs) < len(b.TrailIDs):
			return 1
		case a.Score > b.Score:
			return -1
		case a.Score < b.Score:
			return 1
		default:
			return strings.Compare(a.GroupID, b.GroupID)
		}
	})

	return &SuggestGroupsResponse{Groups: groups}, nil
}

func isIndirectMaintenanceGroup(componentIDs []string, adjacency map[string][]string) bool {
	if len(componentIDs) < 3 {
		return false
	}

	componentSet := make(map[string]struct{}, len(componentIDs))
	for _, id := range componentIDs {
		componentSet[id] = struct{}{}
	}

	for _, id := range componentIDs {
		directMatches := 0
		for _, neighborID := range adjacency[id] {
			if _, ok := componentSet[neighborID]; ok {
				directMatches++
			}
		}

		if directMatches < len(componentIDs)-1 {
			return true
		}
	}

	return false
}

func findMaintenanceCandidateTrails(app core.App, actorID string) ([]*core.Record, error) {
	authoredTrails, err := app.FindRecordsByFilter(
		"trails",
		"author={:actor} && gpx!=''",
		"",
		-1,
		0,
		dbx.Params{"actor": actorID},
	)
	if err != nil {
		return nil, err
	}

	sharedTrails, err := app.FindRecordsByFilter(
		"trail_share",
		"actor={:actor} && permission='edit'",
		"",
		-1,
		0,
		dbx.Params{"actor": actorID},
	)
	if err != nil {
		return nil, err
	}

	trailMap := make(map[string]*core.Record, len(authoredTrails))
	for _, trail := range authoredTrails {
		trailMap[trail.Id] = trail
	}

	for _, share := range sharedTrails {
		trailID := share.GetString("trail")
		if trailID == "" {
			continue
		}
		if _, exists := trailMap[trailID]; exists {
			continue
		}
		trail, err := app.FindRecordById("trails", trailID)
		if err != nil || trail.GetString("gpx") == "" {
			continue
		}
		trailMap[trailID] = trail
	}

	trails := make([]*core.Record, 0, len(trailMap))
	for _, trail := range trailMap {
		trails = append(trails, trail)
	}

	slices.SortFunc(trails, func(a, b *core.Record) int {
		nameCompare := strings.Compare(a.GetString("name"), b.GetString("name"))
		if nameCompare != 0 {
			return nameCompare
		}
		return strings.Compare(a.Id, b.Id)
	})

	return trails, nil
}

func prepareMaintenanceTrailCandidate(app core.App, trail *core.Record) (maintenanceTrailCandidate, bool, error) {
	coords, err := util.TrailCoordinates(app, trail)
	if err != nil {
		return maintenanceTrailCandidate{}, false, nil
	}
	if len(coords) < 2 {
		return maintenanceTrailCandidate{}, false, nil
	}

	return maintenanceTrailCandidate{
		Trail:    trail,
		Coords:   coords,
		StartLat: coords[0][0],
		StartLon: coords[0][1],
		EndLat:   coords[len(coords)-1][0],
		EndLon:   coords[len(coords)-1][1],
		Distance: trail.GetFloat("distance"),
	}, true, nil
}

func groupMaintenanceTrailsByLocation(trails []maintenanceTrailCandidate, useEnd bool) map[string][]int {
	buckets := make(map[string][]int, len(trails))
	for i, trail := range trails {
		lat := trail.StartLat
		lon := trail.StartLon
		if useEnd {
			lat = trail.EndLat
			lon = trail.EndLon
		}

		key := maintenanceLocationBucketKey(lat, lon)
		buckets[key] = append(buckets[key], i)
	}

	return buckets
}

func findMaintenanceComparisonCandidates(
	trail maintenanceTrailCandidate,
	startBuckets map[string][]int,
	endBuckets map[string][]int,
) []int {
	startMatches := make(map[int]struct{})
	endMatches := make(map[int]struct{})

	for _, startKey := range maintenanceNeighborBucketKeys(trail.StartLat, trail.StartLon) {
		for _, index := range startBuckets[startKey] {
			startMatches[index] = struct{}{}
		}
	}

	for _, endKey := range maintenanceNeighborBucketKeys(trail.EndLat, trail.EndLon) {
		for _, index := range endBuckets[endKey] {
			endMatches[index] = struct{}{}
		}
	}

	result := make([]int, 0, len(startMatches))
	for index := range startMatches {
		if _, exists := endMatches[index]; exists {
			result = append(result, index)
		}
	}

	return result
}

func maintenanceDistanceCompatible(a maintenanceTrailCandidate, b maintenanceTrailCandidate) bool {
	startDistance := util.HaversineDistanceMeters(a.StartLat, a.StartLon, b.StartLat, b.StartLon)
	if startDistance > maintenanceLocationToleranceMeters {
		return false
	}

	endDistance := util.HaversineDistanceMeters(a.EndLat, a.EndLon, b.EndLat, b.EndLon)
	if endDistance > maintenanceLocationToleranceMeters {
		return false
	}

	maxDistance := maxFloat64(a.Distance, b.Distance)
	if maxDistance <= 0 {
		return true
	}

	minDistance := minFloat64(a.Distance, b.Distance)
	absoluteGap := maxDistance - minDistance
	relativeGap := absoluteGap / maxDistance

	return absoluteGap <= 2000 || relativeGap <= 0.2
}

func maintenanceLocationBucketKey(lat float64, lon float64) string {
	latBucket := int(lat / maintenanceLocationBucketDegrees)
	lonBucket := int(lon / maintenanceLocationBucketDegrees)
	return fmt.Sprintf("%d:%d", latBucket, lonBucket)
}

func maintenanceNeighborBucketKeys(lat float64, lon float64) []string {
	latBucket := int(lat / maintenanceLocationBucketDegrees)
	lonBucket := int(lon / maintenanceLocationBucketDegrees)
	keys := make([]string, 0, 9)

	for latOffset := -1; latOffset <= 1; latOffset++ {
		for lonOffset := -1; lonOffset <= 1; lonOffset++ {
			keys = append(keys, fmt.Sprintf("%d:%d", latBucket+latOffset, lonBucket+lonOffset))
		}
	}

	return keys
}

func minFloat64(a float64, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func maxFloat64(a float64, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

func collectConnectedTrailIDs(startID string, adjacency map[string][]string, visited map[string]bool) []string {
	queue := []string{startID}
	component := make([]string, 0)
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		component = append(component, current)

		for _, next := range adjacency[current] {
			if visited[next] {
				continue
			}
			visited[next] = true
			queue = append(queue, next)
		}
	}

	return component
}

func chooseSuggestedGroupTarget(app core.App, trails []*core.Record, trailScores map[string]float64) (string, string, error) {
	bestTrailID := ""
	bestReason := "geometry_match"
	bestLogCount := int64(-1)
	bestScore := -1.0

	for _, trail := range trails {
		logCount, err := app.CountRecords("summit_logs", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
		if err != nil {
			return "", "", err
		}

		score := trailScores[trail.Id]
		switch {
		case logCount > bestLogCount:
			bestTrailID = trail.Id
			if logCount > 0 {
				bestReason = "existing_summit_logs"
			} else {
				bestReason = "geometry_match"
			}
			bestLogCount = logCount
			bestScore = score
		case logCount == bestLogCount && score > bestScore:
			bestTrailID = trail.Id
			if logCount > 0 {
				bestReason = "existing_summit_logs"
			} else {
				bestReason = "geometry_match"
			}
			bestScore = score
		case logCount == bestLogCount && score == bestScore && (bestTrailID == "" || strings.Compare(trail.Id, bestTrailID) < 0):
			bestTrailID = trail.Id
			if logCount > 0 {
				bestReason = "existing_summit_logs"
			} else {
				bestReason = "geometry_match"
			}
		}
	}

	return bestTrailID, bestReason, nil
}

func mergeTrailIntoTarget(ctx mergeContext) (mergeSideEffects, error) {
	targetUpdated := false
	sideEffects := mergeSideEffects{
		CreatedSummitLogIDs: []string{},
		CreatedCommentIDs:   []string{},
		TargetTrailID:       ctx.Target.Id,
	}

	summitLogID, err := createTrailSummitLog(ctx)
	if err != nil {
		return sideEffects, err
	}
	sideEffects.CreatedSummitLogIDs = append(sideEffects.CreatedSummitLogIDs, summitLogID)

	if ctx.Settings.Tags {
		currentTags := ctx.Target.GetStringSlice("tags")
		mergedTags := appendUniqueStrings(currentTags, ctx.Source.GetStringSlice("tags")...)
		if len(mergedTags) != len(currentTags) {
			ctx.Target.Set("tags", mergedTags)
			targetUpdated = true
		}
	}

	if ctx.Settings.Likes {
		if err := mergeTrailLikes(ctx); err != nil {
			return sideEffects, err
		}
	}

	if targetUpdated {
		if err := ctx.App.Save(ctx.Target); err != nil {
			return sideEffects, err
		}
	}

	if ctx.Settings.SummitLog {
		summitLogIDs, err := mergeExistingSummitLogs(ctx)
		if err != nil {
			return sideEffects, err
		}
		sideEffects.CreatedSummitLogIDs = append(sideEffects.CreatedSummitLogIDs, summitLogIDs...)
	}

	if ctx.Settings.Comments {
		commentIDs, err := mergeTrailComments(ctx)
		if err != nil {
			return sideEffects, err
		}
		sideEffects.CreatedCommentIDs = append(sideEffects.CreatedCommentIDs, commentIDs...)
	}

	if err := util.ReassignTrailExternalReferences(ctx.App, ctx.Source.Id, ctx.Target.Id); err != nil {
		return sideEffects, err
	}

	if ctx.Settings.Delete {
		if err := ctx.App.Delete(ctx.Source); err != nil {
			return sideEffects, err
		}
	}

	return sideEffects, nil
}

func createTrailSummitLog(ctx mergeContext) (string, error) {
	collection, err := ctx.App.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)
	record.Load(map[string]any{
		"text":           ctx.Source.GetString("description"),
		"distance":       ctx.Source.GetFloat("distance"),
		"elevation_gain": ctx.Source.GetFloat("elevation_gain"),
		"elevation_loss": ctx.Source.GetFloat("elevation_loss"),
		"duration":       ctx.Source.GetFloat("duration"),
		"date":           ctx.Source.GetDateTime("date"),
		"author":         ctx.ActorID,
		"trail":          ctx.Target.Id,
	})

	if gpxFile, err := cloneRecordFile(ctx.App, ctx.Source, "gpx"); err != nil {
		return "", err
	} else if gpxFile != nil {
		record.Set("gpx", gpxFile)
	}

	if ctx.Settings.Photos {
		photos, err := cloneRecordFiles(ctx.App, ctx.Source, "photos")
		if err != nil {
			return "", err
		}
		if len(photos) > 0 {
			record.Set("photos", photos)
		}
	}

	if err := ctx.App.Save(record); err != nil {
		return "", err
	}

	return record.Id, nil
}

func mergeExistingSummitLogs(ctx mergeContext) ([]string, error) {
	logs, err := ctx.App.FindRecordsByFilter(
		"summit_logs",
		"trail={:trail}",
		"+date",
		-1,
		0,
		dbx.Params{"trail": ctx.Source.Id},
	)
	if err != nil {
		return nil, err
	}
	createdIDs := make([]string, 0)

	for _, sourceLog := range logs {
		if isPrimaryTrailSummitLog(ctx.Source, sourceLog) {
			continue
		}

		collection, err := ctx.App.FindCollectionByNameOrId("summit_logs")
		if err != nil {
			return nil, err
		}

		record := core.NewRecord(collection)
		record.Load(map[string]any{
			"text":           sourceLog.GetString("text"),
			"distance":       sourceLog.GetFloat("distance"),
			"elevation_gain": sourceLog.GetFloat("elevation_gain"),
			"elevation_loss": sourceLog.GetFloat("elevation_loss"),
			"duration":       sourceLog.GetFloat("duration"),
			"date":           sourceLog.GetDateTime("date"),
			"author":         sourceLog.GetString("author"),
			"trail":          ctx.Target.Id,
		})

		if gpxFile, err := cloneRecordFile(ctx.App, sourceLog, "gpx"); err != nil {
			return nil, err
		} else if gpxFile != nil {
			record.Set("gpx", gpxFile)
		}

		photos, err := cloneRecordFiles(ctx.App, sourceLog, "photos")
		if err != nil {
			return nil, err
		}
		if len(photos) > 0 {
			record.Set("photos", photos)
		}

		if err := ctx.App.Save(record); err != nil {
			return nil, err
		}
		createdIDs = append(createdIDs, record.Id)
	}

	return createdIDs, nil
}

func isPrimaryTrailSummitLog(source *core.Record, sourceLog *core.Record) bool {
	if source == nil || sourceLog == nil {
		return false
	}

	if !sourceLog.GetDateTime("date").Time().Equal(source.GetDateTime("date").Time()) {
		return false
	}

	if sourceLog.GetString("text") != source.GetString("description") {
		return false
	}

	if sourceLog.GetFloat("distance") != source.GetFloat("distance") {
		return false
	}

	if sourceLog.GetFloat("elevation_gain") != source.GetFloat("elevation_gain") {
		return false
	}

	if sourceLog.GetFloat("elevation_loss") != source.GetFloat("elevation_loss") {
		return false
	}

	if sourceLog.GetFloat("duration") != source.GetFloat("duration") {
		return false
	}

	return true
}

func mergeTrailComments(ctx mergeContext) ([]string, error) {
	comments, err := ctx.App.FindRecordsByFilter(
		"comments",
		"trail={:trail}",
		"+created",
		-1,
		0,
		dbx.Params{"trail": ctx.Source.Id},
	)
	if err != nil {
		return nil, err
	}

	collection, err := ctx.App.FindCollectionByNameOrId("comments")
	if err != nil {
		return nil, err
	}
	createdIDs := make([]string, 0, len(comments))

	for _, sourceComment := range comments {
		record := core.NewRecord(collection)
		record.Load(map[string]any{
			"text":   buildMergedCommentText(ctx.App, sourceComment),
			"author": ctx.ActorID,
			"trail":  ctx.Target.Id,
		})
		if err := ctx.App.Save(record); err != nil {
			return nil, err
		}
		createdIDs = append(createdIDs, record.Id)
	}

	return createdIDs, nil
}

func mergeTrailLikes(ctx mergeContext) error {
	existingLikes, err := ctx.App.FindRecordsByFilter(
		"trail_like",
		"trail={:trail}",
		"",
		-1,
		0,
		dbx.Params{"trail": ctx.Target.Id},
	)
	if err != nil {
		return err
	}

	existingActors := make(map[string]struct{}, len(existingLikes))
	for _, like := range existingLikes {
		existingActors[like.GetString("actor")] = struct{}{}
	}

	sourceLikes, err := ctx.App.FindRecordsByFilter(
		"trail_like",
		"trail={:trail}",
		"",
		-1,
		0,
		dbx.Params{"trail": ctx.Source.Id},
	)
	if err != nil {
		return err
	}

	collection, err := ctx.App.FindCollectionByNameOrId("trail_like")
	if err != nil {
		return err
	}

	for _, like := range sourceLikes {
		actorID := like.GetString("actor")
		if _, exists := existingActors[actorID]; exists {
			continue
		}

		record := core.NewRecord(collection)
		record.Load(map[string]any{
			"trail": ctx.Target.Id,
			"actor": actorID,
		})
		if err := ctx.App.Save(record); err != nil {
			return err
		}
		existingActors[actorID] = struct{}{}
	}

	return nil
}

func geometryWarnings(app core.App, source *core.Record, target *core.Record) []string {
	sourceCoords, err := util.TrailCoordinates(app, source)
	if err != nil {
		return []string{"missing_geometry"}
	}
	targetCoords, err := util.TrailCoordinates(app, target)
	if err != nil {
		return []string{"missing_geometry"}
	}

	metrics, err := util.CompareTrailCoordinates(sourceCoords, targetCoords)
	if err != nil {
		return []string{"missing_geometry"}
	}
	reversedMetrics, reversedErr := util.CompareTrailCoordinatesReversed(sourceCoords, targetCoords)

	warnings := make([]string, 0)
	if metrics.StartDistanceMeters > 500 {
		warnings = append(warnings, "startpoints_far_apart")
	}
	if metrics.EndDistanceMeters > 500 {
		warnings = append(warnings, "endpoints_far_apart")
	}
	if metrics.MeanDistanceMeters > 150 || metrics.MaxDistanceMeters > 750 {
		warnings = append(warnings, "geometry_differs")
	}
	if reversedErr == nil && isStrongGeometryMatch(reversedMetrics) && !isStrongGeometryMatch(metrics) {
		warnings = append(warnings, "reverse_direction_only")
	}

	return warnings
}

func isStrongGeometryMatch(metrics *util.TrailGeometryMetrics) bool {
	if metrics == nil {
		return false
	}

	return metrics.StartDistanceMeters <= 250 &&
		metrics.EndDistanceMeters <= 250 &&
		metrics.MeanDistanceMeters <= 80 &&
		metrics.MaxDistanceMeters <= 400
}

func geometryScore(metrics *util.TrailGeometryMetrics) float64 {
	if metrics == nil {
		return 0
	}

	return 1.0 / (1.0 + metrics.MeanDistanceMeters + metrics.MaxDistanceMeters*0.25)
}

func candidateScore(isSuggested bool, warningCount int) float64 {
	score := 1.0
	if isSuggested {
		score += 1.0
	}

	return score - float64(warningCount)*0.1
}

func reasonForCandidate(isSuggested bool, isSelected bool, suggestedReason string) string {
	if isSuggested || isSelected {
		return suggestedReason
	}

	return "selected_trail"
}

func canEditTrail(app core.App, trail *core.Record, actorID string) bool {
	if trail.GetString("author") == actorID {
		return true
	}

	shares, err := app.FindRecordsByFilter(
		"trail_share",
		"trail={:trail} && actor={:actor} && permission='edit'",
		"",
		1,
		0,
		dbx.Params{
			"trail": trail.Id,
			"actor": actorID,
		},
	)
	return err == nil && len(shares) > 0
}

func canDeleteTrail(trail *core.Record, actorID string) bool {
	return trail.GetString("author") == actorID
}

func buildMergedCommentText(app core.App, comment *core.Record) string {
	authorHandle := "@someone"
	if authorID := comment.GetString("author"); authorID != "" {
		if author, err := app.FindRecordById("activitypub_actors", authorID); err == nil {
			authorHandle = "@" + author.GetString("preferred_username")
			if !author.GetBool("isLocal") && author.GetString("domain") != "" {
				authorHandle += "@" + author.GetString("domain")
			}
		}
	}

	createdDate := comment.GetDateTime("created").Time().Format("2006-01-02")
	return fmt.Sprintf("%s (%s)\n\n%s", authorHandle, createdDate, comment.GetString("text"))
}

func cloneRecordFiles(app core.App, record *core.Record, field string) ([]*filesystem.File, error) {
	fileNames := record.GetStringSlice(field)
	files := make([]*filesystem.File, 0, len(fileNames))
	for _, name := range fileNames {
		file, err := cloneRecordFileByName(app, record, name)
		if err != nil {
			return nil, err
		}
		if file != nil {
			files = append(files, file)
		}
	}

	return files, nil
}

func cloneRecordFile(app core.App, record *core.Record, field string) (*filesystem.File, error) {
	name := record.GetString(field)
	if name == "" {
		return nil, nil
	}

	return cloneRecordFileByName(app, record, name)
}

func cloneRecordFileByName(app core.App, record *core.Record, fileName string) (*filesystem.File, error) {
	if fileName == "" {
		return nil, nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(record.BaseFilesPath() + "/" + fileName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, reader); err != nil {
		return nil, err
	}

	return filesystem.NewFileFromBytes(buf.Bytes(), fileName)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values)+len(additions))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	for _, value := range additions {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
