package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type candidateQuery func(context.Context, QueryPoint, float64) (CandidateResult, error)

type candidateGeometryLoader func(context.Context, map[string]struct{}) (geometryLoad, error)

type refinementResult struct {
	IDs           map[string]struct{}
	Database      time.Duration
	Exact         time.Duration
	LoadedRows    int
	GeometryBytes int64
}

// accuracyOracle holds the exact distance matrix once per run. Keeping a
// compact matrix avoids repeating the expensive point-to-polyline calculation
// for every backend while also avoiding a large set for each radius.
type accuracyOracle struct {
	trailIndex map[string]int
	trailIDs   []string
	queries    map[string]oracleQuery
}

type oracleQuery struct {
	distances      []float64
	expectedCounts map[float64]int
	boundaryCounts map[float64]int
}

func buildAccuracyOracle(ctx context.Context, dataset Dataset, radii []float64) (*accuracyOracle, error) {
	oracle := &accuracyOracle{
		trailIndex: make(map[string]int, len(dataset.Trails)),
		trailIDs:   make([]string, len(dataset.Trails)),
		queries:    make(map[string]oracleQuery, len(dataset.QueryPoints)),
	}
	for index, trail := range dataset.Trails {
		oracle.trailIndex[trail.ID] = index
		oracle.trailIDs[index] = trail.ID
	}

	results := make([]oracleQuery, len(dataset.QueryPoints))
	jobs := make(chan int)
	workerCount := min(runtime.GOMAXPROCS(0), len(dataset.QueryPoints))
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for queryIndex := range jobs {
				queryPoint := dataset.QueryPoints[queryIndex]
				result := oracleQuery{
					distances:      make([]float64, len(dataset.Trails)),
					expectedCounts: make(map[float64]int, len(radii)),
					boundaryCounts: make(map[float64]int, len(radii)),
				}
				for _, radius := range radii {
					result.expectedCounts[radius] = 0
					result.boundaryCounts[radius] = 0
				}
				for trailIndex, trail := range dataset.Trails {
					if trailIndex%256 == 0 && ctx.Err() != nil {
						return
					}
					distance, err := pointToTrailDistanceMetersContext(ctx, queryPoint.Point, trail)
					if err != nil {
						return
					}
					result.distances[trailIndex] = distance
					for _, radius := range radii {
						if isBoundaryDistance(distance, radius) {
							result.boundaryCounts[radius]++
						} else if distance < radius {
							result.expectedCounts[radius]++
						}
					}
				}
				results[queryIndex] = result
			}
		}()
	}

	for queryIndex := range dataset.QueryPoints {
		select {
		case jobs <- queryIndex:
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			return nil, ctx.Err()
		}
	}
	close(jobs)
	workers.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for index, queryPoint := range dataset.QueryPoints {
		oracle.queries[queryPoint.ID] = results[index]
	}
	return oracle, nil
}

func (o *accuracyOracle) counts(queryID string, radius float64) (int, int, error) {
	query, ok := o.queries[queryID]
	if !ok {
		return 0, 0, fmt.Errorf("query %q is missing from the accuracy oracle", queryID)
	}
	expected, ok := query.expectedCounts[radius]
	if !ok {
		return 0, 0, fmt.Errorf("radius %.6f is missing from the accuracy oracle", radius)
	}
	return expected, query.boundaryCounts[radius], nil
}

func (o *accuracyOracle) classify(queryID string, radius float64, ids map[string]struct{}) (int, int, error) {
	query, ok := o.queries[queryID]
	if !ok {
		return 0, 0, fmt.Errorf("query %q is missing from the accuracy oracle", queryID)
	}
	truePositives := 0
	falsePositives := 0
	for id := range ids {
		trailIndex, ok := o.trailIndex[id]
		if !ok {
			falsePositives++
			continue
		}
		distance := query.distances[trailIndex]
		if isBoundaryDistance(distance, radius) {
			continue
		}
		if distance < radius {
			truePositives++
		} else {
			falsePositives++
		}
	}
	return truePositives, falsePositives, nil
}

func isBoundaryDistance(distance, radius float64) bool {
	return math.Abs(distance-radius) <= math.Max(0.5, radius*1e-7)
}

