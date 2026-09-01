package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const (
	geoJSONMaxShards                = 64
	geoJSONShardModeHash            = "hash"
	geoJSONShardModeCapacityCreated = "capacity-created"
	geoJSONCapacityArrivalMixed     = "mixed"
	geoJSONCapacityArrivalClustered = "clustered"

	geoJSONShardAlgorithm         = "fnv1a64(trail_id) modulo shards"
	geoJSONCapacityShardAlgorithm = "synthetic local arrival timestamp ascending, trail_id ascending, chunks of max_trails_per_shard"
	geoJSONCapacitySnapshotScope  = "initial snapshot/build economics only; production requires persisted geo_shard_id and shard manifest; additions, deletes, and federation sync are not tested"
)

func normalizedGeoJSONShardMode(mode string) string {
	if mode == "" {
		return geoJSONShardModeHash
	}
	return mode
}

// geoJSONShardCount returns the number of indexes materialized by a build.
// Capacity mode grows the index set from the corpus size instead of requiring
// an operator to choose a shard count in advance.
func geoJSONShardCount(config GeoJSONIndexConfig, documentCount int) (int, error) {
	if documentCount < 0 {
		return 0, fmt.Errorf("GeoJSON document count must not be negative: %d", documentCount)
	}
	switch normalizedGeoJSONShardMode(config.ShardMode) {
	case geoJSONShardModeHash:
		if config.Shards < 1 || config.Shards > geoJSONMaxShards {
			return 0, fmt.Errorf("GeoJSON shard count must be in 1..%d: %d", geoJSONMaxShards, config.Shards)
		}
		return config.Shards, nil
	case geoJSONShardModeCapacityCreated:
		if config.MaxTrailsPerShard < 1 {
			return 0, fmt.Errorf("GeoJSON max trails per shard must be positive: %d", config.MaxTrailsPerShard)
		}
		if documentCount == 0 {
			return 1, nil
		}
		shards := 1 + (documentCount-1)/config.MaxTrailsPerShard
		if shards > geoJSONMaxShards {
			minimumCapacity := 1 + (documentCount-1)/geoJSONMaxShards
			return 0, fmt.Errorf("GeoJSON capacity-created plan would create %d shards; maximum is %d (for %d trails use max-trails-per-shard >= %d)", shards, geoJSONMaxShards, documentCount, minimumCapacity)
		}
		return shards, nil
	default:
		return 0, fmt.Errorf("unknown GeoJSON shard mode %q", config.ShardMode)
	}
}

func validateGeoJSONShardRun(config GeoJSONIndexConfig, documentCount int, indexOnly bool) (int, error) {
	shards, err := geoJSONShardCount(config, documentCount)
	if err != nil {
		return 0, err
	}
	if normalizedGeoJSONShardMode(config.ShardMode) == geoJSONShardModeCapacityCreated {
		// Full RunConfig validation restricts this to the fixed 1.53 direct
		// product search-only contract. The shard planner itself only needs to
		// establish that the index count is valid.
		if !indexOnly && shards < 2 {
			return 0, errors.New("capacity-created search requires at least two derived shards; lower --geojson-shard-max-trails or use --index-only")
		}
		return shards, nil
	}
	if shards > 1 && !indexOnly {
		return 0, fmt.Errorf("GeoJSON shard mode %s produces %d shards; multiple shards currently require --index-only because federated query semantics are not part of this index-build experiment", normalizedGeoJSONShardMode(config.ShardMode), shards)
	}
	return shards, nil
}

func geoJSONShardAlgorithmForMode(mode string) string {
	if normalizedGeoJSONShardMode(mode) == geoJSONShardModeCapacityCreated {
		return geoJSONCapacityShardAlgorithm
	}
	return geoJSONShardAlgorithm
}

// geoJSONShardForID is deliberately independent of corpus order and process
// state. Existing trails therefore remain on the same shard across rebuilds,
// incremental updates and benchmark hosts as long as the shard count is kept.
func geoJSONShardForID(id string, shards int) int {
	if shards <= 1 {
		return 0
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(id))
	return int(hasher.Sum64() % uint64(shards))
}

