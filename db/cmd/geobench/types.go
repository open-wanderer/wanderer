package main

import "time"

const earthRadiusMeters = 6371008.8

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Trail struct {
	ID       string         `json:"id"`
	Scenario string         `json:"scenario"`
	Parts    [][]Coordinate `json:"parts"`
}

func (t Trail) Start() Coordinate {
	for _, part := range t.Parts {
		if len(part) > 0 {
			return part[0]
		}
	}
	return Coordinate{}
}

type QueryPoint struct {
	ID       string     `json:"id"`
	Scenario string     `json:"scenario"`
	Point    Coordinate `json:"point"`
}

type Dataset struct {
	Seed        int64        `json:"seed"`
	Trails      []Trail      `json:"trails"`
	QueryPoints []QueryPoint `json:"query_points"`
}

type RunConfig struct {
	Profile           string              `json:"profile"`
	Seed              int64               `json:"seed"`
	TrailCount        int                 `json:"trail_count"`
	QueryCount        int                 `json:"query_count"`
	RadiiMeters       []float64           `json:"radii_meters"`
	Repetitions       int                 `json:"repetitions"`
	BatchSize         int                 `json:"batch_size"`
	BatchBytes        int                 `json:"batch_bytes"`
	MaxPendingBatches int                 `json:"max_pending_batches"`
	IndexOnly         bool                `json:"index_only"`
	MeiliIndexThreads int                 `json:"meili_indexing_threads"`
	IncrementalCount  int                 `json:"incremental_count"`
	H3Resolutions     []int               `json:"h3_resolutions"`
	GeoJSONIndex      GeoJSONIndexConfig  `json:"geojson_index"`
	GeoJSONSweep      GeoJSONSweepConfig  `json:"geojson_sweep"`
	GeoJSONDirect     GeoJSONDirectConfig `json:"geojson_direct"`
	MeilisearchImages []string            `json:"meilisearch_images"`
	QueryWorkload     string              `json:"query_workload"`
	QueryClients      int                 `json:"query_clients"`
	PerformanceTarget string              `json:"performance_target"`
	FailOnIncorrect   bool                `json:"fail_on_incorrect"`
	FailOnUX          bool                `json:"fail_on_ux"`
	BackendTimeout    string              `json:"backend_timeout"`
	OverallTimeout    string              `json:"overall_timeout"`
}

type GeoJSONIndexConfig struct {
	SimplifyToleranceMeters float64 `json:"simplify_tolerance_meters"`
	MaxSegmentLengthMeters  float64 `json:"max_segment_length_meters"`
	ShardMode               string  `json:"shard_mode"`
	Shards                  int     `json:"shards"`
	MaxTrailsPerShard       int     `json:"max_trails_per_shard,omitempty"`
	CapacityArrival         string  `json:"capacity_arrival,omitempty"`
}

type GeoJSONIndexReport struct {
	SimplifyToleranceMeters   float64                   `json:"simplify_tolerance_meters"`
	MaxSegmentLengthMeters    float64                   `json:"max_segment_length_meters"`
	ShardMode                 string                    `json:"shard_mode"`
	MaxTrailsPerShard         int                       `json:"max_trails_per_shard,omitempty"`
	CapacityArrival           string                    `json:"capacity_arrival,omitempty"`
	ShardAssignmentSHA256     string                    `json:"shard_assignment_sha256,omitempty"`
	AssignmentScope           string                    `json:"assignment_scope,omitempty"`
	Shards                    int                       `json:"shards"`
	ShardAlgorithm            string                    `json:"shard_algorithm,omitempty"`
	ShardDocumentCounts       []int                     `json:"shard_document_counts,omitempty"`
	ShardSourceVertices       []int                     `json:"shard_source_vertices,omitempty"`
	ShardIndexedVertices      []int                     `json:"shard_indexed_vertices,omitempty"`
	ShardSourceSegments       []int                     `json:"shard_source_segments,omitempty"`
	ShardIndexedSegments      []int                     `json:"shard_indexed_segments,omitempty"`
	ShardBuilds               []GeoJSONShardBuildReport `json:"shard_builds,omitempty"`
	SourceVertices            int                       `json:"source_vertices"`
	PreNormalizationVertices  int                       `json:"pre_normalization_vertices"`
	PostNormalizationVertices int                       `json:"post_normalization_vertices"`
	DensifiedVertices         int                       `json:"densified_vertices"`
	AntimeridianCuts          int                       `json:"antimeridian_cuts"`
	IndexedVertices           int                       `json:"indexed_vertices"`
	SourceSegments            int                       `json:"source_segments"`
	PreNormalizationSegments  int                       `json:"pre_normalization_segments"`
	PostNormalizationSegments int                       `json:"post_normalization_segments"`
	IndexedSegments           int                       `json:"indexed_segments"`
	VertexReductionPercent    float64                   `json:"vertex_reduction_percent"`
}

