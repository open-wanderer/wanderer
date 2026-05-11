package trailmerge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

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

type targetSelectionStats struct {
	TrailID                 string
	SummitLogCount          int64
	CommentCount            int64
	PhotoCount              int
	LikeCount               int64
	TagCount                int
	ExternalReferenceCount  int64
	WaypointCount           int64
	HasDescription          bool
	GeometryCentralityScore float64
	PriorityClass           int
	TotalScore              float64
	CreatedAtUnix           int64
}

type targetSelectionResult struct {
	TrailID string
	Reason  string
	Stats   map[string]targetSelectionStats
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

// Suggest returns merge target suggestions for the requested mode.
// The mode only determines the candidate set; target ranking itself is
// delegated to the shared chooseTargetTrail selection strategy.
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

// SuggestGroups returns temporary groups of potentially repeated or duplicate
// trails for maintenance workflows. Group members are discovered first and the
// suggested target trail is then selected by the shared chooseTargetTrail logic.
func SuggestGroups(app core.App, actorID string, request SuggestRequest) (*SuggestGroupsResponse, error) {
	switch request.Mode {
	case SuggestModeMaintenance:
		return suggestMaintenanceGroups(app, actorID)
	default:
		return nil, ErrUnknownSuggestMode
	}
}

// Merge links a source trail into a target trail in a single transaction.
// It moves or recreates trail-related content according to the provided
// settings and keeps the target trail indexed and federated afterwards.
func Merge(app core.App, client meilisearch.ServiceManager, ctx context.Context, actor *core.Record, sourceTrailID string, targetTrailID string, settings MergeSettings) error {
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

		mergeCtx := mergeContext{
			App:      txApp,
			Client:   client,
			Actor:    actor,
			ActorID:  actor.Id,
			Target:   target,
			Source:   source,
			Settings: settings,
		}

		sideEffects, err := mergeTrailIntoTarget(mergeCtx)
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

		if err := federation.CreateSummitLogActivity(app, ctx, record, pub.CreateType); err != nil {
			return err
		}
	}

	for _, commentID := range effects.CreatedCommentIDs {
		record, err := app.FindRecordById("comments", commentID)
		if err != nil {
			return err
		}
		if err := federation.CreateCommentActivity(app, ctx, record, pub.CreateType); err != nil {
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

// chooseTargetTrail applies the shared target selection strategy used by
// manual selection, auto-discovery and maintenance suggestions.
// It computes trail-level stats, assigns a priority class, derives a weighted
// score and returns both the winning trail and an explainable reason code.
func chooseTargetTrail(app core.App, trails []*core.Record, referenceTrails []*core.Record) (*targetSelectionResult, error) {
	if len(trails) == 0 {
		return &targetSelectionResult{
			TrailID: "",
			Reason:  "deterministic_fallback",
			Stats:   map[string]targetSelectionStats{},
		}, nil
	}

	coordsByTrailID := make(map[string][][2]float64, len(trails)+len(referenceTrails))
	for _, trail := range append(append([]*core.Record{}, trails...), referenceTrails...) {
		if trail == nil {
			continue
		}
		if _, exists := coordsByTrailID[trail.Id]; exists {
			continue
		}
		coords, err := util.TrailCoordinates(app, trail)
		if err != nil {
			coordsByTrailID[trail.Id] = nil
			continue
		}
		coordsByTrailID[trail.Id] = coords
	}

	statsByTrailID := make(map[string]targetSelectionStats, len(trails))
	for _, trail := range trails {
		stats, err := buildTargetSelectionStats(app, trail, trails, referenceTrails, coordsByTrailID)
		if err != nil {
			return nil, err
		}
		statsByTrailID[trail.Id] = stats
	}

	bestTrail := trails[0]
	bestStats := statsByTrailID[bestTrail.Id]
	for _, trail := range trails[1:] {
		stats := statsByTrailID[trail.Id]
		if compareTargetSelectionStats(stats, bestStats) < 0 {
			bestTrail = trail
			bestStats = stats
		}
	}

	return &targetSelectionResult{
		TrailID: bestTrail.Id,
		Reason:  deriveTargetSelectionReason(bestTrail.Id, statsByTrailID),
		Stats:   statsByTrailID,
	}, nil
}

// buildTargetSelectionStats collects the data needed for trail target ranking.
// The score intentionally favors preserving trails that already carry more
// durable user value such as summit logs, external references and richer content.
func buildTargetSelectionStats(
	app core.App,
	trail *core.Record,
	groupTrails []*core.Record,
	referenceTrails []*core.Record,
	coordsByTrailID map[string][][2]float64,
) (targetSelectionStats, error) {
	summitLogCount, err := app.CountRecords("summit_logs", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
	if err != nil {
		return targetSelectionStats{}, err
	}
	commentCount, err := app.CountRecords("comments", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
	if err != nil {
		return targetSelectionStats{}, err
	}
	likeCount, err := app.CountRecords("trail_like", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
	if err != nil {
		return targetSelectionStats{}, err
	}
	externalReferenceCount, err := app.CountRecords("trail_external_reference", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
	if err != nil {
		return targetSelectionStats{}, err
	}
	waypointCount, err := app.CountRecords("waypoints", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trail.Id}))
	if err != nil {
		return targetSelectionStats{}, err
	}

	geometryCentralityScore := trailGeometryCentralityScore(trail, groupTrails, referenceTrails, coordsByTrailID)
	hasDescription := strings.TrimSpace(trail.GetString("description")) != ""
	photoCount := len(trail.GetStringSlice("photos"))
	tagCount := len(trail.GetStringSlice("tags"))
	priorityClass := targetPriorityClass(summitLogCount, externalReferenceCount, commentCount, photoCount, hasDescription)

	totalScore := float64(summitLogCount)*1000.0 +
		float64(externalReferenceCount)*400.0 +
		float64(commentCount)*120.0 +
		float64(photoCount)*80.0 +
		float64(likeCount)*20.0 +
		float64(tagCount)*10.0 +
		boolScore(hasDescription)*15.0 +
		float64(waypointCount)*15.0 +
		geometryCentralityScore*200.0

	return targetSelectionStats{
		TrailID:                 trail.Id,
		SummitLogCount:          summitLogCount,
		CommentCount:            commentCount,
		PhotoCount:              photoCount,
		LikeCount:               likeCount,
		TagCount:                tagCount,
		ExternalReferenceCount:  externalReferenceCount,
		WaypointCount:           waypointCount,
		HasDescription:          hasDescription,
		GeometryCentralityScore: geometryCentralityScore,
		PriorityClass:           priorityClass,
		TotalScore:              totalScore,
		CreatedAtUnix:           trail.GetDateTime("created").Time().Unix(),
	}, nil
}

// trailGeometryCentralityScore measures how well a trail fits geometrically
// within the provided candidate set. Higher values indicate that the trail is
// a more central representative of the group or of the source/candidate set.
func trailGeometryCentralityScore(
	trail *core.Record,
	groupTrails []*core.Record,
	referenceTrails []*core.Record,
	coordsByTrailID map[string][][2]float64,
) float64 {
	trailCoords := coordsByTrailID[trail.Id]
	if len(trailCoords) < 2 {
		return 0
	}

	scoreSum := 0.0
	comparisons := 0
	for _, other := range groupTrails {
		if other == nil || other.Id == trail.Id {
			continue
		}
		metrics, err := util.CompareTrailCoordinates(trailCoords, coordsByTrailID[other.Id])
		if err != nil {
			continue
		}
		scoreSum += geometryScore(metrics)
		comparisons++
	}

	for _, other := range referenceTrails {
		if other == nil || other.Id == trail.Id {
			continue
		}
		metrics, err := util.CompareTrailCoordinates(trailCoords, coordsByTrailID[other.Id])
		if err != nil {
			continue
		}
		scoreSum += geometryScore(metrics)
		comparisons++
	}

	if comparisons == 0 {
		return 0
	}

	return scoreSum / float64(comparisons)
}

// targetPriorityClass creates a coarse ranking tier before weighted scoring.
// Trails with summit logs are preferred first, then trails with external
// references, then trails with richer content, and finally plain trails.
func targetPriorityClass(
	summitLogCount int64,
	externalReferenceCount int64,
	commentCount int64,
	photoCount int,
	hasDescription bool,
) int {
	switch {
	case summitLogCount > 0:
		return 4
	case externalReferenceCount > 0:
		return 3
	case commentCount > 0 || photoCount > 0 || hasDescription:
		return 2
	default:
		return 1
	}
}

func boolScore(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func compareTargetSelectionStats(a targetSelectionStats, b targetSelectionStats) int {
	switch {
	case a.PriorityClass > b.PriorityClass:
		return -1
	case a.PriorityClass < b.PriorityClass:
		return 1
	case a.TotalScore > b.TotalScore:
		return -1
	case a.TotalScore < b.TotalScore:
		return 1
	case a.GeometryCentralityScore > b.GeometryCentralityScore:
		return -1
	case a.GeometryCentralityScore < b.GeometryCentralityScore:
		return 1
	case a.CreatedAtUnix < b.CreatedAtUnix:
		return -1
	case a.CreatedAtUnix > b.CreatedAtUnix:
		return 1
	default:
		return strings.Compare(a.TrailID, b.TrailID)
	}
}

// deriveTargetSelectionReason returns a single explainable reason code for the
// selected trail. The reason reflects the strongest distinguishing factor that
// made the winner stand out against the remaining candidates.
func deriveTargetSelectionReason(selectedTrailID string, statsByTrailID map[string]targetSelectionStats) string {
	selected, ok := statsByTrailID[selectedTrailID]
	if !ok {
		return "deterministic_fallback"
	}

	allStats := make([]targetSelectionStats, 0, len(statsByTrailID))
	for _, stats := range statsByTrailID {
		allStats = append(allStats, stats)
	}

	if selected.SummitLogCount > maxOtherInt64(selectedTrailID, allStats, func(stats targetSelectionStats) int64 {
		return stats.SummitLogCount
	}) {
		return "highest_summit_log_count"
	}
	if selected.ExternalReferenceCount > maxOtherInt64(selectedTrailID, allStats, func(stats targetSelectionStats) int64 {
		return stats.ExternalReferenceCount
	}) {
		return "most_external_references"
	}
	selectedContentScore := trailContentScore(selected)
	if selectedContentScore > maxOtherFloat64(selectedTrailID, allStats, trailContentScore) {
		return "most_complete_content"
	}
	if selected.GeometryCentralityScore > maxOtherFloat64(selectedTrailID, allStats, func(stats targetSelectionStats) float64 {
		return stats.GeometryCentralityScore
	}) {
		return "most_central_geometry"
	}
	if selected.CreatedAtUnix < minOtherInt64(selectedTrailID, allStats, func(stats targetSelectionStats) int64 {
		return stats.CreatedAtUnix
	}) {
		return "oldest_trail"
	}

	return "deterministic_fallback"
}

func trailContentScore(stats targetSelectionStats) float64 {
	return float64(stats.CommentCount)*120.0 +
		float64(stats.PhotoCount)*80.0 +
		float64(stats.LikeCount)*20.0 +
		float64(stats.TagCount)*10.0 +
		boolScore(stats.HasDescription)*15.0 +
		float64(stats.WaypointCount)*15.0
}

func maxOtherInt64(selectedTrailID string, stats []targetSelectionStats, valueFn func(targetSelectionStats) int64) int64 {
	var maxValue int64
	initialized := false
	for _, stat := range stats {
		if stat.TrailID == selectedTrailID {
			continue
		}
		value := valueFn(stat)
		if !initialized || value > maxValue {
			maxValue = value
			initialized = true
		}
	}
	if !initialized {
		return -1
	}
	return maxValue
}

func minOtherInt64(selectedTrailID string, stats []targetSelectionStats, valueFn func(targetSelectionStats) int64) int64 {
	var minValue int64
	initialized := false
	for _, stat := range stats {
		if stat.TrailID == selectedTrailID {
			continue
		}
		value := valueFn(stat)
		if !initialized || value < minValue {
			minValue = value
			initialized = true
		}
	}
	if !initialized {
		return selectedOnlyMinInt64Fallback()
	}
	return minValue
}

func selectedOnlyMinInt64Fallback() int64 {
	return 1<<63 - 1
}

func maxOtherFloat64(selectedTrailID string, stats []targetSelectionStats, valueFn func(targetSelectionStats) float64) float64 {
	maxValue := 0.0
	initialized := false
	for _, stat := range stats {
		if stat.TrailID == selectedTrailID {
			continue
		}
		value := valueFn(stat)
		if !initialized || value > maxValue {
			maxValue = value
			initialized = true
		}
	}
	if !initialized {
		return -1
	}
	return maxValue
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

	selection, err := chooseTargetTrail(app, trails, nil)
	if err != nil {
		return nil, err
	}

	suggestedTrailID := selection.TrailID
	reason := selection.Reason
	if suggestedTrailID == "" && len(trails) > 0 {
		suggestedTrailID = trails[0].Id
		reason = "deterministic_fallback"
	}

	candidates := make([]SuggestCandidate, 0, len(trails))
	for _, candidate := range trails {
		warnings := make([]string, 0)
		for _, other := range trails {
			if candidate.Id == other.Id {
				continue
			}
			warnings = appendUniqueStrings(warnings, geometryWarnings(app, other, candidate)...)
		}

		stats, ok := selection.Stats[candidate.Id]
		score := 0.0
		if ok {
			score = stats.TotalScore - float64(len(warnings))*0.1
		}

		candidates = append(candidates, SuggestCandidate{
			TrailID:    candidate.Id,
			Score:      score,
			Reason:     reasonForCandidate(candidate.Id == suggestedTrailID, reason),
			Warnings:   warnings,
			Selectable: canEditTrail(app, candidate, actorID),
		})
	}

	return &SuggestResponse{
		TargetTrailID: suggestedTrailID,
		Reason:        reason,
		Warnings:      candidateWarningsForTrail(candidates, suggestedTrailID),
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
	candidateTrailsByID := make(map[string]*core.Record)
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
			Reason:     "selected_trail",
			Warnings:   []string{},
			Selectable: canEditTrail(app, candidate, actorID),
		})
		candidateTrailsByID[candidate.Id] = candidate
	}

	response := &SuggestResponse{
		Candidates: candidates,
		Reason:     "no_geometry_match",
		Warnings:   []string{},
	}
	if len(candidates) > 0 {
		eligibleTrails := make([]*core.Record, 0, len(candidates))
		for _, candidate := range candidates {
			if trail, ok := candidateTrailsByID[candidate.TrailID]; ok {
				eligibleTrails = append(eligibleTrails, trail)
			}
		}

		selection, err := chooseTargetTrail(app, eligibleTrails, []*core.Record{source})
		if err != nil {
			return nil, err
		}
		response.TargetTrailID = selection.TrailID
		response.Reason = selection.Reason

		for i := range candidates {
			candidates[i].Reason = reasonForCandidate(candidates[i].TrailID == selection.TrailID, selection.Reason)
			if stats, ok := selection.Stats[candidates[i].TrailID]; ok {
				candidates[i].Score = stats.TotalScore
			}
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

		targetTrailID, reason, err := chooseSuggestedGroupTarget(app, componentTrails)
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

func chooseSuggestedGroupTarget(app core.App, trails []*core.Record) (string, string, error) {
	selection, err := chooseTargetTrail(app, trails, nil)
	if err != nil {
		return "", "", err
	}

	return selection.TrailID, selection.Reason, nil
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

func reasonForCandidate(isSuggested bool, suggestedReason string) string {
	if isSuggested {
		return suggestedReason
	}

	return "selected_trail"
}

func candidateWarningsForTrail(candidates []SuggestCandidate, trailID string) []string {
	for _, candidate := range candidates {
		if candidate.TrailID == trailID {
			return candidate.Warnings
		}
	}

	return []string{}
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