func geoJSONIndexUIDs(shards int) []string {
	if shards <= 1 {
		return []string{benchmarkIndexUID}
	}
	result := make([]string, shards)
	for shard := range result {
		result[shard] = fmt.Sprintf("%s_geojson_shard_%02d", benchmarkIndexUID, shard)
	}
	return result
}

func splitGeoJSONDocumentsByShard(documents []map[string]any, shards int) ([][]map[string]any, error) {
	if shards < 1 {
		return nil, fmt.Errorf("GeoJSON shard count must be positive: %d", shards)
	}
	result := make([][]map[string]any, shards)
	for index, document := range documents {
		id, ok := document["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("GeoJSON document %d has no string id", index)
		}
		shard := geoJSONShardForID(id, shards)
		result[shard] = append(result[shard], document)
	}
	return result, nil
}

type capacityCreatedDocument struct {
	document map[string]any
	id       string
	created  int64
}

// geoJSONCapacityLocalCreatedPlan models the immutable local arrival timestamp
// that production would assign when first storing a trail. It is deliberately
// separate from ProductQueryDocument.CreatedUnix: that existing synthetic
// product field cycles every 1,825 records and is not an arrival history.
func geoJSONCapacityLocalCreatedPlan(trails []Trail, seed int64, arrival string) (map[string]int64, error) {
	if arrival == "" {
		arrival = geoJSONCapacityArrivalMixed
	}
	if arrival != geoJSONCapacityArrivalMixed && arrival != geoJSONCapacityArrivalClustered {
		return nil, fmt.Errorf("unknown GeoJSON capacity arrival %q", arrival)
	}
	type arrivalRecord struct {
		id     string
		digest [sha256.Size]byte
	}
	records := make([]arrivalRecord, 0, len(trails))
	seen := make(map[string]struct{}, len(trails))
	var seedBytes [8]byte
	binary.BigEndian.PutUint64(seedBytes[:], uint64(seed))
	for index, trail := range trails {
		if trail.ID == "" {
			return nil, fmt.Errorf("GeoJSON capacity arrival trail %d has no id", index)
		}
		if _, duplicate := seen[trail.ID]; duplicate {
			return nil, fmt.Errorf("GeoJSON capacity arrival contains duplicate id %q", trail.ID)
		}
		seen[trail.ID] = struct{}{}
		record := arrivalRecord{id: trail.ID}
		if arrival == geoJSONCapacityArrivalMixed {
			hasher := sha256.New()
			_, _ = hasher.Write(seedBytes[:])
			_, _ = hasher.Write([]byte{0})
			_, _ = hasher.Write([]byte(trail.ID))
			copy(record.digest[:], hasher.Sum(nil))
		}
		records = append(records, record)
	}
	if arrival == geoJSONCapacityArrivalMixed {
		sort.Slice(records, func(left, right int) bool {
			if compared := bytes.Compare(records[left].digest[:], records[right].digest[:]); compared != 0 {
				return compared < 0
			}
			return records[left].id < records[right].id
		})
	}
	// These values are planner-only monotone local timestamps. They are not
	// added to the Meilisearch payload and cannot change product sorting.
	result := make(map[string]int64, len(records))
	for index, record := range records {
		result[record.id] = int64(index)
	}
	return result, nil
}