// GeoJSONShardBuildReport makes the sharding experiment auditable without
// pretending that the last task or document count from one index describes
// the complete build.
type GeoJSONShardBuildReport struct {
	Shard    int         `json:"shard"`
	IndexUID string      `json:"index_uid"`
	Phase    PhaseReport `json:"phase"`
}

// GeoJSONDirectConfig is an explicitly approximate, Meilisearch-only product
// contract. It is separate from the exact candidate-plus-refinement sweep.
type GeoJSONDirectConfig struct {
	Enabled                        bool    `json:"enabled"`
	Resolutions                    []int   `json:"resolutions"`
	BoundaryToleranceMeters        float64 `json:"boundary_tolerance_meters"`
	MinimumRecall                  float64 `json:"minimum_recall"`
	MinimumPrecision               float64 `json:"minimum_precision"`
	MinimumTop10Recall             float64 `json:"minimum_top_10_recall"`
	MinimumComparablePageRate      float64 `json:"minimum_comparable_page_recall"`
	MinimumComparablePagePrecision float64 `json:"minimum_comparable_page_precision"`
	MaximumCountP95RelativeError   float64 `json:"maximum_count_p95_relative_error"`
}

type GeoJSONSweepConfig struct {
	Enabled                   bool      `json:"enabled"`
	Resolutions               []int     `json:"resolutions"`
	ExtraPaddingMeters        []float64 `json:"extra_padding_meters"`
	IndexSimplificationMeters float64   `json:"index_simplification_meters"`
	Finalists                 int       `json:"finalists"`
}

type BenchmarkReport struct {
	StartedAt        time.Time             `json:"started_at"`
	Config           RunConfig             `json:"config"`
	Environment      map[string]string     `json:"environment"`
	Reproducibility  ReproducibilityReport `json:"reproducibility"`
	Dataset          DatasetSummary        `json:"dataset"`
	OracleDurationMS float64               `json:"oracle_duration_ms,omitempty"`
	Contracts        []ContractReport      `json:"contracts"`
	Backends         []BackendReport       `json:"backends"`
}

type ReproducibilityReport struct {
	Invocation       []string            `json:"invocation,omitempty"`
	ExecutableSHA256 string              `json:"executable_sha256,omitempty"`
	HarnessSHA256    string              `json:"harness_sha256,omitempty"`
	DatasetSHA256    string              `json:"dataset_sha256,omitempty"`
	DockerImages     []DockerImageReport `json:"docker_images,omitempty"`
	Warnings         []string            `json:"warnings,omitempty"`
}