type queryAccumulator struct {
	report             QueryReport
	first              []float64
	candidate          []float64
	verify             []float64
	database           []float64
	exact              []float64
	total              []float64
	engine             []float64
	candidateTotal     int
	matchTotal         int
	candidateTrue      int
	expectedComparable int
	loadedRows         int
	geometryBytes      int64
	measuredRefines    int
}

func benchmarkQueries(
	ctx context.Context,
	dataset Dataset,
	radii []float64,
	repetitions int,
	geometryLoader candidateGeometryLoader,
	oracle *accuracyOracle,
	query candidateQuery,
) ([]QueryReport, error) {
	if repetitions < 1 {
		repetitions = 1
	}

	groups := make(map[string]*queryAccumulator)
	for _, queryPoint := range dataset.QueryPoints {
		for _, radius := range radii {
			key := fmt.Sprintf("%s\x00%.6f", queryPoint.Scenario, radius)
			acc := groups[key]
			if acc == nil {
				acc = &queryAccumulator{report: QueryReport{
					Scenario:     queryPoint.Scenario,
					RadiusMeters: radius,
				}}
				groups[key] = acc
			}
			acc.report.Queries++

			expectedCount, boundaryCount, err := oracle.counts(queryPoint.ID, radius)
			if err != nil {
				return nil, err
			}
			acc.report.BoundaryExcludedCount += boundaryCount
			acc.expectedComparable += expectedCount

			first, err := query(ctx, queryPoint, radius)
			if err != nil {
				return nil, fmt.Errorf("query %s at %.0f m: %w", queryPoint.ID, radius, err)
			}
			firstRefinement, err := refineCandidateGeometries(ctx, first.IDs, queryPoint.Point, radius, geometryLoader)
			if err != nil {
				return nil, fmt.Errorf("verify query %s at %.0f m: %w", queryPoint.ID, radius, err)
			}
			firstFinal := firstRefinement.IDs
			firstVerify := firstRefinement.Database + firstRefinement.Exact
			firstTotal := first.WallDuration + firstVerify
			acc.first = append(acc.first, milliseconds(firstTotal))
			candidateTrue, _, err := oracle.classify(queryPoint.ID, radius, first.IDs)
			if err != nil {
				return nil, err
			}
			tp, fp, err := oracle.classify(queryPoint.ID, radius, firstFinal)
			if err != nil {
				return nil, err
			}
			// Fan-out is an operational cost, so retain the raw candidate count.
			// Accuracy classification may deliberately exclude numerical boundary
			// IDs, but database rows and exact checks still pay for them.
			acc.candidateTotal += len(first.IDs)
			acc.matchTotal += tp + fp
			acc.candidateTrue += candidateTrue
			fn := expectedCount - tp
			acc.report.TruePositives += tp
			acc.report.FalsePositives += fp
			acc.report.FalseNegatives += fn

			for range repetitions {
				result, err := query(ctx, queryPoint, radius)
				if err != nil {
					return nil, fmt.Errorf("query %s at %.0f m: %w", queryPoint.ID, radius, err)
				}
				refinement, err := refineCandidateGeometries(ctx, result.IDs, queryPoint.Point, radius, geometryLoader)
				if err != nil {
					return nil, fmt.Errorf("verify query %s at %.0f m: %w", queryPoint.ID, radius, err)
				}
				verifyDuration := refinement.Database + refinement.Exact
				acc.candidate = append(acc.candidate, milliseconds(result.WallDuration))
				acc.verify = append(acc.verify, milliseconds(verifyDuration))
				acc.database = append(acc.database, milliseconds(refinement.Database))
				acc.exact = append(acc.exact, milliseconds(refinement.Exact))
				acc.total = append(acc.total, milliseconds(result.WallDuration+verifyDuration))
				acc.engine = append(acc.engine, result.EngineProcessingMS)
				acc.loadedRows += refinement.LoadedRows
				acc.geometryBytes += refinement.GeometryBytes
				acc.measuredRefines++
				acc.report.Samples++
			}
		}
	}

	reports := make([]QueryReport, 0, len(groups))
	for _, acc := range groups {
		acc.report.FirstSampleMS = mean(acc.first)
		acc.report.CandidateLatency = latencyStats(acc.candidate)
		acc.report.VerifyLatency = latencyStats(acc.verify)
		acc.report.DatabaseLatency = latencyStats(acc.database)
		acc.report.ExactLatency = latencyStats(acc.exact)
		acc.report.TotalLatency = latencyStats(acc.total)
		acc.report.EngineProcessing = latencyStats(acc.engine)
		if acc.report.Queries > 0 {
			acc.report.AverageCandidates = float64(acc.candidateTotal) / float64(acc.report.Queries)
			acc.report.AverageMatches = float64(acc.matchTotal) / float64(acc.report.Queries)
			acc.report.AverageExpected = float64(acc.expectedComparable) / float64(acc.report.Queries)
		}
		if acc.measuredRefines > 0 {
			acc.report.AverageLoadedRows = float64(acc.loadedRows) / float64(acc.measuredRefines)
			acc.report.AverageDatabaseBytes = float64(acc.geometryBytes) / float64(acc.measuredRefines)
		}
		acc.report.CandidateRecall = ratio(acc.candidateTrue, acc.expectedComparable)
		acc.report.ExpectedMatches = acc.expectedComparable
		acc.report.Precision = ratio(acc.report.TruePositives, acc.report.TruePositives+acc.report.FalsePositives)
		acc.report.Recall = ratio(acc.report.TruePositives, acc.report.TruePositives+acc.report.FalseNegatives)
		if acc.report.Precision+acc.report.Recall > 0 {
			acc.report.F1 = 2 * acc.report.Precision * acc.report.Recall / (acc.report.Precision + acc.report.Recall)
		}
		reports = append(reports, acc.report)
	}

	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Scenario != reports[j].Scenario {
			return reports[i].Scenario < reports[j].Scenario
		}
		return reports[i].RadiusMeters < reports[j].RadiusMeters
	})
	return reports, nil
}