// splitGeoJSONDocuments partitions the exact Meilisearch documents that will
// be submitted. Capacity mode consumes only the explicit local-created plan;
// neither the indexed product `created` field nor federation/origin fields are
// inspected or accepted as fallbacks.
func splitGeoJSONDocuments(documents []map[string]any, config GeoJSONIndexConfig, localCreated map[string]int64) ([][]map[string]any, error) {
	mode := normalizedGeoJSONShardMode(config.ShardMode)
	if mode == geoJSONShardModeHash {
		return splitGeoJSONDocumentsByShard(documents, config.Shards)
	}
	if mode != geoJSONShardModeCapacityCreated {
		return nil, fmt.Errorf("unknown GeoJSON shard mode %q", config.ShardMode)
	}
	shardCount, err := geoJSONShardCount(config, len(documents))
	if err != nil {
		return nil, err
	}
	ordered := make([]capacityCreatedDocument, 0, len(documents))
	seenIDs := make(map[string]struct{}, len(documents))
	for index, document := range documents {
		id, ok := document["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("GeoJSON document %d has no string id", index)
		}
		if _, duplicate := seenIDs[id]; duplicate {
			return nil, fmt.Errorf("GeoJSON capacity-created documents contain duplicate id %q", id)
		}
		seenIDs[id] = struct{}{}
		created, exists := localCreated[id]
		if !exists {
			return nil, fmt.Errorf("GeoJSON document %q is missing from the local-created arrival plan", id)
		}
		ordered = append(ordered, capacityCreatedDocument{document: document, id: id, created: created})
	}
	if len(localCreated) != len(documents) {
		return nil, fmt.Errorf("GeoJSON local-created arrival plan contains %d trails, want %d", len(localCreated), len(documents))
	}
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].created != ordered[right].created {
			return ordered[left].created < ordered[right].created
		}
		return ordered[left].id < ordered[right].id
	})

	result := make([][]map[string]any, shardCount)
	for index, value := range ordered {
		shard := index / config.MaxTrailsPerShard
		result[shard] = append(result[shard], value.document)
	}
	return result, nil
}

