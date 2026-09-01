package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	defaultMeilisearchImages                         = "getmeili/meilisearch:v1.36.0,getmeili/meilisearch:v1.44.0,getmeili/meilisearch:v1.53.1"
	defaultGeoJSONDirectMaximumCountP95RelativeError = 0.01
)

type cliOptions struct {
	profile                 string
	trailCount              int
	queryCount              int
	seed                    int64
	repetitions             int
	batchSize               int
	batchBytesMiB           int
	maxPendingBatches       int
	indexOnly               bool
	meiliIndexThreads       int
	incremental             int
	radii                   string
	h3Resolutions           string
	geoJSONTune             bool
	geoJSONRes              string
	geoJSONPadding          string
	geoJSONFinalists        int
	geoJSONSimplifyMeters   float64
	geoJSONMaxSegmentMeters float64
	geoJSONShardMode        string
	geoJSONShards           int
	geoJSONShardMaxTrails   int
	geoJSONCapacityArrival  string
	geoJSONDirect           bool
	geoJSONDirectRes        string
	geoJSONUXBoundary       float64
	geoJSONUXRecall         float64
	geoJSONUXPrecision      float64
	geoJSONUXTop10          float64
	geoJSONUXPage           float64
	geoJSONUXPagePrecision  float64
	meiliImages             string
	backends                string
	queryWorkload           string
	queryClients            int
	performanceTarget       string
	failOnIncorrect         bool
	failOnUX                bool
	datasetPath             string
	datasetOut              string
	output                  string
	workDir                 string
	keepWorkDir             bool
	contractOnly            bool
	skipContract            bool
	backendTimeout          time.Duration
	timeout                 time.Duration
}

