package main

import (
	"context"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestValidateDatasetRejectsInvalidQueryPoints(t *testing.T) {
	base := Dataset{
		Trails:      []Trail{{ID: "trail", Scenario: "test", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}}}}},
		QueryPoints: []QueryPoint{{ID: "query", Scenario: "test", Point: Coordinate{Lat: 0, Lon: 0}}},
	}

	tests := []struct {
		name   string
		mutate func(*Dataset)
		want   string
	}{
		{name: "empty id", mutate: func(dataset *Dataset) { dataset.QueryPoints[0].ID = "" }, want: "without id"},
		{name: "duplicate id", mutate: func(dataset *Dataset) { dataset.QueryPoints = append(dataset.QueryPoints, dataset.QueryPoints[0]) }, want: "duplicate query point"},
		{name: "nan", mutate: func(dataset *Dataset) { dataset.QueryPoints[0].Point.Lat = math.NaN() }, want: "invalid coordinate"},
		{name: "infinity", mutate: func(dataset *Dataset) { dataset.QueryPoints[0].Point.Lon = math.Inf(1) }, want: "invalid coordinate"},
		{name: "singleton part", mutate: func(dataset *Dataset) {
			dataset.Trails[0].Parts = append(dataset.Trails[0].Parts, []Coordinate{{Lat: 1, Lon: 1}})
		}, want: "one coordinate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dataset := base
			dataset.Trails = append([]Trail(nil), base.Trails...)
			dataset.QueryPoints = append([]QueryPoint(nil), base.QueryPoints...)
			test.mutate(&dataset)
			err := validateDataset(dataset)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateDataset error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestResolveConfigSmokeDefaults(t *testing.T) {
	config, backends, err := resolveConfig(cliOptions{
		profile:       "smoke",
		batchSize:     100,
		radii:         "500,5000,25000,100000",
		h3Resolutions: "5,6,7,8",
		meiliImages:   "getmeili/meilisearch:v1.36.0",
		backends:      "point,geojson,h3,rtree",
	})
	if err != nil {
		t.Fatalf("resolve smoke defaults: %v", err)
	}
	if config.TrailCount != 500 || config.QueryCount != 24 || config.Repetitions != 2 || config.QueryWorkload != "both" || config.QueryClients != 2 {
		t.Fatalf("unexpected smoke config: %+v", config)
	}
	if config.GeoJSONIndex.MaxSegmentLengthMeters != defaultGeoJSONMaxSegmentLengthMeters {
		t.Fatalf("default GeoJSON maximum segment length = %v", config.GeoJSONIndex.MaxSegmentLengthMeters)
	}
	if len(backends) != 4 {
		t.Fatalf("backends = %v", backends)
	}
}

func TestResolveConfigProductWorkloadAndClients(t *testing.T) {
	base := cliOptions{
		profile:       "standard",
		batchSize:     100,
		radii:         "500,5000,25000,100000",
		h3Resolutions: "5,6,7,8",
		backends:      "rtree",
		queryWorkload: "product",
		queryClients:  11,
	}
	config, _, err := resolveConfig(base)
	if err != nil {
		t.Fatalf("resolve product workload: %v", err)
	}
	if config.QueryWorkload != "product" || config.QueryClients != 11 {
		t.Fatalf("query config = %+v", config)
	}
	base.queryWorkload = "almost"
	if _, _, err := resolveConfig(base); err == nil || !strings.Contains(err.Error(), "query workload") {
		t.Fatalf("invalid workload error = %v", err)
	}
}

func TestResolveConfigIndexOnlyOptimization(t *testing.T) {
	config, backends, err := resolveConfig(cliOptions{
		profile:               "nas",
		batchSize:             50_000,
		batchBytesMiB:         90,
		maxPendingBatches:     2,
		indexOnly:             true,
		meiliIndexThreads:     3,
		geoJSONSimplifyMeters: 50,
		geoJSONShards:         4,
		radii:                 "500,5000,25000,100000",
		h3Resolutions:         "5,6,7,8",
		meiliImages:           "getmeili/meilisearch:v1.53.1",
		backends:              "geojson",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !config.IndexOnly || config.PerformanceTarget != "none" || config.BatchBytes != 90<<20 ||
		config.MaxPendingBatches != 2 || config.MeiliIndexThreads != 3 ||
		config.GeoJSONIndex.SimplifyToleranceMeters != 50 || config.GeoJSONIndex.Shards != 4 ||
		config.GeoJSONSweep.IndexSimplificationMeters != 50 ||
		!reflect.DeepEqual(backends, []string{"geojson"}) {
		t.Fatalf("index optimization config = %+v, backends = %v", config, backends)
	}
}

func TestResolveConfigCapacityCreatedSharding(t *testing.T) {
	config, backends, err := resolveConfig(cliOptions{
		profile:                "standard",
		batchSize:              50_000,
		batchBytesMiB:          90,
		indexOnly:              true,
		geoJSONShardMode:       geoJSONShardModeCapacityCreated,
		geoJSONShardMaxTrails:  2_500,
		geoJSONCapacityArrival: geoJSONCapacityArrivalClustered,
		radii:                  "500",
		h3Resolutions:          "7",
		meiliImages:            "getmeili/meilisearch:v1.53.1",
		backends:               "geojson",
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.GeoJSONIndex.ShardMode != geoJSONShardModeCapacityCreated ||
		config.GeoJSONIndex.Shards != 1 || config.GeoJSONIndex.MaxTrailsPerShard != 2_500 ||
		config.GeoJSONIndex.CapacityArrival != geoJSONCapacityArrivalClustered ||
		!reflect.DeepEqual(backends, []string{"geojson"}) {
		t.Fatalf("capacity-created config = %+v, backends = %v", config.GeoJSONIndex, backends)
	}
}

func TestResolveConfigCapacityCreatedFederatedSearchContract(t *testing.T) {
	options := cliOptions{
		profile:                "smoke",
		batchSize:              100,
		batchBytesMiB:          16,
		geoJSONShardMode:       geoJSONShardModeCapacityCreated,
		geoJSONShardMaxTrails:  100,
		geoJSONTune:            true,
		geoJSONDirect:          true,
		geoJSONDirectRes:       "100",
		geoJSONUXRecall:        0.99,
		geoJSONUXPrecision:     0.99,
		geoJSONUXTop10:         0.99,
		geoJSONUXPage:          0.99,
		geoJSONUXPagePrecision: 0.99,
		radii:                  "500,5000,25000,100000",
		h3Resolutions:          "7",
		queryWorkload:          "product",
		meiliImages:            "getmeili/meilisearch:v1.53.1",
		backends:               "geojson",
	}
	config, _, err := resolveConfig(options)
	if err != nil {
		t.Fatal(err)
	}
	if config.IndexOnly || config.QueryWorkload != "product" || config.GeoJSONSweep.Enabled ||
		!reflect.DeepEqual(config.GeoJSONDirect.Resolutions, []int{100}) {
		t.Fatalf("capacity search config = %+v", config)
	}
	if config.GeoJSONDirect.MinimumRecall != 0.99 ||
		config.GeoJSONDirect.MinimumPrecision != 0.99 ||
		config.GeoJSONDirect.MinimumTop10Recall != 0.99 ||
		config.GeoJSONDirect.MinimumComparablePageRate != 0.99 ||
		config.GeoJSONDirect.MinimumComparablePagePrecision != 0.99 ||
		config.GeoJSONDirect.MaximumCountP95RelativeError != 0.01 {
		t.Fatalf("capacity search did not preserve CLI UX thresholds: %+v", config.GeoJSONDirect)
	}
	newerVersion := options
	newerVersion.meiliImages = "getmeili/meilisearch:v1.54.0"
	if _, _, err := resolveConfig(newerVersion); err != nil {
		t.Fatalf("newer official Meilisearch release was rejected: %v", err)
	}

	invalidVersion := options
	invalidVersion.meiliImages = "getmeili/meilisearch:v1.53.0"
	if _, _, err := resolveConfig(invalidVersion); err == nil || !strings.Contains(err.Error(), "v1.53.1") {
		t.Fatalf("invalid version error = %v", err)
	}
	invalidResolutions := options
	invalidResolutions.geoJSONDirectRes = "100,125"
	if _, _, err := resolveConfig(invalidResolutions); err == nil || !strings.Contains(err.Error(), "exactly one direct GeoJSON resolution") {
		t.Fatalf("invalid resolutions error = %v", err)
	}
	r250 := options
	r250.geoJSONDirectRes = "250"
	r250.geoJSONRes = "250"
	config, _, err = resolveConfig(r250)
	if err != nil {
		t.Fatalf("capacity r250 diagnostic: %v", err)
	}
	if !reflect.DeepEqual(config.GeoJSONDirect.Resolutions, []int{250}) {
		t.Fatalf("capacity r250 resolutions = %v", config.GeoJSONDirect.Resolutions)
	}
}

func TestValidateMeiliQualificationImage(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	tests := []struct {
		name      string
		reference string
		wantError bool
	}{
		{name: "minimum release", reference: "getmeili/meilisearch:v1.53.1"},
		{name: "newer release", reference: "getmeili/meilisearch:v1.54.0"},
		{name: "tag and digest", reference: "getmeili/meilisearch:v1.54.0@" + digest},
		{name: "immutable digest", reference: "getmeili/meilisearch@" + digest},
		{name: "older release", reference: "getmeili/meilisearch:v1.53.0", wantError: true},
		{name: "latest alias", reference: "getmeili/meilisearch:latest", wantError: true},
		{name: "unofficial repository", reference: "example.invalid/meilisearch:v1.54.0", wantError: true},
		{name: "missing tag or digest", reference: "getmeili/meilisearch", wantError: true},
		{name: "prerelease", reference: "getmeili/meilisearch:v1.54.0-rc.1", wantError: true},
		{name: "malformed version", reference: "getmeili/meilisearch:v1.54", wantError: true},
		{name: "malformed digest", reference: "getmeili/meilisearch@sha256:abc", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMeiliQualificationImage(test.reference)
			if (err != nil) != test.wantError {
				t.Fatalf("validateMeiliQualificationImage(%q) error = %v, wantError %t", test.reference, err, test.wantError)
			}
		})
	}
}

func TestResolveConfigRejectsInvalidIndexOptimization(t *testing.T) {
	base := cliOptions{
		profile:       "smoke",
		batchSize:     100,
		batchBytesMiB: 16,
		radii:         "500",
		h3Resolutions: "7",
		backends:      "geojson",
	}
	tests := []struct {
		name   string
		mutate func(*cliOptions)
		want   string
	}{
		{name: "payload", mutate: func(options *cliOptions) { options.batchBytesMiB = 100 }, want: "batch-bytes-mib"},
		{name: "pending", mutate: func(options *cliOptions) { options.maxPendingBatches = -1 }, want: "max-pending-batches"},
		{name: "simplification", mutate: func(options *cliOptions) { options.geoJSONSimplifyMeters = -1 }, want: "geojson-simplify-meters"},
		{name: "maximum segment", mutate: func(options *cliOptions) { options.geoJSONMaxSegmentMeters = -1 }, want: "geojson-max-segment-meters"},
		{name: "shards range", mutate: func(options *cliOptions) { options.geoJSONShards = 65 }, want: "geojson-shards"},
		{name: "shards need index only", mutate: func(options *cliOptions) { options.geoJSONShards = 2 }, want: "require --index-only"},
		{name: "unknown shard mode", mutate: func(options *cliOptions) { options.geoJSONShardMode = "calendar" }, want: "geojson-shard-mode"},
		{name: "capacity on hash", mutate: func(options *cliOptions) { options.geoJSONShardMaxTrails = 100 }, want: "only valid"},
		{name: "capacity missing", mutate: func(options *cliOptions) { options.geoJSONShardMode = geoJSONShardModeCapacityCreated }, want: "requires --geojson-shard-max-trails"},
		{name: "capacity search needs product workload", mutate: func(options *cliOptions) {
			options.geoJSONShardMode = geoJSONShardModeCapacityCreated
			options.geoJSONShardMaxTrails = 100
			options.meiliImages = "getmeili/meilisearch:v1.53.1"
		}, want: "--query-workload product"},
		{name: "capacity fixed shards", mutate: func(options *cliOptions) {
			options.geoJSONShardMode = geoJSONShardModeCapacityCreated
			options.geoJSONShardMaxTrails = 100
			options.geoJSONShards = 2
		}, want: "not used"},
		{name: "capacity arrival", mutate: func(options *cliOptions) { options.geoJSONCapacityArrival = "federated" }, want: "geojson-capacity-arrival"},
		{name: "composite target", mutate: func(options *cliOptions) {
			options.indexOnly = true
			options.performanceTarget = "ds923plus"
		}, want: "index-only requires"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := base
			test.mutate(&options)
			if _, _, err := resolveConfig(options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestResolveConfigStandardAndStressDefaults(t *testing.T) {
	for _, test := range []struct {
		profile     string
		trails      int
		queries     int
		repetitions int
	}{
		{profile: "standard", trails: 5_000, queries: 36, repetitions: 2},
		{profile: "stress", trails: 25_000, queries: 60, repetitions: 3},
	} {
		config, _, err := resolveConfig(cliOptions{
			profile:       test.profile,
			batchSize:     100,
			radii:         "500,5000,25000,100000",
			h3Resolutions: "5,6,7,8",
			meiliImages:   "getmeili/meilisearch:v1.36.0",
			backends:      "rtree",
		})
		if err != nil {
			t.Fatalf("resolve %s defaults: %v", test.profile, err)
		}
		if config.TrailCount != test.trails || config.QueryCount != test.queries || config.Repetitions != test.repetitions {
			t.Errorf("%s config = %+v", test.profile, config)
		}
	}
}

func TestResolveConfigNASDefaultsAndGeoJSONSweep(t *testing.T) {
	config, _, err := resolveConfig(cliOptions{
		profile:          "nas",
		batchSize:        100,
		radii:            "500,5000,25000,100000",
		h3Resolutions:    "5,6,7,8",
		geoJSONTune:      true,
		geoJSONRes:       "1000,125,250",
		geoJSONPadding:   "0.5,0",
		geoJSONFinalists: 2,
		meiliImages:      "getmeili/meilisearch:v1.53.1",
		backends:         "geojson",
	})
	if err != nil {
		t.Fatalf("resolve NAS defaults: %v", err)
	}
	if config.TrailCount != 50_000 || config.QueryCount != 60 || config.Repetitions != 84 || config.QueryClients != 4 {
		t.Fatalf("unexpected NAS config: %+v", config)
	}
	if config.PerformanceTarget != "ds923plus" {
		t.Fatalf("performance target = %q", config.PerformanceTarget)
	}
	if !reflect.DeepEqual(config.GeoJSONSweep.Resolutions, []int{125, 250, 1000}) ||
		!reflect.DeepEqual(config.GeoJSONSweep.ExtraPaddingMeters, []float64{0, 0.5}) ||
		config.GeoJSONSweep.Finalists != 2 {
		t.Fatalf("GeoJSON sweep = %+v", config.GeoJSONSweep)
	}
}

func TestResolveConfigRejectsInvalidGeoJSONSweep(t *testing.T) {
	base := cliOptions{
		profile:          "smoke",
		batchSize:        100,
		radii:            "500",
		h3Resolutions:    "7",
		geoJSONTune:      true,
		geoJSONRes:       "125",
		geoJSONPadding:   "0,0.5",
		geoJSONFinalists: 1,
		meiliImages:      "getmeili/meilisearch:v1.53.1",
		backends:         "geojson",
	}
	for _, resolution := range []string{"2", "1001"} {
		options := base
		options.geoJSONRes = resolution
		if _, _, err := resolveConfig(options); err == nil || !strings.Contains(err.Error(), "outside 3..1000") {
			t.Fatalf("resolution %s error = %v", resolution, err)
		}
	}
	options := base
	options.geoJSONPadding = "-0.1"
	if _, _, err := resolveConfig(options); err == nil || !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("negative padding error = %v", err)
	}
	options = base
	options.geoJSONRes = "3,4,5,6,7,8,9,10,11,12,13"
	if _, _, err := resolveConfig(options); err == nil || !strings.Contains(err.Error(), "maximum is 32") {
		t.Fatalf("variant-cap error = %v", err)
	}
}

func TestResolveConfigValidatesGeoJSONDirectContract(t *testing.T) {
	base := cliOptions{
		profile:                "smoke",
		batchSize:              100,
		radii:                  "500,100000",
		h3Resolutions:          "7",
		geoJSONTune:            true,
		geoJSONRes:             "64,100,125,250",
		geoJSONPadding:         "0,0.5",
		geoJSONFinalists:       1,
		geoJSONDirect:          true,
		geoJSONDirectRes:       "100,125,250",
		geoJSONUXBoundary:      50,
		geoJSONUXRecall:        0.99,
		geoJSONUXPrecision:     0.99,
		geoJSONUXTop10:         0.99,
		geoJSONUXPage:          0.99,
		geoJSONUXPagePrecision: 0.99,
		meiliImages:            "getmeili/meilisearch:v1.53.1",
		backends:               "geojson",
	}
	config, _, err := resolveConfig(base)
	if err != nil {
		t.Fatal(err)
	}
	if !config.GeoJSONDirect.Enabled || !reflect.DeepEqual(config.GeoJSONDirect.Resolutions, []int{100, 125, 250}) || config.GeoJSONDirect.BoundaryToleranceMeters != 50 {
		t.Fatalf("direct config = %+v", config.GeoJSONDirect)
	}
	if config.GeoJSONDirect.MaximumCountP95RelativeError != 0.01 {
		t.Fatalf("direct count contract = %+v", config.GeoJSONDirect)
	}

	withoutSweep := base
	withoutSweep.geoJSONTune = false
	if _, _, err := resolveConfig(withoutSweep); err == nil || !strings.Contains(err.Error(), "requires --geojson-tune=true") {
		t.Fatalf("direct without sweep error = %v", err)
	}
	notScreened := base
	notScreened.geoJSONDirectRes = "112"
	if _, _, err := resolveConfig(notScreened); err == nil || !strings.Contains(err.Error(), "not present") {
		t.Fatalf("unscreened direct resolution error = %v", err)
	}
}

func TestGeoJSONDirectQualifiedRequiresRequestedEvidence(t *testing.T) {
	passed := GeoJSONDirectAccuracy{Status: "passed"}
	report := &GeoJSONDirectReport{
		SelectionStatus: "spatial_qualified",
		Plan:            &GeoJSONQueryPlan{ID: "r125-exact"},
		Accuracy:        &passed,
	}
	if !geoJSONDirectQualified(report, false) {
		t.Fatal("spatial evidence did not qualify an engine-only run")
	}
	if geoJSONDirectQualified(report, true) {
		t.Fatal("spatial evidence alone qualified a product run")
	}
	report.SelectionStatus = "qualified_with_unsupported_proximity_sort"
	report.ProductWorkload = &GeoJSONDirectProductWorkloadReport{Status: "passed"}
	if !geoJSONDirectQualified(report, true) {
		t.Fatal("passed product evidence with explicit proximity limitation did not qualify")
	}
	if geoJSONDirectTargetQualified(report, true, true) {
		t.Fatal("direct path qualified for a requested target without performance evidence")
	}
	report.Performance = &PerformanceReport{Status: "passed"}
	if !geoJSONDirectTargetQualified(report, true, true) {
		t.Fatal("direct path with product and performance evidence did not qualify")
	}

	backend := BackendReport{
		Backend:       "geojson",
		Performance:   &PerformanceReport{Status: "failed"},
		GeoJSONDirect: report,
	}
	if !backendPerformanceQualified(backend) {
		t.Fatal("passing direct architecture did not satisfy the backend performance choice")
	}
	backend.Backend = "h3"
	if backendPerformanceQualified(backend) {
		t.Fatal("H3 incorrectly reused GeoJSON direct performance evidence")
	}
}

func TestRunBackendWithTimeoutClassifiesDeadline(t *testing.T) {
	report := runBackendWithTimeout(context.Background(), time.Millisecond, func(ctx context.Context) BackendReport {
		<-ctx.Done()
		return BackendReport{Backend: "slow", Status: "failed", Error: ctx.Err().Error()}
	})
	if report.Status != "timed_out" {
		t.Fatalf("status = %q, want timed_out", report.Status)
	}
	if report.ElapsedMS <= 0 {
		t.Fatalf("elapsed_ms = %v, want positive", report.ElapsedMS)
	}
	if !strings.Contains(report.Error, "time budget 1ms reached") {
		t.Fatalf("error = %q", report.Error)
	}

	parent, cancel := context.WithCancel(context.Background())
	cancel()
	report = runBackendWithTimeout(parent, time.Minute, func(ctx context.Context) BackendReport {
		return BackendReport{Backend: "canceled", Status: "failed", Error: ctx.Err().Error()}
	})
	if report.Status != "failed" {
		t.Fatalf("parent-canceled status = %q, want failed", report.Status)
	}
}

func TestParseFloatListRejectsNonFiniteValues(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf", "-Inf"} {
		if _, err := parseFloatList(value); err == nil {
			t.Fatalf("parseFloatList(%q) succeeded", value)
		}
	}
	values, err := parseFloatList("5000,500,500.0")
	if err != nil {
		t.Fatalf("parse finite list: %v", err)
	}
	if len(values) != 2 || values[0] != 500 || values[1] != 5_000 {
		t.Fatalf("parsed radii = %v, want [500 5000]", values)
	}
}

func TestGeoJSONContractErrorIsAGate(t *testing.T) {
	if err := geoJSONContractError([]ContractReport{{MeilisearchVersion: "1.44.0", Status: "passed"}}); err != nil {
		t.Fatalf("passed contract returned error: %v", err)
	}
	if err := geoJSONContractError([]ContractReport{{
		MeilisearchVersion: "1.53.1",
		Status:             "passed",
		Diagnostics: []GeoJSONContractDiagnosticReport{{
			Name:   "geojson_cell_coverage",
			Status: "known_engine_limitation",
		}},
	}}); err != nil {
		t.Fatalf("additive known engine limitation changed the top-level gate: %v", err)
	}
	if err := geoJSONContractError([]ContractReport{{
		MeilisearchVersion: "1.53.1",
		Status:             "passed",
		Diagnostics: []GeoJSONContractDiagnosticReport{{
			Name:   "geojson_cell_coverage",
			Status: "diagnostic_error",
			Error:  "probe index unavailable",
		}},
	}}); err == nil || !strings.Contains(err.Error(), "geojson_cell_coverage") {
		t.Fatalf("diagnostic error did not fail the contract gate: %v", err)
	}
	for _, status := range []string{"failed", "unsupported"} {
		err := geoJSONContractError([]ContractReport{{MeilisearchVersion: "1.11.3", Status: status, Error: "not supported"}})
		if err == nil {
			t.Fatalf("%s contract returned no error", status)
		}
	}
}

func TestClassifyGeoJSONContractResultPrioritizesDiagnosticTimeout(t *testing.T) {
	contract := ContractReport{
		MeilisearchVersion: "1.53.1",
		Status:             "passed",
		Diagnostics: []GeoJSONContractDiagnosticReport{{
			Name:   "geojson_hierarchical_split",
			Status: "diagnostic_error",
			Error:  "search diagnostic: context deadline exceeded",
		}},
	}

	got := classifyGeoJSONContractResult(contract, true, 15*time.Minute)
	if got.Status != "timed_out" {
		t.Fatalf("contract status = %q, want timed_out", got.Status)
	}
	for _, wanted := range []string{"time budget 15m0s reached", "geojson_hierarchical_split", "context deadline exceeded"} {
		if !strings.Contains(got.Error, wanted) {
			t.Fatalf("contract error %q does not contain %q", got.Error, wanted)
		}
	}

	nonTimedOut := classifyGeoJSONContractResult(contract, false, 15*time.Minute)
	if nonTimedOut.Status != "failed" || !strings.Contains(nonTimedOut.Error, "geojson_hierarchical_split") {
		t.Fatalf("non-timeout diagnostic result = %+v, want failed with diagnostic detail", nonTimedOut)
	}

	passed := classifyGeoJSONContractResult(ContractReport{
		MeilisearchVersion: "1.53.1",
		Status:             "passed",
		Diagnostics: []GeoJSONContractDiagnosticReport{{
			Name: "geojson_hierarchical_split", Status: "passed",
		}},
	}, true, 15*time.Minute)
	if passed.Status != "passed" || passed.Error != "" {
		t.Fatalf("completed contract at deadline = %+v, want passed", passed)
	}
}