func geoJSONShardAssignmentSHA256(documentShards [][]map[string]any) (string, error) {
	type assignment struct {
		id    string
		shard int
	}
	assignments := make([]assignment, 0)
	seen := make(map[string]struct{})
	for shard, documents := range documentShards {
		for _, document := range documents {
			id, ok := document["id"].(string)
			if !ok || id == "" {
				return "", fmt.Errorf("GeoJSON shard %d contains a document without string id", shard)
			}
			if _, duplicate := seen[id]; duplicate {
				return "", fmt.Errorf("GeoJSON assignment contains duplicate id %q", id)
			}
			seen[id] = struct{}{}
			assignments = append(assignments, assignment{id: id, shard: shard})
		}
	}
	sort.Slice(assignments, func(left, right int) bool { return assignments[left].id < assignments[right].id })

	hasher := sha256.New()
	var number [8]byte
	for _, assignment := range assignments {
		binary.BigEndian.PutUint64(number[:], uint64(assignment.shard))
		_, _ = hasher.Write(number[:])
		binary.BigEndian.PutUint64(number[:], uint64(len(assignment.id)))
		_, _ = hasher.Write(number[:])
		_, _ = hasher.Write([]byte(assignment.id))
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func populateGeoJSONShardReport(
	report *GeoJSONIndexReport,
	sourceTrails, indexedTrails []Trail,
	documentShards [][]map[string]any,
	config GeoJSONIndexConfig,
) error {
	if report == nil {
		return fmt.Errorf("GeoJSON shard report is nil")
	}
	if len(sourceTrails) != len(indexedTrails) {
		return fmt.Errorf("GeoJSON source/indexed trail mismatch: %d/%d", len(sourceTrails), len(indexedTrails))
	}
	report.ShardMode = normalizedGeoJSONShardMode(config.ShardMode)
	report.MaxTrailsPerShard = config.MaxTrailsPerShard
	report.CapacityArrival = config.CapacityArrival
	if report.ShardMode == geoJSONShardModeCapacityCreated {
		report.AssignmentScope = geoJSONCapacitySnapshotScope
	}
	report.Shards = len(documentShards)
	report.ShardAlgorithm = geoJSONShardAlgorithmForMode(config.ShardMode)
	assignmentSHA256, err := geoJSONShardAssignmentSHA256(documentShards)
	if err != nil {
		return err
	}
	report.ShardAssignmentSHA256 = assignmentSHA256
	report.ShardDocumentCounts = make([]int, len(documentShards))
	report.ShardSourceVertices = make([]int, len(documentShards))
	report.ShardIndexedVertices = make([]int, len(documentShards))
	report.ShardSourceSegments = make([]int, len(documentShards))
	report.ShardIndexedSegments = make([]int, len(documentShards))

	trailIndexes := make(map[string]int, len(sourceTrails))
	for index, trail := range sourceTrails {
		trailIndexes[trail.ID] = index
	}
	seen := make(map[string]struct{}, len(sourceTrails))
	for shard, documents := range documentShards {
		for _, document := range documents {
			id, ok := document["id"].(string)
			if !ok || id == "" {
				return fmt.Errorf("GeoJSON shard %d contains a document without string id", shard)
			}
			trailIndex, exists := trailIndexes[id]
			if !exists {
				return fmt.Errorf("GeoJSON shard %d contains unknown trail %q", shard, id)
			}
			if _, duplicate := seen[id]; duplicate {
				return fmt.Errorf("GeoJSON trail %q occurs in more than one shard", id)
			}
			seen[id] = struct{}{}
			report.ShardDocumentCounts[shard]++
			for _, part := range sourceTrails[trailIndex].Parts {
				report.ShardSourceVertices[shard] += len(part)
				report.ShardSourceSegments[shard] += segmentCount(part)
			}
			for _, part := range indexedTrails[trailIndex].Parts {
				report.ShardIndexedVertices[shard] += len(part)
				report.ShardIndexedSegments[shard] += segmentCount(part)
			}
		}
	}
	if len(seen) != len(sourceTrails) {
		return fmt.Errorf("GeoJSON shards contain %d trails, want %d", len(seen), len(sourceTrails))
	}
	return nil
}

// indexGeoJSONShards measures one common wall-clock/RSS/storage phase. All
// shard tasks are enqueued before the default completion barrier so the task
// scheduler sees the complete workload. Meilisearch may still process the
// indexes serially. Per-shard rows only contain metrics that can be attributed
// without double-counting.
func indexGeoJSONShards(
	ctx context.Context,
	client meilisearch.ServiceManager,
	container *meiliContainer,
	indexUIDs []string,
	documentShards [][]map[string]any,
	batchSize, maxBatchBytes, maxPendingBatches int,
) (PhaseReport, []GeoJSONShardBuildReport, error) {
	if len(indexUIDs) == 0 || len(indexUIDs) != len(documentShards) {
		return PhaseReport{}, nil, fmt.Errorf("GeoJSON indexes/shards mismatch: %d indexes for %d shards", len(indexUIDs), len(documentShards))
	}

	if batchSize < 1 {
		batchSize = 100
	}
	if maxBatchBytes < 3 {
		maxBatchBytes = meiliMaxBatchBytes
	}

	type pendingTask struct {
		shard int
		info  *meilisearch.TaskInfo
	}
	type completedTask struct {
		pending pendingTask
		task    *meilisearch.Task
		err     error
	}

	started := time.Now()
	aggregate := PhaseReport{}
	builds := make([]GeoJSONShardBuildReport, len(indexUIDs))
	for shard, indexUID := range indexUIDs {
		builds[shard] = GeoJSONShardBuildReport{Shard: shard, IndexUID: indexUID}
	}
	engineMonitor := startContainerRSSMonitor(container)
	clientMonitor := startRSSMonitor(os.Getpid())
	monitorsFinished := false
	finishMonitors := func() {
		if monitorsFinished {
			return
		}
		monitorsFinished = true
		aggregate.EngineBaselineBytes, aggregate.EnginePeakBytes, aggregate.EnginePeakDelta = engineMonitor.finish()
		aggregate.ClientBaselineRSSBytes, aggregate.ClientPeakRSSBytes, aggregate.ClientPeakRSSDelta = clientMonitor.finish()
	}

	waitPending := func(tasks []pendingTask) error {
		if len(tasks) == 0 {
			return nil
		}
		aggregate.BarrierCount++
		participatingShards := make(map[int]struct{}, len(tasks))
		for _, pending := range tasks {
			participatingShards[pending.shard] = struct{}{}
		}
		for shard := range participatingShards {
			builds[shard].Phase.BarrierCount++
		}
		results := make(chan completedTask, len(tasks))
		for _, pending := range tasks {
			pending := pending
			go func() {
				task, err := waitForSuccessfulTask(ctx, client, pending.info)
				results <- completedTask{pending: pending, task: task, err: err}
			}()
		}
		var firstErr error
		for range tasks {
			result := <-results
			readyMS := milliseconds(time.Since(started))
			if result.task != nil && !result.task.FinishedAt.IsZero() {
				readyMS = max(0, milliseconds(result.task.FinishedAt.Sub(started)))
			}
			phase := &builds[result.pending.shard].Phase
			phase.ReadyMS = max(phase.ReadyMS, readyMS)
			if result.err != nil && firstErr == nil {
				firstErr = fmt.Errorf("GeoJSON shard %d (%s): %w", result.pending.shard, indexUIDs[result.pending.shard], result.err)
			}
		}
		return firstErr
	}

	populateStats := func(statsCtx context.Context) error {
		stats, err := client.GetStatsWithContext(statsCtx)
		if err != nil {
			return err
		}
		aggregate.DocumentCount = 0
		for shard, indexUID := range indexUIDs {
			indexStats, found := stats.Indexes[indexUID]
			if !found {
				return fmt.Errorf("Meilisearch stats are missing GeoJSON shard index %s", indexUID)
			}
			builds[shard].Phase.DocumentCount = indexStats.NumberOfDocuments
			builds[shard].Phase.LastBatch = latestMeiliBatchReportForIndexes(statsCtx, client, []string{indexUID})
			aggregate.DocumentCount += indexStats.NumberOfDocuments
		}
		aggregate.DiskBytes = stats.DatabaseSize
		aggregate.UsedDiskBytes = stats.UsedDatabaseSize
		aggregate.FilesystemBytes, err = directorySize(statsCtx, container.dataDir)
		aggregate.LastBatch = nil
		return err
	}

	fail := func(buildErr error) (PhaseReport, []GeoJSONShardBuildReport, error) {
		aggregate.ReadyMS = milliseconds(time.Since(started))
		finishMonitors()
		captureCtx, captureCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = populateStats(captureCtx)
		captureCancel()
		return aggregate, builds, buildErr
	}

	pending := make([]pendingTask, 0)
	for shard, indexUID := range indexUIDs {
		shardSubmitStarted := time.Now()
		err := forEachMeiliDocumentBatch(ctx, documentShards[shard], batchSize, maxBatchBytes, func(batch []byte, documentCount int) error {
			requestStarted := time.Now()
			taskInfo, err := client.Index(indexUID).AddDocumentsWithContext(ctx, batch, nil)
			requestMS := milliseconds(time.Since(requestStarted))
			aggregate.RequestMS += requestMS
			builds[shard].Phase.RequestMS += requestMS
			if err != nil {
				return err
			}
			if taskInfo == nil {
				return fmt.Errorf("Meilisearch returned no document task for shard %d (%s)", shard, indexUID)
			}
			aggregate.SubmittedDocuments += documentCount
			aggregate.SubmittedBytes += int64(len(batch))
			aggregate.TaskCount++
			builds[shard].Phase.SubmittedDocuments += documentCount
			builds[shard].Phase.SubmittedBytes += int64(len(batch))
			builds[shard].Phase.TaskCount++
			pending = append(pending, pendingTask{shard: shard, info: taskInfo})
			if maxPendingBatches > 0 && len(pending) >= maxPendingBatches {
				err := waitPending(pending)
				pending = pending[:0]
				return err
			}
			return nil
		})
		builds[shard].Phase.SubmitMS = milliseconds(time.Since(shardSubmitStarted))
		aggregate.SubmitMS = milliseconds(time.Since(started))
		if err != nil {
			return fail(fmt.Errorf("submit GeoJSON shard %d (%s): %w", shard, indexUID, err))
		}
	}
	if err := waitPending(pending); err != nil {
		return fail(err)
	}
	aggregate.ReadyMS = milliseconds(time.Since(started))
	finishMonitors()
	if err := populateStats(ctx); err != nil {
		return aggregate, builds, err
	}
	if aggregate.DocumentCount != int64(aggregate.SubmittedDocuments) {
		return aggregate, builds, fmt.Errorf("sharded GeoJSON indexes contain %d documents, want %d", aggregate.DocumentCount, aggregate.SubmittedDocuments)
	}
	for shard := range builds {
		if builds[shard].Phase.DocumentCount != int64(len(documentShards[shard])) {
			return aggregate, builds, fmt.Errorf("GeoJSON shard %d contains %d documents, want %d", shard, builds[shard].Phase.DocumentCount, len(documentShards[shard]))
		}
	}
	return aggregate, builds, nil
}