func refineCandidateGeometries(
	ctx context.Context,
	candidates map[string]struct{},
	point Coordinate,
	radius float64,
	load candidateGeometryLoader,
) (refinementResult, error) {
	if err := ctx.Err(); err != nil {
		return refinementResult{}, err
	}
	if load == nil {
		return refinementResult{IDs: candidates}, nil
	}

	databaseStarted := time.Now()
	loaded, err := load(ctx, candidates)
	databaseDuration := time.Since(databaseStarted)
	if err != nil {
		return refinementResult{Database: databaseDuration}, err
	}
	if loaded.Rows != len(candidates) {
		return refinementResult{
			Database:      databaseDuration,
			LoadedRows:    loaded.Rows,
			GeometryBytes: loaded.EncodedBytes,
		}, fmt.Errorf("candidate index returned %d IDs but exact-refinement database loaded %d records", len(candidates), loaded.Rows)
	}

	exactStarted := time.Now()
	matches := make(map[string]struct{})
	for id, trail := range loaded.Trails {
		distance, err := pointToTrailDistanceMetersContext(ctx, point, trail)
		if err != nil {
			return refinementResult{Database: databaseDuration, Exact: time.Since(exactStarted)}, err
		}
		if distance <= radius {
			matches[id] = struct{}{}
		}
	}
	return refinementResult{
		IDs:           matches,
		Database:      databaseDuration,
		Exact:         time.Since(exactStarted),
		LoadedRows:    loaded.Rows,
		GeometryBytes: loaded.EncodedBytes,
	}, nil
}

func exactMatches(trails []Trail, point Coordinate, radius float64) (map[string]struct{}, map[string]struct{}) {
	expected := make(map[string]struct{})
	boundary := make(map[string]struct{})
	for _, trail := range trails {
		distance := pointToTrailDistanceMeters(point, trail)
		if isBoundaryDistance(distance, radius) {
			boundary[trail.ID] = struct{}{}
			continue
		}
		if distance < radius {
			expected[trail.ID] = struct{}{}
		}
	}
	return expected, boundary
}