func main() {
	log.SetFlags(0)
	options := parseFlags()
	if err := run(options); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() cliOptions {
	var options cliOptions
	flag.StringVar(&options.profile, "profile", "smoke", "benchmark profile: smoke, standard, stress, or nas")
	flag.IntVar(&options.trailCount, "trails", 0, "number of synthetic trails (profile default when 0)")
	flag.IntVar(&options.queryCount, "queries", 0, "number of query points (profile default when 0)")
	flag.Int64Var(&options.seed, "seed", 42, "deterministic dataset seed")
	flag.IntVar(&options.repetitions, "repetitions", 0, "warm repetitions per query and radius")
	flag.IntVar(&options.batchSize, "batch-size", 100, "Meilisearch document batch size")
	flag.IntVar(&options.batchBytesMiB, "batch-bytes-mib", 16, "maximum encoded Meilisearch request size in MiB (stay below the server payload limit)")
	flag.IntVar(&options.maxPendingBatches, "max-pending-batches", 0, "maximum submitted document tasks before a completion barrier (0 lets Meilisearch autobatch the whole phase)")
	flag.BoolVar(&options.indexOnly, "index-only", false, "stop each backend after the full index build; intended for bounded scaling diagnostics")
	flag.IntVar(&options.meiliIndexThreads, "meili-indexing-threads", 0, "MEILI_MAX_INDEXING_THREADS for isolated containers (0 uses the Meilisearch default)")
	flag.IntVar(&options.incremental, "incremental", 0, "number of trails changed in the incremental run (default 1%)")
	flag.StringVar(&options.radii, "radii", "500,5000,25000,100000", "comma-separated radii in meters")
	flag.StringVar(&options.h3Resolutions, "h3-resolutions", "5,6,7,8", "comma-separated H3 resolutions, low to high")
	flag.BoolVar(&options.geoJSONTune, "geojson-tune", true, "screen GeoJSON radius parameters and fully benchmark safe finalists")
	flag.StringVar(&options.geoJSONRes, "geojson-resolutions", "3,8,16,32,64,100,125,250,500,1000", "comma-separated _geoRadius polygon resolutions (3..1000)")
	flag.StringVar(&options.geoJSONPadding, "geojson-extra-padding-meters", "0,0.5", "comma-separated non-negative padding added after conservative radius expansion")
	flag.IntVar(&options.geoJSONFinalists, "geojson-finalists", 3, "number of safe GeoJSON variants to run through exact and product workloads")
	flag.Float64Var(&options.geoJSONSimplifyMeters, "geojson-simplify-meters", 0, "Ramer-Douglas-Peucker tolerance for indexed GeoJSON only; result geometry and the exact oracle remain unchanged")
	flag.Float64Var(&options.geoJSONMaxSegmentMeters, "geojson-max-segment-meters", defaultGeoJSONMaxSegmentLengthMeters, "maximum spherical segment length after simplification in indexed GeoJSON; long arcs are densified and antimeridian crossings are split")
	flag.StringVar(&options.geoJSONShardMode, "geojson-shard-mode", geoJSONShardModeHash, "GeoJSON shard assignment: hash or capacity-created")
	flag.IntVar(&options.geoJSONShards, "geojson-shards", 1, "number of disjoint GeoJSON indexes in hash mode")
	flag.IntVar(&options.geoJSONShardMaxTrails, "geojson-shard-max-trails", 0, "maximum trails per index in capacity-created mode; shards follow the synthetic local-arrival plan then id")
	flag.StringVar(&options.geoJSONCapacityArrival, "geojson-capacity-arrival", geoJSONCapacityArrivalMixed, "synthetic local trail arrival order for capacity-created mode: mixed or clustered")
	flag.BoolVar(&options.geoJSONDirect, "geojson-direct", true, "benchmark UX-qualified direct GeoJSON variants without exact refinement (ADR release profile: r100)")
	flag.StringVar(&options.geoJSONDirectRes, "geojson-direct-resolutions", "100,125,250", "comma-separated direct GeoJSON resolutions for diagnostic comparison (ADR release profile: 100)")
	flag.Float64Var(&options.geoJSONUXBoundary, "geojson-ux-boundary-meters", 50, "neutral distance band around the requested radius for direct GeoJSON")
	flag.Float64Var(&options.geoJSONUXRecall, "geojson-ux-min-recall", 0.99, "minimum aggregate overall, clear-interior, nearest-result, and non-empty-request success rate for direct GeoJSON")
	flag.Float64Var(&options.geoJSONUXPrecision, "geojson-ux-min-precision", 0.99, "minimum aggregate material precision for direct GeoJSON")
	flag.Float64Var(&options.geoJSONUXTop10, "geojson-ux-min-top10-recall", 0.99, "minimum aggregate nearest-ten coverage for direct GeoJSON")
	flag.Float64Var(&options.geoJSONUXPage, "geojson-ux-min-page-recall", 0.99, "minimum aggregate comparable product-page recall for direct GeoJSON")
	flag.Float64Var(&options.geoJSONUXPagePrecision, "geojson-ux-min-page-precision", 0.99, "minimum aggregate comparable product-page precision for direct GeoJSON")
	flag.StringVar(&options.meiliImages, "meili-images", defaultMeilisearchImages, "comma-separated Meilisearch images")
	flag.StringVar(&options.backends, "backends", "point,geojson,h3,rtree", "comma-separated backends")
	flag.StringVar(&options.queryWorkload, "query-workload", "both", "query workload: engine, product, or both")
	flag.IntVar(&options.queryClients, "query-clients", 0, "parallel clients for the product workload (profile default when 0)")
	flag.StringVar(&options.performanceTarget, "performance-target", "", "optional performance gate: ds923plus (nas profile default) or none")
	flag.BoolVar(&options.failOnIncorrect, "fail-on-incorrect", false, "exit non-zero when a completed backend has correctness mismatches")
	flag.BoolVar(&options.failOnUX, "fail-on-ux", false, "exit non-zero when an enabled direct GeoJSON UX path does not qualify")
	flag.StringVar(&options.datasetPath, "dataset", "", "read a previously generated Dataset JSON file")
	flag.StringVar(&options.datasetOut, "dataset-out", "", "optionally write the generated Dataset JSON")
	flag.StringVar(&options.output, "out", "", "report .json path or output directory (default: a new /tmp directory)")
	flag.StringVar(&options.workDir, "work-dir", "", "parent directory for isolated benchmark data")
	flag.BoolVar(&options.keepWorkDir, "keep-work-dir", false, "keep Meilisearch and SQLite data after the run")
	flag.BoolVar(&options.contractOnly, "contract-only", false, "only run the GeoJSON radius/viewport contract and cell-coverage/hierarchical-split diagnostics")
	flag.BoolVar(&options.skipContract, "skip-contract", false, "skip the GeoJSON contract preflight")
	flag.DurationVar(&options.backendTimeout, "backend-timeout", 15*time.Minute, "timeout for each isolated backend/version run (0 disables)")
	flag.DurationVar(&options.timeout, "timeout", 0, "optional overall benchmark timeout (0 disables)")
	flag.Parse()
	return options
}

func run(options cliOptions) error {
	if options.timeout < 0 || options.backendTimeout < 0 {
		return errors.New("timeout and backend-timeout must not be negative")
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("geo benchmark currently requires Linux for process RSS measurements (got %s)", runtime.GOOS)
	}
	if _, err := processRSSBytes(os.Getpid()); err != nil {
		return fmt.Errorf("read benchmark process RSS: %w", err)
	}
	config, selectedBackends, err := resolveConfig(options)
	if err != nil {
		return err
	}

	var workRoot string
	if options.workDir == "" {
		workRoot, err = os.MkdirTemp("", "wanderer-geobench-work-")
		if err != nil {
			return err
		}
	} else {
		workBase, err := filepath.Abs(options.workDir)
		if err != nil {
			return fmt.Errorf("resolve work directory: %w", err)
		}
		if err := os.MkdirAll(workBase, 0o750); err != nil {
			return err
		}
		workRoot, err = os.MkdirTemp(workBase, "run-")
		if err != nil {
			return fmt.Errorf("create isolated work directory: %w", err)
		}
	}
	if !options.keepWorkDir {
		defer func() {
			if err := os.RemoveAll(workRoot); err != nil {
				log.Printf("warning   remove work directory %s: %v", workRoot, err)
			}
		}()
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := contextWithOptionalTimeout(signalCtx, options.timeout)
	defer cancel()

	report := BenchmarkReport{
		StartedAt:       time.Now().UTC(),
		Config:          config,
		Environment:     benchmarkEnvironment(ctx),
		Reproducibility: initialReproducibility(os.Args),
	}

	contractStatus := make(map[string]ContractReport, len(config.MeilisearchImages))
	runContract := !options.skipContract && (options.contractOnly || slices.Contains(selectedBackends, "geojson"))
	if runContract {
		for _, image := range config.MeilisearchImages {
			if ctx.Err() != nil {
				break
			}
			log.Printf("contract  %-8s  %s", "geojson", image)
			contractCtx, contractCancel := contextWithOptionalTimeout(ctx, options.backendTimeout)
			contract := runGeoJSONContract(contractCtx, image, workRoot)
			contractTimedOut := errors.Is(contractCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
			contractCancel()
			contract = classifyGeoJSONContractResult(contract, contractTimedOut, options.backendTimeout)
			report.Contracts = append(report.Contracts, contract)
			contractStatus[image] = contract
			log.Printf("          %-11s Meilisearch %s", contract.Status, contract.MeilisearchVersion)
		}
	}

	if options.contractOnly {
		addDockerImageMetadata(&report)
		jsonPath, markdownPath, err := writeReports(report, options.output)
		if err != nil {
			return err
		}
		log.Printf("report    %s", jsonPath)
		log.Printf("summary   %s", markdownPath)
		if options.keepWorkDir {
			log.Printf("work dir  %s", workRoot)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("contract run stopped: %w; partial report was written", err)
		}
		if err := geoJSONContractError(report.Contracts); err != nil {
			return err
		}
		return nil
	}

	dataset, err := loadOrGenerateDataset(ctx, options, config)
	if err != nil {
		return err
	}
	if err := validateDataset(dataset); err != nil {
		return err
	}
	addDatasetDigest(&report, dataset)
	report.Dataset = summarizeDataset(dataset)
	config.Seed = dataset.Seed
	config.TrailCount = len(dataset.Trails)
	config.QueryCount = len(dataset.QueryPoints)
	if slices.Contains(selectedBackends, "geojson") {
		if _, err := validateGeoJSONShardRun(config.GeoJSONIndex, len(dataset.Trails), config.IndexOnly); err != nil {
			return err
		}
	}
	if options.incremental <= 0 {
		config.IncrementalCount = max(1, len(dataset.Trails)/100)
	}
	config.IncrementalCount = min(config.IncrementalCount, len(dataset.Trails))
	capacitySearchOnly := !config.IndexOnly && slices.Contains(selectedBackends, "geojson") &&
		normalizedGeoJSONShardMode(config.GeoJSONIndex.ShardMode) == geoJSONShardModeCapacityCreated
	if capacitySearchOnly {
		config.IncrementalCount = 0
	}
	report.Config = config
	var updates []Trail
	if !config.IndexOnly && !capacitySearchOnly {
		updates = mutateTrails(dataset.Trails, config.IncrementalCount)
	}

	var oracle *accuracyOracle
	needsOracle := !config.IndexOnly && (capacitySearchOnly ||
		config.QueryWorkload != "product" ||
		(config.GeoJSONSweep.Enabled && slices.Contains(selectedBackends, "geojson")))
	if needsOracle {
		log.Printf("reference %-8s  exact spherical point-to-polyline", "oracle")
		oracleStarted := time.Now()
		oracle, err = buildAccuracyOracle(ctx, dataset, config.RadiiMeters)
		report.OracleDurationMS = milliseconds(time.Since(oracleStarted))
		if err != nil {
			return fmt.Errorf("build exact accuracy reference: %w", err)
		}
		log.Printf("          ready       %.0f ms", report.OracleDurationMS)
	}

	if options.datasetOut != "" && options.datasetPath == "" {
		if err := writeJSON(options.datasetOut, dataset); err != nil {
			return fmt.Errorf("write dataset: %w", err)
		}
	}

backendLoop:
	for _, image := range config.MeilisearchImages {
		for _, backend := range []string{"point", "h3", "geojson"} {
			if ctx.Err() != nil {
				break backendLoop
			}
			if !slices.Contains(selectedBackends, backend) {
				continue
			}
			if backend == "geojson" && runContract {
				contract := contractStatus[image]
				if contract.Status != "passed" {
					report.Backends = append(report.Backends, BackendReport{
						Backend:            backend,
						MeilisearchVersion: contract.MeilisearchVersion,
						Status:             contract.Status,
						Error:              contract.Error,
					})
					continue
				}
			}
			debug.FreeOSMemory()
			log.Printf("benchmark %-8s  %s", backend, image)
			backendReport := runBackendWithTimeout(ctx, options.backendTimeout, func(backendCtx context.Context) BackendReport {
				return runMeiliBackend(backendCtx, image, backend, workRoot, dataset, updates, config, oracle)
			})
			report.Backends = append(report.Backends, backendReport)
			log.Printf("          %-11s elapsed %.0f ms, full %.0f ms, update %.0f ms, add %.0f ms, delete %.0f ms", backendReport.Status, backendReport.ElapsedMS, phaseTotalMS(backendReport.FullIndex), phaseTotalMS(backendReport.IncrementalIndex), phaseTotalMS(backendReport.AddIndex), phaseTotalMS(backendReport.DeleteIndex))
			if backendReport.ProductWorkload != nil {
				log.Printf("          product     %d/%d correct, p95 %.1f ms, %.1f q/s", backendReport.ProductWorkload.CorrectCases, backendReport.ProductWorkload.Cases, backendReport.ProductWorkload.TotalLatency.P95, backendReport.ProductWorkload.ThroughputQPS)
			}
			if backendReport.GeoJSONSweep != nil {
				log.Printf("          geojson     selected %s", emptyDash(backendReport.GeoJSONSweep.SelectedID))
			}
			if backendReport.GeoJSONDirect != nil {
				log.Printf("          direct      %s (%s)", emptyDash(backendReport.GeoJSONDirect.SelectedID), backendReport.GeoJSONDirect.SelectionStatus)
				if workload := backendReport.GeoJSONDirect.ProductWorkload; workload != nil {
					log.Printf("          direct UX   %s (accuracy %s, count %s), p95 %.1f ms, %.1f q/s", workload.Status, workload.Accuracy.Status, workload.CountDiagnostics.Status, workload.TotalLatency.P95, workload.ThroughputQPS)
				}
			}
			if backendReport.Performance != nil {
				log.Printf("          target      %s: %s", backendReport.Performance.Target, backendReport.Performance.Status)
			}
			if backendReport.GeoJSONDirect != nil && backendReport.GeoJSONDirect.Performance != nil {
				log.Printf("          direct SLO  %s: %s", backendReport.GeoJSONDirect.Performance.Target, backendReport.GeoJSONDirect.Performance.Status)
			}
		}
	}

	if ctx.Err() == nil && slices.Contains(selectedBackends, "rtree") {
		debug.FreeOSMemory()
		log.Printf("benchmark %-8s", "rtree")
		backendReport := runBackendWithTimeout(ctx, options.backendTimeout, func(backendCtx context.Context) BackendReport {
			return runRTreeBenchmark(backendCtx, workRoot, dataset, updates, config, oracle)
		})
		report.Backends = append(report.Backends, backendReport)
		log.Printf("          %-11s elapsed %.0f ms, full %.0f ms, update %.0f ms, add %.0f ms, delete %.0f ms", backendReport.Status, backendReport.ElapsedMS, phaseTotalMS(backendReport.FullIndex), phaseTotalMS(backendReport.IncrementalIndex), phaseTotalMS(backendReport.AddIndex), phaseTotalMS(backendReport.DeleteIndex))
		if backendReport.ProductWorkload != nil {
			log.Printf("          product     %d/%d correct, p95 %.1f ms, %.1f q/s", backendReport.ProductWorkload.CorrectCases, backendReport.ProductWorkload.Cases, backendReport.ProductWorkload.TotalLatency.P95, backendReport.ProductWorkload.ThroughputQPS)
		}
		if backendReport.Performance != nil {
			log.Printf("          target      %s: %s", backendReport.Performance.Target, backendReport.Performance.Status)
		}
	}

	addDockerImageMetadata(&report)
	jsonPath, markdownPath, err := writeReports(report, options.output)
	if err != nil {
		return err
	}
	log.Printf("report    %s", jsonPath)
	log.Printf("summary   %s", markdownPath)
	if options.keepWorkDir {
		log.Printf("work dir  %s", workRoot)
	}

	failed := 0
	timedOut := 0
	incorrect := 0
	performanceNotPassed := 0
	uxNotQualified := 0
	for _, backend := range report.Backends {
		switch backend.Status {
		case "failed":
			failed++
		case "timed_out":
			timedOut++
		case "incorrect":
			incorrect++
		}
		if backend.Performance != nil {
			if !backendPerformanceQualified(backend) {
				performanceNotPassed++
			}
		}
		if !config.IndexOnly && config.GeoJSONDirect.Enabled && backend.Backend == "geojson" &&
			!geoJSONDirectTargetQualified(
				backend.GeoJSONDirect,
				config.QueryWorkload != "engine",
				config.PerformanceTarget != "none" && config.PerformanceTarget != "",
			) {
			uxNotQualified++
		}
	}
	if timedOut > 0 {
		log.Printf("warning   %d backend(s) reached their time budget; unfinished phase times are lower bounds", timedOut)
	}
	if incorrect > 0 {
		log.Printf("warning   %d backend(s) completed with correctness mismatches", incorrect)
	}
	if performanceNotPassed > 0 {
		log.Printf("warning   %d backend(s) did not pass the %s performance target", performanceNotPassed, config.PerformanceTarget)
	}
	if uxNotQualified > 0 {
		log.Printf("warning   %d GeoJSON backend(s) did not qualify for the direct UX contract", uxNotQualified)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("overall benchmark stopped: %w; partial report was written", err)
	}
	if failed > 0 {
		return fmt.Errorf("%d backend(s) failed; partial report was written", failed)
	}
	if timedOut > 0 {
		return fmt.Errorf("%d backend(s) timed out; partial report was written", timedOut)
	}
	if options.failOnIncorrect && incorrect > 0 {
		return fmt.Errorf("%d backend(s) had correctness mismatches; report was written", incorrect)
	}
	if options.failOnUX && uxNotQualified > 0 {
		return fmt.Errorf("%d GeoJSON backend(s) did not qualify for the direct UX contract; report was written", uxNotQualified)
	}
	if config.PerformanceTarget != "none" && performanceNotPassed > 0 {
		return fmt.Errorf("%d backend(s) did not pass the %s performance target; report was written", performanceNotPassed, config.PerformanceTarget)
	}
	return nil
}

func geoJSONDirectQualified(report *GeoJSONDirectReport, requireProduct bool) bool {
	if report == nil || report.Plan == nil || report.Accuracy == nil || report.Accuracy.Status != "passed" {
		return false
	}
	if !requireProduct {
		return report.SelectionStatus == "spatial_qualified" || strings.HasPrefix(report.SelectionStatus, "qualified")
	}
	return report.ProductWorkload != nil && report.ProductWorkload.Status == "passed" &&
		strings.HasPrefix(report.SelectionStatus, "qualified")
}

func geoJSONDirectTargetQualified(report *GeoJSONDirectReport, requireProduct, requirePerformance bool) bool {
	if !geoJSONDirectQualified(report, requireProduct) {
		return false
	}
	return !requirePerformance || (report.Performance != nil && report.Performance.Status == "passed")
}

// A GeoJSON backend exposes both the conservative exact path and, when
// enabled, the direct UX path. The NAS gate is satisfied when either complete
// product architecture passes; --fail-on-ux independently requires the direct
// architecture when it is requested.
func backendPerformanceQualified(backend BackendReport) bool {
	if backend.Performance == nil {
		return true
	}
	if backend.Performance.Status == "passed" {
		return true
	}
	return backend.Backend == "geojson" &&
		geoJSONDirectTargetQualified(backend.GeoJSONDirect, true, true)
}

func contextWithOptionalTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

func runBackendWithTimeout(
	parent context.Context,
	timeout time.Duration,
	runBackend func(context.Context) BackendReport,
) BackendReport {
	ctx, cancel := contextWithOptionalTimeout(parent, timeout)
	started := time.Now()
	report := runBackend(ctx)
	report.ElapsedMS = milliseconds(time.Since(started))
	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded) && parent.Err() == nil
	cancel()
	if timedOut && report.Status != "ok" {
		report.Status = "timed_out"
		report.Error = timeoutMessage(timeout, report.Error)
	}
	return report
}

func timeoutMessage(timeout time.Duration, detail string) string {
	message := fmt.Sprintf("time budget %s reached", timeout)
	if detail != "" {
		message += ": " + detail
	}
	return message
}

func resolveConfig(options cliOptions) (RunConfig, []string, error) {
	if options.contractOnly && options.skipContract {
		return RunConfig{}, nil, errors.New("contract-only and skip-contract cannot be combined")
	}
	if options.batchBytesMiB == 0 {
		options.batchBytesMiB = 16
	}
	// cliOptions is also constructed directly by focused tests. Preserve the
	// historical zero value there while the command-line flag defaults to one.
	if options.geoJSONShards == 0 {
		options.geoJSONShards = 1
	}
	if options.trailCount < 0 || options.queryCount < 0 || options.repetitions < 0 || options.incremental < 0 || options.queryClients < 0 || options.geoJSONFinalists < 0 || options.maxPendingBatches < 0 || options.meiliIndexThreads < 0 {
		return RunConfig{}, nil, errors.New("trails, queries, repetitions, incremental, query-clients, geojson-finalists, max-pending-batches, and meili-indexing-threads must not be negative")
	}
	shardMode := options.geoJSONShardMode
	if shardMode == "" {
		shardMode = geoJSONShardModeHash
	}
	if !slices.Contains([]string{geoJSONShardModeHash, geoJSONShardModeCapacityCreated}, shardMode) {
		return RunConfig{}, nil, fmt.Errorf("unknown geojson-shard-mode %q (want %s or %s)", shardMode, geoJSONShardModeHash, geoJSONShardModeCapacityCreated)
	}
	capacityArrival := options.geoJSONCapacityArrival
	if capacityArrival == "" {
		capacityArrival = geoJSONCapacityArrivalMixed
	}
	if !slices.Contains([]string{geoJSONCapacityArrivalMixed, geoJSONCapacityArrivalClustered}, capacityArrival) {
		return RunConfig{}, nil, fmt.Errorf("unknown geojson-capacity-arrival %q (want %s or %s)", capacityArrival, geoJSONCapacityArrivalMixed, geoJSONCapacityArrivalClustered)
	}
	if options.geoJSONShards < 1 || options.geoJSONShards > geoJSONMaxShards {
		return RunConfig{}, nil, fmt.Errorf("geojson-shards must be in 1..%d", geoJSONMaxShards)
	}
	if shardMode == geoJSONShardModeHash {
		if options.geoJSONShardMaxTrails != 0 {
			return RunConfig{}, nil, errors.New("geojson-shard-max-trails is only valid with --geojson-shard-mode capacity-created")
		}
		if options.geoJSONShards > 1 && !options.indexOnly {
			return RunConfig{}, nil, errors.New("geojson-shards greater than 1 currently require --index-only; federated query semantics are not part of this index-build experiment")
		}
	} else {
		if options.geoJSONShards != 1 {
			return RunConfig{}, nil, errors.New("geojson-shards is not used in capacity-created mode; leave it at 1 and set --geojson-shard-max-trails")
		}
		if options.geoJSONShardMaxTrails < 1 {
			return RunConfig{}, nil, errors.New("capacity-created mode requires --geojson-shard-max-trails greater than zero")
		}
	}
	if options.batchBytesMiB < 1 || options.batchBytesMiB > 99 {
		return RunConfig{}, nil, errors.New("batch-bytes-mib must be in 1..99")
	}
	if math.IsNaN(options.geoJSONSimplifyMeters) || math.IsInf(options.geoJSONSimplifyMeters, 0) || options.geoJSONSimplifyMeters < 0 {
		return RunConfig{}, nil, errors.New("geojson-simplify-meters must be a finite non-negative value")
	}
	if options.geoJSONMaxSegmentMeters == 0 {
		options.geoJSONMaxSegmentMeters = defaultGeoJSONMaxSegmentLengthMeters
	}
	if math.IsNaN(options.geoJSONMaxSegmentMeters) || math.IsInf(options.geoJSONMaxSegmentMeters, 0) || options.geoJSONMaxSegmentMeters <= 0 {
		return RunConfig{}, nil, errors.New("geojson-max-segment-meters must be a finite positive value")
	}
	if options.geoJSONRes == "" {
		options.geoJSONRes = "3,8,16,32,64,100,125,250,500,1000"
	}
	if options.geoJSONPadding == "" {
		options.geoJSONPadding = "0,0.5"
	}
	if options.geoJSONFinalists == 0 {
		options.geoJSONFinalists = 3
	}
	queryWorkload := options.queryWorkload
	if queryWorkload == "" {
		queryWorkload = "both"
	}
	if !slices.Contains([]string{"engine", "product", "both"}, queryWorkload) {
		return RunConfig{}, nil, fmt.Errorf("unknown query workload %q (want engine, product, or both)", queryWorkload)
	}
	config := RunConfig{
		Profile:           options.profile,
		Seed:              options.seed,
		BatchSize:         options.batchSize,
		BatchBytes:        options.batchBytesMiB << 20,
		MaxPendingBatches: options.maxPendingBatches,
		IndexOnly:         options.indexOnly,
		MeiliIndexThreads: options.meiliIndexThreads,
		QueryWorkload:     queryWorkload,
		FailOnIncorrect:   options.failOnIncorrect,
		FailOnUX:          options.failOnUX,
		BackendTimeout:    durationLabel(options.backendTimeout),
		OverallTimeout:    durationLabel(options.timeout),
	}
	config.GeoJSONIndex.SimplifyToleranceMeters = options.geoJSONSimplifyMeters
	config.GeoJSONIndex.MaxSegmentLengthMeters = options.geoJSONMaxSegmentMeters
	config.GeoJSONIndex.ShardMode = shardMode
	config.GeoJSONIndex.Shards = options.geoJSONShards
	config.GeoJSONIndex.MaxTrailsPerShard = options.geoJSONShardMaxTrails
	if shardMode == geoJSONShardModeCapacityCreated {
		config.GeoJSONIndex.CapacityArrival = capacityArrival
	}
	switch options.profile {
	case "smoke":
		config.TrailCount = 500
		config.QueryCount = 24
		config.Repetitions = 2
		config.QueryClients = 2
	case "standard":
		config.TrailCount = 5_000
		config.QueryCount = 36
		config.Repetitions = 2
		config.QueryClients = 8
	case "stress":
		config.TrailCount = 25_000
		config.QueryCount = 60
		config.Repetitions = 3
		config.QueryClients = 16
	case "nas":
		config.TrailCount = 50_000
		config.QueryCount = 60
		// 60 cases times 84 repetitions provides 5,040 warm product
		// samples, enough to make p99 more than a decorative statistic.
		config.Repetitions = 84
		config.QueryClients = 4
	default:
		return RunConfig{}, nil, fmt.Errorf("unknown profile %q (want smoke, standard, stress, or nas)", options.profile)
	}
	if options.trailCount > 0 {
		config.TrailCount = options.trailCount
	}
	if options.queryCount > 0 {
		config.QueryCount = options.queryCount
	}
	if options.repetitions > 0 {
		config.Repetitions = options.repetitions
	}
	if options.queryClients > 0 {
		config.QueryClients = options.queryClients
	}
	config.IncrementalCount = options.incremental

	var err error
	config.RadiiMeters, err = parseFloatList(options.radii)
	if err != nil {
		return RunConfig{}, nil, fmt.Errorf("radii: %w", err)
	}
	config.H3Resolutions, err = parseIntList(options.h3Resolutions)
	if err != nil {
		return RunConfig{}, nil, fmt.Errorf("H3 resolutions: %w", err)
	}
	for _, resolution := range config.H3Resolutions {
		if resolution < 0 || resolution > 15 {
			return RunConfig{}, nil, fmt.Errorf("H3 resolution %d is outside 0..15", resolution)
		}
	}
	config.GeoJSONSweep.Enabled = options.geoJSONTune
	config.GeoJSONSweep.IndexSimplificationMeters = config.GeoJSONIndex.SimplifyToleranceMeters
	config.GeoJSONSweep.Resolutions, err = parseIntList(options.geoJSONRes)
	if err != nil {
		return RunConfig{}, nil, fmt.Errorf("GeoJSON resolutions: %w", err)
	}
	for _, resolution := range config.GeoJSONSweep.Resolutions {
		if resolution < 3 || resolution > 1000 {
			return RunConfig{}, nil, fmt.Errorf("GeoJSON resolution %d is outside 3..1000", resolution)
		}
	}
	config.GeoJSONSweep.ExtraPaddingMeters, err = parseNonNegativeFloatList(options.geoJSONPadding)
	if err != nil {
		return RunConfig{}, nil, fmt.Errorf("GeoJSON extra padding: %w", err)
	}
	config.GeoJSONSweep.Finalists = options.geoJSONFinalists
	variantCount := len(geoJSONQueryPlans(config.GeoJSONSweep))
	if variantCount > 32 {
		return RunConfig{}, nil, fmt.Errorf("GeoJSON sweep expands to %d variants; maximum is 32", variantCount)
	}
	if options.geoJSONDirect && !config.GeoJSONSweep.Enabled {
		return RunConfig{}, nil, errors.New("GeoJSON direct requires --geojson-tune=true; disable it explicitly with --geojson-direct=false for the legacy unscreened run")
	}
	config.GeoJSONDirect.Enabled = options.geoJSONDirect
	if config.GeoJSONDirect.Enabled {
		if options.geoJSONDirectRes == "" {
			options.geoJSONDirectRes = "100,125,250"
		}
		config.GeoJSONDirect.Resolutions, err = parseIntList(options.geoJSONDirectRes)
		if err != nil {
			return RunConfig{}, nil, fmt.Errorf("GeoJSON direct resolutions: %w", err)
		}
		for _, resolution := range config.GeoJSONDirect.Resolutions {
			if resolution < 3 || resolution > 1000 {
				return RunConfig{}, nil, fmt.Errorf("GeoJSON direct resolution %d is outside 3..1000", resolution)
			}
			if !slices.Contains(config.GeoJSONSweep.Resolutions, resolution) {
				return RunConfig{}, nil, fmt.Errorf("GeoJSON direct resolution %d is not present in geojson-resolutions", resolution)
			}
		}
		if len(config.GeoJSONDirect.Resolutions) == 0 {
			return RunConfig{}, nil, errors.New("GeoJSON direct requires at least one resolution")
		}
		if !finiteUnitInterval(options.geoJSONUXRecall) || !finiteUnitInterval(options.geoJSONUXPrecision) || !finiteUnitInterval(options.geoJSONUXTop10) || !finiteUnitInterval(options.geoJSONUXPage) || !finiteUnitInterval(options.geoJSONUXPagePrecision) {
			return RunConfig{}, nil, errors.New("GeoJSON UX thresholds must be finite values in 0..1")
		}
		if math.IsNaN(options.geoJSONUXBoundary) || math.IsInf(options.geoJSONUXBoundary, 0) || options.geoJSONUXBoundary < 0 {
			return RunConfig{}, nil, errors.New("GeoJSON UX boundary must be a finite non-negative distance")
		}
		config.GeoJSONDirect.BoundaryToleranceMeters = options.geoJSONUXBoundary
		config.GeoJSONDirect.MinimumRecall = options.geoJSONUXRecall
		config.GeoJSONDirect.MinimumPrecision = options.geoJSONUXPrecision
		config.GeoJSONDirect.MinimumTop10Recall = options.geoJSONUXTop10
		config.GeoJSONDirect.MinimumComparablePageRate = options.geoJSONUXPage
		config.GeoJSONDirect.MinimumComparablePagePrecision = options.geoJSONUXPagePrecision
		config.GeoJSONDirect.MaximumCountP95RelativeError = defaultGeoJSONDirectMaximumCountP95RelativeError
	}
	config.PerformanceTarget = options.performanceTarget
	if config.PerformanceTarget == "" {
		if config.IndexOnly {
			config.PerformanceTarget = "none"
		} else if options.profile == "nas" {
			config.PerformanceTarget = "ds923plus"
		} else {
			config.PerformanceTarget = "none"
		}
	}
	if !slices.Contains([]string{"none", "ds923plus"}, config.PerformanceTarget) {
		return RunConfig{}, nil, fmt.Errorf("unknown performance target %q (want none or ds923plus)", config.PerformanceTarget)
	}
	if config.IndexOnly && config.PerformanceTarget != "none" {
		return RunConfig{}, nil, errors.New("index-only requires --performance-target none because query gates are intentionally skipped")
	}
	config.MeilisearchImages = splitList(options.meiliImages)
	selectedBackends := splitList(options.backends)
	if options.contractOnly && len(config.MeilisearchImages) == 0 {
		return RunConfig{}, nil, errors.New("contract-only requires at least one Meilisearch image")
	}
	if len(config.MeilisearchImages) == 0 && containsAny(selectedBackends, "point", "geojson", "h3") {
		return RunConfig{}, nil, errors.New("at least one Meilisearch image is required")
	}
	for _, backend := range selectedBackends {
		if !slices.Contains([]string{"point", "geojson", "h3", "rtree"}, backend) {
			return RunConfig{}, nil, fmt.Errorf("unknown backend %q", backend)
		}
	}
	if shardMode == geoJSONShardModeCapacityCreated && !config.IndexOnly {
		if len(selectedBackends) != 1 || selectedBackends[0] != "geojson" {
			return RunConfig{}, nil, errors.New("capacity-created search is an isolated GeoJSON cell and requires --backends geojson")
		}
		if config.QueryWorkload != "product" {
			return RunConfig{}, nil, errors.New("capacity-created search requires --query-workload product; engine, safe-hybrid, and mutation workloads are intentionally excluded")
		}
		requiredRadii := []float64{500, 5_000, 25_000, 100_000}
		actualRadii := append([]float64(nil), config.RadiiMeters...)
		slices.Sort(actualRadii)
		if !slices.Equal(actualRadii, requiredRadii) {
			return RunConfig{}, nil, fmt.Errorf("capacity-created search requires the complete spatial audit radii %v meters", requiredRadii)
		}
		if !config.GeoJSONDirect.Enabled || len(config.GeoJSONDirect.Resolutions) != 1 {
			return RunConfig{}, nil, errors.New("capacity-created search requires exactly one direct GeoJSON resolution (--geojson-direct=true --geojson-direct-resolutions N)")
		}
		if len(config.MeilisearchImages) != 1 {
			return RunConfig{}, nil, errors.New("capacity-created exhaustive federated search requires exactly one official Meilisearch release image")
		}
		if err := validateMeiliQualificationImage(config.MeilisearchImages[0]); err != nil {
			return RunConfig{}, nil, fmt.Errorf("capacity-created exhaustive federated search image: %w", err)
		}
		// The capacity query run has one pinned resolution and does not spend
		// time on the single-index parameter sweep represented by this flag.
		config.GeoJSONSweep.Enabled = false
	}
	if len(selectedBackends) == 0 && !options.contractOnly {
		return RunConfig{}, nil, errors.New("at least one backend is required")
	}
	if config.TrailCount < 3 || config.QueryCount < 3 || config.BatchSize < 1 || config.Repetitions < 1 {
		return RunConfig{}, nil, errors.New("trails and queries must be >= 3; batch size and repetitions must be >= 1")
	}
	return config, selectedBackends, nil
}

func durationLabel(duration time.Duration) string {
	if duration <= 0 {
		return "disabled"
	}
	return duration.String()
}

func finiteUnitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func geoJSONContractError(contracts []ContractReport) error {
	for _, contract := range contracts {
		if contract.Status != "passed" {
			return fmt.Errorf("GeoJSON contract failed for Meilisearch %s: %s", contract.MeilisearchVersion, contract.Error)
		}
		for _, diagnostic := range contract.Diagnostics {
			if diagnostic.Status == "known_engine_limitation" || diagnostic.Status == "passed" {
				continue
			}
			detail := diagnostic.Error
			if detail == "" {
				detail = diagnostic.Status
			}
			return fmt.Errorf("GeoJSON contract diagnostic %s failed for Meilisearch %s: %s", diagnostic.Name, contract.MeilisearchVersion, detail)
		}
	}
	return nil
}

func classifyGeoJSONContractResult(contract ContractReport, timedOut bool, timeout time.Duration) ContractReport {
	gateErr := geoJSONContractError([]ContractReport{contract})
	if gateErr == nil {
		return contract
	}
	if timedOut {
		detail := contract.Error
		if detail == "" {
			detail = gateErr.Error()
		}
		contract.Status = "timed_out"
		contract.Error = timeoutMessage(timeout, detail)
		return contract
	}
	if contract.Status == "passed" {
		contract.Status = "failed"
		contract.Error = gateErr.Error()
	}
	return contract
}

func loadOrGenerateDataset(ctx context.Context, options cliOptions, config RunConfig) (Dataset, error) {
	if options.datasetPath == "" {
		return generateDatasetContext(ctx, config.TrailCount, config.QueryCount, config.Seed)
	}
	if err := ctx.Err(); err != nil {
		return Dataset{}, err
	}
	file, err := os.Open(options.datasetPath)
	if err != nil {
		return Dataset{}, fmt.Errorf("open dataset: %w", err)
	}
	defer file.Close()
	var dataset Dataset
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&dataset); err != nil {
		return Dataset{}, fmt.Errorf("decode dataset: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Dataset{}, err
	}
	return dataset, nil
}

func validateDataset(dataset Dataset) error {
	if len(dataset.Trails) < 1 || len(dataset.QueryPoints) < 1 {
		return errors.New("dataset needs at least one trail and one query point")
	}
	ids := make(map[string]struct{}, len(dataset.Trails))
	for _, trail := range dataset.Trails {
		if trail.ID == "" {
			return errors.New("dataset contains a trail without id")
		}
		if _, exists := ids[trail.ID]; exists {
			return fmt.Errorf("duplicate trail id %q", trail.ID)
		}
		ids[trail.ID] = struct{}{}
		if trail.Scenario == "" {
			return fmt.Errorf("trail %q has no scenario", trail.ID)
		}
		segments := 0
		for partIndex, part := range trail.Parts {
			if len(part) == 1 {
				return fmt.Errorf("trail %q part %d has one coordinate; a line needs at least two", trail.ID, partIndex)
			}
			if len(part) >= 2 {
				segments += len(part) - 1
			}
			for _, coordinate := range part {
				if !validCoordinate(coordinate) {
					return fmt.Errorf("trail %q contains invalid coordinate %+v", trail.ID, coordinate)
				}
			}
		}
		if segments == 0 {
			return fmt.Errorf("trail %q has no line segment", trail.ID)
		}
	}
	queryIDs := make(map[string]struct{}, len(dataset.QueryPoints))
	for _, queryPoint := range dataset.QueryPoints {
		if queryPoint.ID == "" {
			return errors.New("dataset contains a query point without id")
		}
		if _, exists := queryIDs[queryPoint.ID]; exists {
			return fmt.Errorf("duplicate query point id %q", queryPoint.ID)
		}
		queryIDs[queryPoint.ID] = struct{}{}
		if queryPoint.Scenario == "" {
			return fmt.Errorf("query point %q has no scenario", queryPoint.ID)
		}
		if !validCoordinate(queryPoint.Point) {
			return fmt.Errorf("query point %q contains invalid coordinate %+v", queryPoint.ID, queryPoint.Point)
		}
	}
	return nil
}

func validCoordinate(coordinate Coordinate) bool {
	return !math.IsNaN(coordinate.Lat) && !math.IsInf(coordinate.Lat, 0) &&
		!math.IsNaN(coordinate.Lon) && !math.IsInf(coordinate.Lon, 0) &&
		coordinate.Lat >= -90 && coordinate.Lat <= 90 &&
		coordinate.Lon >= -180 && coordinate.Lon <= 180
}

func parseFloatList(value string) ([]float64, error) {
	parts := splitList(value)
	values := make([]float64, 0, len(parts))
	seen := make(map[float64]struct{}, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseFloat(part, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed <= 0 {
			return nil, fmt.Errorf("invalid positive number %q", part)
		}
		if _, exists := seen[parsed]; !exists {
			seen[parsed] = struct{}{}
			values = append(values, parsed)
		}
	}
	if len(values) == 0 {
		return nil, errors.New("list is empty")
	}
	slices.Sort(values)
	return values, nil
}

func parseNonNegativeFloatList(value string) ([]float64, error) {
	parts := splitList(value)
	values := make([]float64, 0, len(parts))
	seen := make(map[float64]struct{}, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseFloat(part, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 {
			return nil, fmt.Errorf("invalid non-negative number %q", part)
		}
		if _, exists := seen[parsed]; !exists {
			seen[parsed] = struct{}{}
			values = append(values, parsed)
		}
	}
	if len(values) == 0 {
		return nil, errors.New("list is empty")
	}
	slices.Sort(values)
	return values, nil
}

func parseIntList(value string) ([]int, error) {
	parts := splitList(value)
	values := make([]int, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", part)
		}
		values = append(values, parsed)
	}
	if len(values) == 0 {
		return nil, errors.New("list is empty")
	}
	return sortedUniqueInts(values), nil
}

func splitList(value string) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" || item == "none" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func containsAny(values []string, wanted ...string) bool {
	for _, value := range wanted {
		if slices.Contains(values, value) {
			return true
		}
	}
	return false
}

func benchmarkEnvironment(ctx context.Context) map[string]string {
	return collectBenchmarkEnvironment(ctx)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o640)
}