type DockerImageReport struct {
	Reference   string   `json:"reference"`
	ImageID     string   `json:"image_id,omitempty"`
	RepoDigests []string `json:"repo_digests,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type DatasetSummary struct {
	Trails          int            `json:"trails"`
	QueryPoints     int            `json:"query_points"`
	Vertices        int            `json:"vertices"`
	Segments        int            `json:"segments"`
	MultiLineTrails int            `json:"multiline_trails"`
	Scenarios       map[string]int `json:"scenarios"`
}

type ContractReport struct {
	MeilisearchVersion string                            `json:"meilisearch_version"`
	Status             string                            `json:"status"`
	DurationMS         float64                           `json:"duration_ms"`
	Error              string                            `json:"error,omitempty"`
	Diagnostics        []GeoJSONContractDiagnosticReport `json:"diagnostics,omitempty"`
}

// GeoJSONContractDiagnosticReport records an engine limitation without
// changing the basic LineString/MultiLineString contract status.
type GeoJSONContractDiagnosticReport struct {
	Name       string                                 `json:"name"`
	Status     string                                 `json:"status"`
	Cause      string                                 `json:"cause,omitempty"`
	QueryPoint Coordinate                             `json:"query_point"`
	Cases      []GeoJSONContractDiagnosticCaseReport  `json:"cases,omitempty"`
	Counts     []GeoJSONContractDiagnosticCountReport `json:"counts,omitempty"`
	Error      string                                 `json:"error,omitempty"`
}

type GeoJSONContractDiagnosticCountReport struct {
	Name          string `json:"name"`
	Filter        string `json:"filter"`
	ExpectedCount int64  `json:"expected_count"`
	ActualCount   int64  `json:"actual_count"`
	MissingCount  int64  `json:"missing_count"`
}

type GeoJSONContractDiagnosticCaseReport struct {
	Name          string   `json:"name"`
	Filter        string   `json:"filter"`
	RadiusMeters  float64  `json:"radius_meters,omitempty"`
	ExpectedIDs   []string `json:"expected_ids"`
	ActualIDs     []string `json:"actual_ids"`
	MissingIDs    []string `json:"missing_ids,omitempty"`
	UnexpectedIDs []string `json:"unexpected_ids,omitempty"`
}

type BackendReport struct {
	Backend                    string                 `json:"backend"`
	MeilisearchVersion         string                 `json:"meilisearch_version,omitempty"`
	Status                     string                 `json:"status"`
	Error                      string                 `json:"error,omitempty"`
	ElapsedMS                  float64                `json:"elapsed_ms"`
	IndexSetupMS               float64                `json:"index_setup_ms,omitempty"`
	ApplicationDatabaseSetupMS float64                `json:"application_database_setup_ms,omitempty"`
	ApplicationDatabaseBytes   int64                  `json:"application_database_bytes,omitempty"`
	FullIndex                  PhaseReport            `json:"full_index"`
	IncrementalIndex           PhaseReport            `json:"incremental_index"`
	AddIndex                   PhaseReport            `json:"add_index"`
	DeleteIndex                PhaseReport            `json:"delete_index"`
	GeoJSONIndex               *GeoJSONIndexReport    `json:"geojson_index,omitempty"`
	Queries                    []QueryReport          `json:"queries,omitempty"`
	ProductWorkload            *ProductWorkloadReport `json:"product_workload,omitempty"`
	GeoJSONPlan                *GeoJSONQueryPlan      `json:"geojson_query_plan,omitempty"`
	GeoJSONSweep               *GeoJSONSweepReport    `json:"geojson_sweep,omitempty"`
	GeoJSONDirect              *GeoJSONDirectReport   `json:"geojson_direct,omitempty"`
	Performance                *PerformanceReport     `json:"performance_target,omitempty"`
	ExecutionScope             string                 `json:"execution_scope,omitempty"`
}

type GeoJSONQueryPlan struct {
	ID                        string  `json:"id"`
	RadiusMode                string  `json:"radius_mode"`
	Resolution                int     `json:"resolution,omitempty"`
	ExtraPaddingMeters        float64 `json:"extra_padding_meters"`
	IndexSimplificationMeters float64 `json:"index_simplification_meters"`
	Conservative              bool    `json:"conservative"`
	ExactRefinement           bool    `json:"exact_refinement"`
	RadiusFormula             string  `json:"radius_formula"`
}

type GeoJSONSweepReport struct {
	SelectionRule    string                 `json:"selection_rule"`
	SelectedID       string                 `json:"selected_id,omitempty"`
	SelectionStatus  string                 `json:"selection_status,omitempty"`
	SelectionSamples int                    `json:"selection_samples,omitempty"`
	Variants         []GeoJSONVariantReport `json:"variants"`
}

type GeoJSONVariantReport struct {
	Plan                     GeoJSONQueryPlan       `json:"plan"`
	Status                   string                 `json:"status"`
	Finalist                 bool                   `json:"finalist"`
	Selectable               bool                   `json:"selectable"`
	ScreenSamples            int                    `json:"screen_samples"`
	TruncatedSamples         int                    `json:"truncated_samples"`
	CandidateLatency         LatencyStats           `json:"candidate_latency_ms"`
	AverageCandidates        float64                `json:"average_candidates"`
	MaximumCandidates        int                    `json:"maximum_candidates"`
	ExpectedInterior         int                    `json:"expected_interior"`
	ExpectedBoundary         int                    `json:"expected_boundary_zone"`
	CandidateTruePositives   int                    `json:"candidate_true_positives"`
	CandidateFalsePositives  int                    `json:"candidate_false_positives"`
	CandidateFalseNegatives  int                    `json:"candidate_false_negatives"`
	BoundaryCandidateMisses  int                    `json:"boundary_candidate_misses"`
	CandidateRecall          float64                `json:"candidate_recall"`
	MaxInsideMissMeters      float64                `json:"max_inside_miss_meters"`
	MaxRadiusExpansionMeters float64                `json:"max_radius_expansion_meters"`
	Queries                  []QueryReport          `json:"queries,omitempty"`
	ProductWorkload          *ProductWorkloadReport `json:"product_workload,omitempty"`
	DirectUX                 *GeoJSONDirectAccuracy `json:"direct_ux,omitempty"`
	Error                    string                 `json:"error,omitempty"`
}

// GeoJSONDirectReport is additive evidence for a fuzzy product contract. A
// failure here does not weaken or relabel the exact GeoJSON backend result.
type GeoJSONDirectReport struct {
	SelectionRule   string                              `json:"selection_rule"`
	SelectedID      string                              `json:"selected_id,omitempty"`
	SelectionStatus string                              `json:"selection_status"`
	Plan            *GeoJSONQueryPlan                   `json:"plan,omitempty"`
	Accuracy        *GeoJSONDirectAccuracy              `json:"accuracy,omitempty"`
	ProductAudits   []GeoJSONDirectProductAuditReport   `json:"product_audits,omitempty"`
	ProductOracleMS float64                             `json:"product_oracle_ms,omitempty"`
	PlanAuditWallMS float64                             `json:"plan_audit_wall_ms,omitempty"`
	SpatialAudit    *GeoJSONDirectSpatialAuditReport    `json:"spatial_audit,omitempty"`
	ProductWorkload *GeoJSONDirectProductWorkloadReport `json:"product_workload,omitempty"`
	Performance     *PerformanceReport                  `json:"performance,omitempty"`
	Error           string                              `json:"error,omitempty"`
}

// GeoJSONDirectSpatialAuditReport covers every query-point/radius pair without
// product ACLs. Product cases rotate through radii, so they cannot replace this
// complete spatial contract.
type GeoJSONDirectSpatialAuditReport struct {
	Status           string                `json:"status"`
	Cases            int                   `json:"cases"`
	ShardFanout      int                   `json:"shard_fanout"`
	HTTPRequests     int                   `json:"http_requests"`
	ShardQueries     int                   `json:"shard_queries"`
	RequestBytes     int64                 `json:"meilisearch_request_bytes"`
	ResponseBytes    int64                 `json:"meilisearch_response_bytes"`
	CandidateLatency LatencyStats          `json:"candidate_latency_ms"`
	EngineProcessing LatencyStats          `json:"engine_processing_ms"`
	Accuracy         GeoJSONDirectAccuracy `json:"accuracy"`
	Error            string                `json:"error,omitempty"`
}

// GeoJSONDirectProductAuditReport records the untimed correctness audit and
// one product-page calibration request for every spatially qualified plan.
// The selected plan is therefore based on the query shape users will execute,
// rather than on the ID-only spatial screen alone.
type GeoJSONDirectProductAuditReport struct {
	Plan               *GeoJSONQueryPlan                   `json:"plan,omitempty"`
	PlanID             string                              `json:"plan_id"`
	Resolution         int                                 `json:"resolution"`
	Status             string                              `json:"status"`
	Cases              int                                 `json:"cases"`
	ProductPageLatency LatencyStats                        `json:"product_page_latency_ms"`
	Accuracy           GeoJSONDirectAccuracy               `json:"accuracy"`
	CountQualification GeoJSONDirectProductCountReport     `json:"count_qualification"`
	WarmWorkload       *GeoJSONDirectProductWorkloadReport `json:"warm_workload,omitempty"`
	Performance        *PerformanceReport                  `json:"performance,omitempty"`
	WarmError          string                              `json:"warm_error,omitempty"`
	Error              string                              `json:"error,omitempty"`
}

type GeoJSONDirectAccuracy struct {
	Status                         string                       `json:"status"`
	BoundaryToleranceMeters        float64                      `json:"boundary_tolerance_meters"`
	MinimumRecall                  float64                      `json:"minimum_recall"`
	MinimumPrecision               float64                      `json:"minimum_precision"`
	MinimumTop10Recall             float64                      `json:"minimum_top_10_recall"`
	MinimumComparablePageRecall    float64                      `json:"minimum_comparable_page_recall"`
	MinimumComparablePagePrecision float64                      `json:"minimum_comparable_page_precision"`
	Requests                       int                          `json:"requests"`
	RequestsWithExpected           int                          `json:"requests_with_expected"`
	RequestsWithClearExpected      int                          `json:"requests_with_clear_expected"`
	ReturnedHits                   int                          `json:"returned_hits"`
	ExpectedHits                   int                          `json:"expected_hits"`
	TruePositives                  int                          `json:"true_positives"`
	FalsePositives                 int                          `json:"false_positives"`
	FalseNegatives                 int                          `json:"false_negatives"`
	ClearExpectedHits              int                          `json:"clear_expected_hits"`
	ClearTruePositives             int                          `json:"clear_true_positives"`
	MaterialFalseNegatives         int                          `json:"material_false_negatives"`
	BoundaryMisses                 int                          `json:"boundary_misses"`
	AllowedBoundaryFalsePositives  int                          `json:"allowed_boundary_false_positives"`
	MaterialFalsePositives         int                          `json:"material_false_positives"`
	SemanticFalsePositives         int                          `json:"semantic_false_positives"`
	FalseEmptyRequests             int                          `json:"false_empty_requests"`
	NearestExpected                int                          `json:"nearest_expected"`
	NearestReturned                int                          `json:"nearest_returned"`
	Top10Expected                  int                          `json:"top_10_expected"`
	Top10Returned                  int                          `json:"top_10_returned"`
	ComparablePageExpected         int                          `json:"comparable_page_expected"`
	ComparablePageReturned         int                          `json:"comparable_page_returned"`
	ComparablePageActual           int                          `json:"comparable_page_actual"`
	ComparablePageMaterialFP       int                          `json:"comparable_page_material_false_positives"`
	ComparablePageCases            int                          `json:"comparable_page_cases"`
	UnsupportedSortCases           int                          `json:"unsupported_sort_cases"`
	TruncatedRequests              int                          `json:"truncated_requests"`
	Recall                         float64                      `json:"recall"`
	Precision                      float64                      `json:"precision"`
	ClearRecall                    float64                      `json:"clear_recall"`
	NearestRecall                  float64                      `json:"nearest_recall"`
	Top10Recall                    float64                      `json:"top_10_recall"`
	ComparablePageRecall           float64                      `json:"comparable_page_recall"`
	ComparablePagePrecision        float64                      `json:"comparable_page_precision"`
	NonEmptyRequestSuccessRate     float64                      `json:"non_empty_request_success_rate"`
	WorstComparablePageRecall      float64                      `json:"worst_comparable_page_recall"`
	WorstComparablePagePrecision   float64                      `json:"worst_comparable_page_precision"`
	WorstRequestRecall             float64                      `json:"worst_request_recall"`
	MaxInsideMissMeters            float64                      `json:"max_inside_miss_meters"`
	MaxOutsideHitMeters            float64                      `json:"max_outside_hit_meters"`
	CountError                     GeoJSONDirectCountError      `json:"count_error"`
	Violations                     []GeoJSONDirectCaseViolation `json:"violations,omitempty"`
}

type GeoJSONDirectCountError struct {
	MeanAbsolute float64 `json:"mean_absolute"`
	P95Absolute  float64 `json:"p95_absolute"`
	MaxAbsolute  int     `json:"max_absolute"`
	MeanRelative float64 `json:"mean_relative"`
	P95Relative  float64 `json:"p95_relative"`
	MaxRelative  float64 `json:"max_relative"`
}

type GeoJSONDirectCaseViolation struct {
	QueryID                     string                              `json:"query_id"`
	Scenario                    string                              `json:"scenario"`
	RadiusMeters                float64                             `json:"radius_meters"`
	Expected                    int                                 `json:"expected"`
	Returned                    int                                 `json:"returned"`
	FalseNegatives              int                                 `json:"false_negatives"`
	MaterialFalseNegative       int                                 `json:"material_false_negatives"`
	FalsePositives              int                                 `json:"false_positives"`
	MaterialFalsePositive       int                                 `json:"material_false_positives"`
	Recall                      float64                             `json:"recall"`
	MaxInsideMissMeters         float64                             `json:"max_inside_miss_meters"`
	MaxOutsideHitMeters         float64                             `json:"max_outside_hit_meters"`
	MaterialFalseNegativeTrails []GeoJSONDirectTrailViolationDetail `json:"material_false_negative_trails,omitempty"`
	MaterialFalsePositiveTrails []GeoJSONDirectTrailViolationDetail `json:"material_false_positive_trails,omitempty"`
	FalseEmpty                  bool                                `json:"false_empty"`
	Error                       string                              `json:"error,omitempty"`
}

// GeoJSONDirectTrailViolationDetail retains a bounded, deterministic sample
// of the worst material spatial errors so a benchmark failure can be traced
// back to concrete source geometry without expanding every result ID.
type GeoJSONDirectTrailViolationDetail struct {
	TrailID           string  `json:"trail_id"`
	DistanceMeters    float64 `json:"distance_meters"`
	RadiusDeltaMeters float64 `json:"radius_delta_meters"`
}

type GeoJSONDirectProductWorkloadReport struct {
	Status                   string                           `json:"status"`
	Clients                  int                              `json:"clients"`
	Cases                    int                              `json:"cases"`
	ExpectedSamples          int                              `json:"expected_samples"`
	Samples                  int                              `json:"samples"`
	UnstableWarmSamples      int                              `json:"unstable_warm_samples"`
	AverageResponseBytes     float64                          `json:"average_response_bytes"`
	AverageRequestBytes      float64                          `json:"average_meilisearch_request_bytes,omitempty"`
	AverageWireResponseBytes float64                          `json:"average_meilisearch_response_bytes,omitempty"`
	AverageHTTPRequests      float64                          `json:"average_http_requests,omitempty"`
	AverageShardQueries      float64                          `json:"average_shard_queries,omitempty"`
	ShardFanout              int                              `json:"shard_fanout,omitempty"`
	EngineBaselineBytes      int64                            `json:"engine_baseline_bytes,omitempty"`
	EnginePeakBytes          int64                            `json:"engine_peak_bytes,omitempty"`
	EnginePeakDelta          int64                            `json:"engine_peak_delta_bytes,omitempty"`
	ClientBaselineRSSBytes   int64                            `json:"client_baseline_rss_bytes,omitempty"`
	ClientPeakRSSBytes       int64                            `json:"client_peak_rss_bytes,omitempty"`
	ClientPeakRSSDelta       int64                            `json:"client_peak_rss_delta_bytes,omitempty"`
	FirstSampleLatency       LatencyStats                     `json:"first_sample_latency_ms"`
	TotalLatency             LatencyStats                     `json:"total_latency_ms"`
	EngineProcessing         LatencyStats                     `json:"engine_processing_ms"`
	ConcurrentWallMS         float64                          `json:"concurrent_wall_ms"`
	ThroughputQPS            float64                          `json:"throughput_queries_per_second"`
	Accuracy                 GeoJSONDirectAccuracy            `json:"accuracy"`
	CountDiagnostics         GeoJSONDirectProductCountReport  `json:"count_diagnostics"`
	RadiusReports            []GeoJSONDirectRadiusReport      `json:"radius_reports"`
	CaseReports              []GeoJSONDirectProductCaseReport `json:"case_reports"`
}

// GeoJSONDirectProductCountReport separates the normative relative p95 gate
// from additional absolute, exact-count and totalPages diagnostics.
type GeoJSONDirectProductCountReport struct {
	Status                  string                  `json:"status"`
	MaximumP95RelativeError float64                 `json:"maximum_p95_relative_error"`
	Cases                   int                     `json:"cases"`
	ExactCountCases         int                     `json:"exact_count_cases"`
	ExactCountRate          float64                 `json:"exact_count_rate"`
	WithinOnePercentCases   int                     `json:"within_one_percent_cases"`
	WithinOnePercentRate    float64                 `json:"within_one_percent_rate"`
	TotalPagesMatchCases    int                     `json:"total_pages_match_cases"`
	TotalPagesMatchRate     float64                 `json:"total_pages_match_rate"`
	CountError              GeoJSONDirectCountError `json:"count_error"`
}

type GeoJSONDirectRadiusReport struct {
	RadiusMeters float64      `json:"radius_meters"`
	Samples      int          `json:"samples"`
	TotalLatency LatencyStats `json:"total_latency_ms"`
}

type GeoJSONDirectProductCaseReport struct {
	Name                   string   `json:"name"`
	QueryID                string   `json:"query_id"`
	Scenario               string   `json:"scenario"`
	RadiusMeters           float64  `json:"radius_meters"`
	Sort                   string   `json:"sort"`
	SortSupported          bool     `json:"sort_supported"`
	PageComparable         bool     `json:"page_comparable"`
	ExpectedTotalItems     int      `json:"expected_total_items"`
	ActualTotalItems       int      `json:"actual_total_items"`
	SignedCountDelta       int      `json:"signed_count_delta"`
	FalseNegatives         int      `json:"false_negatives"`
	BoundaryFalseNegatives int      `json:"boundary_false_negatives"`
	MaterialFalseNegatives int      `json:"material_false_negatives"`
	FalsePositives         int      `json:"false_positives"`
	BoundaryFalsePositives int      `json:"boundary_false_positives"`
	MaterialFalsePositives int      `json:"material_false_positives"`
	SemanticFalsePositives int      `json:"semantic_false_positives"`
	ExpectedTotalPages     int      `json:"expected_total_pages"`
	ActualTotalPages       int      `json:"actual_total_pages"`
	ExpectedHitIDs         []string `json:"expected_hit_ids"`
	ActualHitIDs           []string `json:"actual_hit_ids"`
	PageRecall             float64  `json:"page_recall"`
	PagePrecision          float64  `json:"page_precision"`
	FalseEmpty             bool     `json:"false_empty"`
	CorrectSemantics       bool     `json:"correct_non_spatial_semantics"`
	ShardFanout            int      `json:"shard_fanout,omitempty"`
	HTTPRequests           int      `json:"http_requests,omitempty"`
	ShardQueries           int      `json:"shard_queries,omitempty"`
	WireResponseBytes      int64    `json:"meilisearch_response_bytes,omitempty"`
	RequestBytes           int64    `json:"meilisearch_request_bytes,omitempty"`
	Error                  string   `json:"error,omitempty"`
}

type PerformanceReport struct {
	Target             string       `json:"target"`
	Status             string       `json:"status"`
	Reason             string       `json:"reason,omitempty"`
	MinimumSamples     int          `json:"minimum_samples"`
	MeasuredSamples    int          `json:"measured_samples"`
	RequiredClients    int          `json:"required_clients"`
	MeasuredClients    int          `json:"measured_clients"`
	NormalThresholds   LatencyStats `json:"normal_thresholds_ms"`
	BroadThresholds    LatencyStats `json:"broad_thresholds_ms"`
	NormalWorst        LatencyStats `json:"normal_worst_ms"`
	BroadWorst         LatencyStats `json:"broad_worst_ms"`
	NormalLatencyPass  bool         `json:"normal_latency_pass"`
	BroadLatencyPass   bool         `json:"broad_latency_pass"`
	CorrectnessPass    bool         `json:"correctness_pass"`
	ConcurrencyPass    bool         `json:"concurrency_pass"`
	MinimumThroughput  float64      `json:"minimum_throughput_queries_per_second"`
	MeasuredThroughput float64      `json:"measured_throughput_queries_per_second"`
	ThroughputPass     bool         `json:"throughput_pass"`
	RequiredTrails     int          `json:"required_trails"`
	MeasuredTrails     int          `json:"measured_trails"`
	ScalePass          bool         `json:"scale_pass"`
	RebuildBudgetMS    float64      `json:"full_rebuild_budget_ms"`
	FullRebuildMS      float64      `json:"full_rebuild_ms"`
	RebuildPass        bool         `json:"full_rebuild_pass"`
}

type PhaseReport struct {
	PrepareMS              float64           `json:"prepare_ms"`
	SubmitMS               float64           `json:"submit_ms"`
	ReadyMS                float64           `json:"ready_ms"`
	RequestMS              float64           `json:"request_ms,omitempty"`
	SubmittedDocuments     int               `json:"submitted_documents,omitempty"`
	SubmittedBytes         int64             `json:"submitted_bytes,omitempty"`
	TaskCount              int               `json:"task_count,omitempty"`
	BarrierCount           int               `json:"barrier_count,omitempty"`
	EngineBaselineBytes    int64             `json:"engine_baseline_bytes,omitempty"`
	EnginePeakBytes        int64             `json:"engine_peak_bytes,omitempty"`
	EnginePeakDelta        int64             `json:"engine_peak_delta_bytes,omitempty"`
	ClientBaselineRSSBytes int64             `json:"client_baseline_rss_bytes,omitempty"`
	ClientPeakRSSBytes     int64             `json:"client_peak_rss_bytes,omitempty"`
	ClientPeakRSSDelta     int64             `json:"client_peak_rss_delta_bytes,omitempty"`
	DiskBytes              int64             `json:"disk_bytes,omitempty"`
	UsedDiskBytes          int64             `json:"used_disk_bytes,omitempty"`
	FilesystemBytes        int64             `json:"filesystem_bytes,omitempty"`
	DocumentCount          int64             `json:"document_count,omitempty"`
	LastBatch              *MeiliBatchReport `json:"last_batch,omitempty"`
}

type MeiliBatchReport struct {
	UID                int               `json:"uid"`
	TaskCount          int               `json:"task_count,omitempty"`
	Strategy           string            `json:"strategy,omitempty"`
	Duration           string            `json:"duration,omitempty"`
	ProgressPercent    float64           `json:"progress_percent,omitempty"`
	ProgressSteps      []string          `json:"progress_steps,omitempty"`
	ProgressTrace      map[string]string `json:"progress_trace,omitempty"`
	WriteBlockingRatio float64           `json:"write_blocking_ratio,omitempty"`
}

type QueryReport struct {
	Scenario              string       `json:"scenario"`
	RadiusMeters          float64      `json:"radius_meters"`
	Queries               int          `json:"queries"`
	Samples               int          `json:"samples"`
	FirstSampleMS         float64      `json:"first_sample_ms"`
	CandidateLatency      LatencyStats `json:"candidate_latency_ms"`
	VerifyLatency         LatencyStats `json:"verify_latency_ms"`
	DatabaseLatency       LatencyStats `json:"database_load_decode_latency_ms"`
	ExactLatency          LatencyStats `json:"exact_geometry_latency_ms"`
	TotalLatency          LatencyStats `json:"total_latency_ms"`
	EngineProcessing      LatencyStats `json:"engine_processing_ms"`
	AverageCandidates     float64      `json:"average_candidates"`
	AverageMatches        float64      `json:"average_matches"`
	ExpectedMatches       int          `json:"expected_matches"`
	AverageExpected       float64      `json:"average_expected_matches"`
	AverageLoadedRows     float64      `json:"average_database_rows_loaded"`
	AverageDatabaseBytes  float64      `json:"average_database_bytes_loaded"`
	TruePositives         int          `json:"true_positives"`
	FalsePositives        int          `json:"false_positives"`
	FalseNegatives        int          `json:"false_negatives"`
	CandidateRecall       float64      `json:"candidate_recall"`
	Precision             float64      `json:"precision"`
	Recall                float64      `json:"recall"`
	F1                    float64      `json:"f1"`
	BoundaryExcludedCount int          `json:"boundary_excluded_count"`
}

type ProductWorkloadReport struct {
	Status               string                `json:"status"`
	SetupMS              float64               `json:"setup_ms"`
	OracleMS             float64               `json:"oracle_ms"`
	DatabaseBytes        int64                 `json:"application_database_bytes"`
	Clients              int                   `json:"clients"`
	Cases                int                   `json:"cases"`
	CorrectCases         int                   `json:"correct_cases"`
	Samples              int                   `json:"samples"`
	IncorrectCases       int                   `json:"incorrect_cases"`
	IncorrectWarmSamples int                   `json:"incorrect_warm_samples"`
	FirstSampleLatency   LatencyStats          `json:"first_sample_latency_ms"`
	CandidateLatency     LatencyStats          `json:"candidate_latency_ms"`
	DatabaseLatency      LatencyStats          `json:"database_load_decode_latency_ms"`
	FinalizeLatency      LatencyStats          `json:"acl_filter_exact_sort_page_latency_ms"`
	TotalLatency         LatencyStats          `json:"total_latency_ms"`
	AverageCandidates    float64               `json:"average_candidates"`
	AverageLoadedRows    float64               `json:"average_database_rows_loaded"`
	AverageEncodedBytes  float64               `json:"average_database_bytes_loaded"`
	AverageResponseBytes float64               `json:"average_response_bytes"`
	AverageExactChecks   float64               `json:"average_exact_geometry_checks"`
	ConcurrentWallMS     float64               `json:"concurrent_wall_ms"`
	ThroughputQPS        float64               `json:"throughput_queries_per_second"`
	RadiusReports        []ProductRadiusReport `json:"radius_reports"`
	CaseReports          []ProductCaseReport   `json:"case_reports"`
}

type ProductRadiusReport struct {
	RadiusMeters      float64      `json:"radius_meters"`
	Samples           int          `json:"samples"`
	IncorrectSamples  int          `json:"incorrect_samples"`
	TotalLatency      LatencyStats `json:"total_latency_ms"`
	AverageCandidates float64      `json:"average_candidates"`
}

type ProductCaseReport struct {
	Name                   string   `json:"name"`
	QueryID                string   `json:"query_id"`
	Scenario               string   `json:"scenario"`
	RadiusMeters           float64  `json:"radius_meters"`
	ActorID                string   `json:"actor_id,omitempty"`
	Text                   string   `json:"text,omitempty"`
	Sort                   string   `json:"sort"`
	Page                   int      `json:"page"`
	PerPage                int      `json:"per_page"`
	ExpectedTotalItems     int      `json:"expected_total_items"`
	ActualTotalItems       int      `json:"actual_total_items"`
	ExpectedHitIDs         []string `json:"expected_hit_ids"`
	ActualHitIDs           []string `json:"actual_hit_ids"`
	CandidateCount         int      `json:"candidate_count"`
	LoadedRows             int      `json:"loaded_rows"`
	EncodedBytes           int64    `json:"encoded_bytes"`
	ResponseBytes          int64    `json:"response_bytes"`
	ExactChecks            int      `json:"exact_checks"`
	MissingExpectedMatches int      `json:"missing_expected_matches"`
	Correct                bool     `json:"correct"`
	Error                  string   `json:"error,omitempty"`
}

type LatencyStats struct {
	Min  float64 `json:"min"`
	P50  float64 `json:"p50"`
	P95  float64 `json:"p95"`
	P99  float64 `json:"p99"`
	Max  float64 `json:"max"`
	Mean float64 `json:"mean"`
}

type CandidateResult struct {
	IDs                map[string]struct{}
	WallDuration       time.Duration
	EngineProcessingMS float64
	EstimatedTotalHits int64
	Truncated          bool
}