func refineMatches(
	ctx context.Context,
	candidates map[string]struct{},
	trailsByID map[string]Trail,
	point Coordinate,
	radius float64,
	refine bool,
) (map[string]struct{}, time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if !refine {
		return candidates, 0, nil
	}
	start := time.Now()
	matches := make(map[string]struct{})
	for id := range candidates {
		trail, ok := trailsByID[id]
		if !ok {
			continue
		}
		distance, err := pointToTrailDistanceMetersContext(ctx, point, trail)
		if err != nil {
			return nil, time.Since(start), err
		}
		if distance <= radius {
			matches[id] = struct{}{}
		}
	}
	return matches, time.Since(start), nil
}

func intersectionSize(a, b, excluded map[string]struct{}) int {
	count := 0
	for id := range a {
		if _, skip := excluded[id]; skip {
			continue
		}
		if _, ok := b[id]; ok {
			count++
		}
	}
	return count
}

func differenceSize(a, b, excluded map[string]struct{}) int {
	count := 0
	for id := range a {
		if _, skip := excluded[id]; skip {
			continue
		}
		if _, ok := b[id]; !ok {
			count++
		}
	}
	return count
}

func countComparable(values, excluded map[string]struct{}) int {
	if len(excluded) == 0 {
		return len(values)
	}
	count := 0
	for id := range values {
		if _, skip := excluded[id]; !skip {
			count++
		}
	}
	return count
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func completedBackendStatus(report BackendReport) string {
	for _, query := range report.Queries {
		if query.FalsePositives > 0 || query.FalseNegatives > 0 {
			return "incorrect"
		}
	}
	if report.ProductWorkload != nil && report.ProductWorkload.Status != "correct" {
		return "incorrect"
	}
	return "ok"
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func latencyStats(values []float64) LatencyStats {
	if len(values) == 0 {
		return LatencyStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return LatencyStats{
		Min:  sorted[0],
		P50:  percentile(sorted, 0.50),
		P95:  percentile(sorted, 0.95),
		P99:  percentile(sorted, 0.99),
		Max:  sorted[len(sorted)-1],
		Mean: mean(sorted),
	}
}

func percentile(sorted []float64, quantile float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := quantile * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	weight := position - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

type rssMonitor struct {
	sample   func() (int64, error)
	label    string
	baseline int64
	peak     atomic.Int64
	errors   atomic.Int64
	stop     chan struct{}
	done     chan struct{}
}

func startRSSMonitor(pid int) *rssMonitor {
	if pid <= 0 {
		pid = os.Getpid()
	}
	return startRSSSampler("client RSS", func() (int64, error) { return processRSSBytes(pid) })
}

func startContainerRSSMonitor(container *meiliContainer) *rssMonitor {
	return startRSSSampler("Meilisearch RAM", container.memoryBytes)
}

func startRSSSampler(label string, sample func() (int64, error)) *rssMonitor {
	baseline, baselineErr := sample()
	monitor := &rssMonitor{
		sample:   sample,
		label:    label,
		baseline: baseline,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	if baselineErr != nil {
		monitor.errors.Add(1)
	}
	monitor.peak.Store(baseline)
	go func() {
		defer close(monitor.done)
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if rss, err := monitor.sample(); err == nil {
					for current := monitor.peak.Load(); rss > current; current = monitor.peak.Load() {
						if monitor.peak.CompareAndSwap(current, rss) {
							break
						}
					}
				} else {
					monitor.errors.Add(1)
				}
			case <-monitor.stop:
				return
			}
		}
	}()
	return monitor
}

func (monitor *rssMonitor) finish() (baseline, peak, delta int64) {
	close(monitor.stop)
	<-monitor.done
	if rss, err := monitor.sample(); err == nil {
		if rss > monitor.peak.Load() {
			monitor.peak.Store(rss)
		}
	} else {
		monitor.errors.Add(1)
	}
	peak = monitor.peak.Load()
	delta = peak - monitor.baseline
	if delta < 0 {
		delta = 0
	}
	if sampleErrors := monitor.errors.Load(); sampleErrors > 0 {
		log.Printf("warning   %s sampler missed %d sample(s)", monitor.label, sampleErrors)
	}
	return monitor.baseline, peak, delta
}

func processRSSBytes(pid int) (int64, error) {
	file, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "VmRSS:" {
			kilobytes, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			return kilobytes * 1024, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("VmRSS not found for pid %d", pid)
}

func directorySize(ctx context.Context, path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	return total, err
}
