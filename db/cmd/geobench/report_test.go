package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarkdownReportShowsSearchOnlyScopeWithoutContractSection(t *testing.T) {
	markdown := markdownReport(BenchmarkReport{
		Config: RunConfig{GeoJSONIndex: GeoJSONIndexConfig{ShardMode: geoJSONShardModeCapacityCreated}},
		Backends: []BackendReport{{
			Backend:        "geojson",
			ExecutionScope: geoJSONCapacitySearchScope,
		}},
	})
	if strings.Contains(markdown, "## GeoJSON contract") {
		t.Fatal("empty contract section unexpectedly rendered")
	}
	if !strings.Contains(markdown, geoJSONCapacitySearchScope) {
		t.Fatalf("search-only scope missing from markdown:\n%s", markdown)
	}
}

func TestContractReportsGeoJSONCoverageLimitationWithoutFailingTopLevel(t *testing.T) {
	report := BenchmarkReport{
		Contracts: []ContractReport{{
			MeilisearchVersion: "1.53.1",
			Status:             "passed",
			Diagnostics: []GeoJSONContractDiagnosticReport{{
				Name:       "geojson_cell_coverage",
				Status:     "known_engine_limitation",
				Cause:      "planar Cellulite shape-cell coverage mismatch",
				QueryPoint: Coordinate{Lat: 79.1011091432348, Lon: -17.696734637372288},
				Cases: []GeoJSONContractDiagnosticCaseReport{{
					Name:        "100 km point and line coverage",
					Filter:      "_geoRadius(79.1011091, -17.6967346, 100000, 100)",
					ExpectedIDs: []string{"high-latitude-line", "high-latitude-query-point"},
					ActualIDs:   []string{},
					MissingIDs:  []string{"high-latitude-line", "high-latitude-query-point"},
				}},
			}, {
				Name:   "geojson_hierarchical_split",
				Status: "known_engine_limitation",
				Cause:  "non-geometric H3 parent optimization",
				Counts: []GeoJSONContractDiagnosticCountReport{{
					Name:          "GeoJSON polygon over identical extent",
					Filter:        "_geoPolygon([50.9, 6.9], [50.9, 7.7], [51.7, 7.7], [51.7, 6.9])",
					ExpectedCount: 5_000,
					ActualCount:   4_980,
					MissingCount:  20,
				}},
			}},
		}},
	}
	markdown := markdownReport(report)
	for _, wanted := range []string{
		"### Additive contract diagnostics",
		"Diagnostics with status `passed` or `known_engine_limitation` do not change the top-level contract status",
		"A `diagnostic_error` is gating",
		"| 1.53.1 | geojson_cell_coverage | known_engine_limitation | planar Cellulite shape-cell coverage mismatch |",
		"`_geoRadius(79.1011091, -17.6967346, 100000, 100)`",
		"high-latitude-line, high-latitude-query-point | ∅ | high-latitude-line, high-latitude-query-point",
		"| 1.53.1 | GeoJSON polygon over identical extent | `_geoPolygon([50.9, 6.9], [50.9, 7.7], [51.7, 7.7], [51.7, 6.9])` | 5000 | 4980 | 20 |",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("markdown is missing %q:\n%s", wanted, markdown)
		}
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(encoded)
	for _, wanted := range []string{
		`"status":"passed"`,
		`"status":"known_engine_limitation"`,
		`"cause":"planar Cellulite shape-cell coverage mismatch"`,
		`"expected_ids":["high-latitude-line","high-latitude-query-point"]`,
		`"actual_ids":[]`,
		`"expected_count":5000`,
		`"actual_count":4980`,
		`"missing_count":20`,
	} {
		if !strings.Contains(jsonText, wanted) {
			t.Fatalf("JSON is missing %q: %s", wanted, jsonText)
		}
	}
}

func TestMarkdownReportShowsDirectCountQualificationAndDiagnostics(t *testing.T) {
	report := BenchmarkReport{Backends: []BackendReport{{
		Backend:            "geojson",
		MeilisearchVersion: "1.53.1",
		GeoJSONDirect: &GeoJSONDirectReport{
			SelectedID: "r250-exact",
			ProductWorkload: &GeoJSONDirectProductWorkloadReport{
				CountDiagnostics: GeoJSONDirectProductCountReport{
					Status:                  "passed",
					MaximumP95RelativeError: 0.01,
					Cases:                   1,
					WithinOnePercentCases:   1,
					WithinOnePercentRate:    1,
					TotalPagesMatchCases:    1,
					TotalPagesMatchRate:     1,
					CountError: GeoJSONDirectCountError{
						MeanAbsolute: 1,
						P95Absolute:  1,
						MaxAbsolute:  1,
						MeanRelative: 0.005,
						P95Relative:  0.005,
						MaxRelative:  0.005,
					},
				},
				CaseReports: []GeoJSONDirectProductCaseReport{{
					Name:                   "count case",
					QueryID:                "query-3",
					RadiusMeters:           5_000,
					ExpectedTotalItems:     200,
					ActualTotalItems:       201,
					SignedCountDelta:       1,
					FalseNegatives:         2,
					BoundaryFalseNegatives: 1,
					MaterialFalseNegatives: 1,
					FalsePositives:         3,
					BoundaryFalsePositives: 2,
					MaterialFalsePositives: 1,
					ExpectedTotalPages:     20,
					ActualTotalPages:       20,
				}},
			},
		},
	}}}

	markdown := markdownReport(report)
	for _, wanted := range []string{
		"### Direct product count diagnostics",
		"0/1 (0.0000)",
		"1/1 (1.0000)",
		"200/201/+1",
		"2/1/1",
		"3/2/1/0",
		"20/20",
		"passed",
		"normative 1% count gate",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("count diagnostic markdown is missing %q:\n%s", wanted, markdown)
		}
	}
}

func TestMarkdownReportSeparatesHybridDatabaseAndResponseBytes(t *testing.T) {
	markdown := markdownReport(BenchmarkReport{Backends: []BackendReport{{
		Backend:            "h3",
		MeilisearchVersion: "1.53.1",
		ProductWorkload: &ProductWorkloadReport{
			Status:               "correct",
			Cases:                1,
			CorrectCases:         1,
			Clients:              2,
			AverageEncodedBytes:  2048,
			AverageResponseBytes: 512,
		},
	}}})
	for _, wanted := range []string{
		"DB loaded avg | Final response avg",
		"| h3 | 1.53.1 | 1/1",
		"| 2.0 KiB | 512.0 B |",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("markdown is missing %q:\n%s", wanted, markdown)
		}
	}
}

func TestMarkdownReportIncludesGeoJSONShardBuildsWithoutDoubleCountingGauges(t *testing.T) {
	report := BenchmarkReport{
		Config: RunConfig{GeoJSONIndex: GeoJSONIndexConfig{Shards: 2}},
		Backends: []BackendReport{{
			Backend:            "geojson",
			MeilisearchVersion: "1.53.1",
			IndexSetupMS:       42,
			FullIndex: PhaseReport{
				ReadyMS:            1200,
				DocumentCount:      20,
				UsedDiskBytes:      4096,
				EnginePeakDelta:    8192,
				SubmittedDocuments: 20,
			},
			GeoJSONIndex: &GeoJSONIndexReport{
				Shards:               2,
				ShardDocumentCounts:  []int{9, 11},
				ShardIndexedVertices: []int{90, 120},
				ShardIndexedSegments: []int{81, 109},
				ShardBuilds: []GeoJSONShardBuildReport{
					{Shard: 0, IndexUID: "shard-0", Phase: PhaseReport{DocumentCount: 9, ReadyMS: 900, SubmittedBytes: 1024, TaskCount: 1, LastBatch: &MeiliBatchReport{ProgressPercent: 100}}},
					{Shard: 1, IndexUID: "shard-1", Phase: PhaseReport{DocumentCount: 11, ReadyMS: 1200, SubmittedBytes: 2048, TaskCount: 1, LastBatch: &MeiliBatchReport{ProgressPercent: 100}}},
				},
			},
		}},
	}
	markdown := markdownReport(report)
	for _, wanted := range []string{
		"Index setup",
		"### GeoJSON shard builds",
		"| 1.53.1 | 0 | shard-0 | 9 | 90 | 81 | 900.0 ms | 1 | 1.0 KiB | 100.0%",
		"| 1.53.1 | 1 | shard-1 | 11 | 120 | 109 | 1200.0 ms | 1 | 2.0 KiB | 100.0%",
		"RSS and storage are measured once",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("markdown is missing %q:\n%s", wanted, markdown)
		}
	}
	if strings.Contains(markdown, "%!") {
		t.Fatalf("markdown contains a formatting error:\n%s", markdown)
	}
}

func TestMarkdownReportIncludesGeoJSONNormalizationConfigurationAndMetrics(t *testing.T) {
	markdown := markdownReport(BenchmarkReport{
		Config: RunConfig{GeoJSONIndex: GeoJSONIndexConfig{
			MaxSegmentLengthMeters: 5_000,
		}},
		Backends: []BackendReport{{
			Backend:            "geojson",
			MeilisearchVersion: "1.53.1",
			GeoJSONIndex: &GeoJSONIndexReport{
				SimplifyToleranceMeters:   50,
				MaxSegmentLengthMeters:    5_000,
				SourceVertices:            1_000,
				PreNormalizationVertices:  400,
				DensifiedVertices:         200,
				AntimeridianCuts:          3,
				PostNormalizationVertices: 606,
				SourceSegments:            900,
				PreNormalizationSegments:  300,
				PostNormalizationSegments: 600,
				VertexReductionPercent:    60,
			},
		}},
	})
	for _, wanted := range []string{
		"GeoJSON max segment: 5000.0 m",
		"Pre-normalization vertices",
		"Densified vertices",
		"Antimeridian cuts",
		"| 1.53.1 | 50.0 m | 5000.0 m |",
		"| 1000 | 400 | 200 | 3 | 606 | 60.00% | 900 | 300 | 600 |",
		"RFC 7946-compatible LineStrings",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("markdown is missing %q:\n%s", wanted, markdown)
		}
	}
}

func TestMarkdownReportClarifiesDirectExactPlanIdentifier(t *testing.T) {
	markdown := markdownReport(BenchmarkReport{Backends: []BackendReport{{
		Backend: "geojson",
		GeoJSONDirect: &GeoJSONDirectReport{
			SelectedID: "r100-exact",
			Plan:       &GeoJSONQueryPlan{ID: "r100-exact", Resolution: 100},
		},
	}}})
	if !strings.Contains(markdown, "does **not** mean exact route geometry or exact results") {
		t.Fatalf("direct plan identifier remains ambiguous:\n%s", markdown)
	}
}

func TestMarkdownReportIncludesGeoJSONSweepAndPerformanceGate(t *testing.T) {
	report := BenchmarkReport{
		Config: RunConfig{PerformanceTarget: "ds923plus"},
		Backends: []BackendReport{{
			Backend:            "geojson",
			MeilisearchVersion: "1.53.1",
			GeoJSONSweep: &GeoJSONSweepReport{
				SelectedID:       "r125-safe-p0p5",
				SelectionStatus:  "qualified",
				SelectionSamples: 5_040,
				Variants: []GeoJSONVariantReport{{
					Plan: GeoJSONQueryPlan{
						ID:                 "r125-safe-p0p5",
						RadiusMode:         geoJSONRadiusSafe,
						Resolution:         125,
						ExtraPaddingMeters: 0.5,
					},
					Status:          "ok",
					Finalist:        true,
					CandidateRecall: 1,
					ProductWorkload: &ProductWorkloadReport{
						Status:       "correct",
						TotalLatency: LatencyStats{P95: 42},
					},
				}},
			},
			Performance: &PerformanceReport{Target: "ds923plus", Status: "passed", MinimumSamples: 5_000, MeasuredSamples: 5_040, RequiredClients: 4, MeasuredClients: 4, CorrectnessPass: true},
		}},
	}
	markdown := markdownReport(report)
	for _, wanted := range []string{
		"## GeoJSON parameter sweep",
		"r125-safe-p0p5",
		"| yes | qualified (5040 samples) | ok | inscribed_safe | 125 | 0.50 m",
		"42.00 ms",
		"## Performance target",
		"5040/5000",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("markdown is missing %q:\n%s", wanted, markdown)
		}
	}
	if strings.Contains(markdown, "## GeoJSON direct UX path") {
		t.Fatalf("legacy report unexpectedly contains a direct UX section:\n%s", markdown)
	}
	for _, unwanted := range []string{"### Direct product audits", "### Direct warm qualification by plan", "## GeoJSON direct performance target"} {
		if strings.Contains(markdown, unwanted) {
			t.Fatalf("legacy report unexpectedly contains %q:\n%s", unwanted, markdown)
		}
	}
}

func TestMarkdownReportIncludesGeoJSONDirectUXEvidence(t *testing.T) {
	passed := GeoJSONDirectAccuracy{
		Status:                         "passed",
		BoundaryToleranceMeters:        50,
		MinimumRecall:                  0.99,
		MinimumPrecision:               0.99,
		MinimumTop10Recall:             0.99,
		MinimumComparablePageRecall:    0.99,
		MinimumComparablePagePrecision: 0.99,
		Requests:                       40,
		RequestsWithExpected:           39,
		RequestsWithClearExpected:      39,
		ReturnedHits:                   1001,
		ExpectedHits:                   1000,
		TruePositives:                  999,
		FalsePositives:                 2,
		FalseNegatives:                 1,
		ClearExpectedHits:              950,
		ClearTruePositives:             950,
		BoundaryMisses:                 1,
		AllowedBoundaryFalsePositives:  2,
		NearestExpected:                39,
		NearestReturned:                39,
		Top10Expected:                  320,
		Top10Returned:                  320,
		Recall:                         0.999,
		Precision:                      1,
		ClearRecall:                    1,
		NearestRecall:                  1,
		Top10Recall:                    1,
		NonEmptyRequestSuccessRate:     1,
		WorstRequestRecall:             1,
		MaxInsideMissMeters:            2.4,
		MaxOutsideHitMeters:            3.1,
		CountError: GeoJSONDirectCountError{
			MeanAbsolute: 0.2,
			P95Absolute:  1,
			MaxAbsolute:  2,
			MeanRelative: 0.001,
			P95Relative:  0.005,
			MaxRelative:  0.01,
		},
	}
	failed := passed
	failed.Status = "failed"
	failed.MaterialFalseNegatives = 1
	failed.MaterialFalsePositives = 2
	failed.FalseEmptyRequests = 1
	failed.Violations = []GeoJSONDirectCaseViolation{{
		QueryID:               "query-17",
		Scenario:              "alpine_dense",
		RadiusMeters:          5000,
		Expected:              12,
		Returned:              13,
		FalseNegatives:        2,
		MaterialFalseNegative: 1,
		FalsePositives:        3,
		MaterialFalsePositive: 2,
		Recall:                10.0 / 12.0,
		MaxInsideMissMeters:   72,
		MaxOutsideHitMeters:   88,
		MaterialFalseNegativeTrails: []GeoJSONDirectTrailViolationDetail{{
			TrailID: "trail-missed", DistanceMeters: 4928, RadiusDeltaMeters: 72,
		}},
		MaterialFalsePositiveTrails: []GeoJSONDirectTrailViolationDetail{{
			TrailID: "trail-extra", DistanceMeters: 5088, RadiusDeltaMeters: 88,
		}},
		FalseEmpty: true,
		Error:      "material spatial mismatch",
	}}
	productAccuracy := passed
	productAccuracy.ComparablePageExpected = 30
	productAccuracy.ComparablePageReturned = 30
	productAccuracy.ComparablePageActual = 30
	productAccuracy.ComparablePageCases = 5
	productAccuracy.ComparablePageRecall = 1
	productAccuracy.ComparablePagePrecision = 1
	productAccuracy.WorstComparablePageRecall = 1
	productAccuracy.WorstComparablePagePrecision = 1
	productAccuracy.UnsupportedSortCases = 1

	report := BenchmarkReport{
		Config: RunConfig{
			GeoJSONDirect:     GeoJSONDirectConfig{Enabled: true},
			FailOnUX:          true,
			PerformanceTarget: "ds923plus",
		},
		Backends: []BackendReport{{
			Backend:            "geojson",
			MeilisearchVersion: "1.53.1",
			Performance: &PerformanceReport{
				Target:          "ds923plus",
				Status:          "failed",
				CorrectnessPass: true,
				Reason:          "exact path exceeds latency budget",
			},
			GeoJSONSweep: &GeoJSONSweepReport{Variants: []GeoJSONVariantReport{
				{Plan: GeoJSONQueryPlan{ID: "r100-exact", RadiusMode: geoJSONRadiusExact, Resolution: 100}, DirectUX: &failed},
				{Plan: GeoJSONQueryPlan{ID: "r125-exact", RadiusMode: geoJSONRadiusExact, Resolution: 125}, DirectUX: &passed},
			}},
			GeoJSONDirect: &GeoJSONDirectReport{
				SelectedID:      "r125-exact",
				SelectionStatus: "product_qualified",
				SelectionRule:   "UX pass then lowest p95",
				Plan:            &GeoJSONQueryPlan{ID: "r125-exact", RadiusMode: geoJSONRadiusExact, Resolution: 125},
				Accuracy:        &passed,
				SpatialAudit: &GeoJSONDirectSpatialAuditReport{
					Status: "failed", Accuracy: failed,
				},
				ProductAudits: []GeoJSONDirectProductAuditReport{
					{
						PlanID:             "r100-exact",
						Resolution:         100,
						Status:             "failed",
						Cases:              6,
						ProductPageLatency: LatencyStats{P95: 31, P99: 44},
						Accuracy:           failed,
						Error:              "material miss",
					},
					{
						PlanID:             "r125-exact",
						Resolution:         125,
						Status:             "passed",
						Cases:              6,
						ProductPageLatency: LatencyStats{P95: 18, P99: 27},
						Accuracy:           productAccuracy,
						WarmWorkload: &GeoJSONDirectProductWorkloadReport{
							Status: "passed", Clients: 4, Cases: 6, ExpectedSamples: 5040, Samples: 5040,
							TotalLatency: LatencyStats{P50: 8, P95: 21, P99: 35}, ThroughputQPS: 19.5,
							Accuracy: productAccuracy,
						},
						Performance: &PerformanceReport{Target: "ds923plus", Status: "passed"},
					},
				},
				ProductWorkload: &GeoJSONDirectProductWorkloadReport{
					Status:               "passed",
					Clients:              4,
					Cases:                6,
					ExpectedSamples:      5040,
					Samples:              5040,
					AverageResponseBytes: 2048,
					TotalLatency:         LatencyStats{P50: 8, P95: 21, P99: 35},
					EngineProcessing:     LatencyStats{P95: 14},
					ThroughputQPS:        19.5,
					Accuracy:             productAccuracy,
					CaseReports: []GeoJSONDirectProductCaseReport{{
						Name:               "numeric filters and proximity sort",
						QueryID:            "query-3",
						Sort:               "proximity_asc",
						SortSupported:      false,
						PageComparable:     false,
						PagePrecision:      1,
						ExpectedTotalItems: 12,
						ActualTotalItems:   12,
						CorrectSemantics:   true,
						Error:              "exact line proximity is unsupported",
					}},
				},
				Performance: &PerformanceReport{
					Target:             "ds923plus",
					Status:             "passed",
					MinimumSamples:     5_000,
					MeasuredSamples:    5_040,
					RequiredClients:    4,
					MeasuredClients:    4,
					MinimumThroughput:  4,
					MeasuredThroughput: 19.5,
					RequiredTrails:     50_000,
					MeasuredTrails:     50_000,
					RebuildBudgetMS:    30 * 60 * 1000,
					FullRebuildMS:      12 * 60 * 1000,
					NormalWorst:        LatencyStats{P50: 8, P95: 21, P99: 35},
					BroadWorst:         LatencyStats{P50: 15, P95: 80, P99: 120},
					CorrectnessPass:    true,
				},
			},
		}},
	}

	markdown := markdownReport(report)
	for _, wanted := range []string{
		"Direct GeoJSON UX path: `true`; fail on UX: `true`",
		"## GeoJSON direct UX path",
		"`geojson_direct.plan.index_simplification_meters`",
		"### Selection and policy",
		"r125-exact",
		"product_qualified",
		"±50.0 m",
		"### Direct product audits",
		"Calibrated product-page p95/p99",
		"| 1.53.1 | r125-exact | 125 | passed | 6 | 18.00/27.00 ms | 0.9990 | 1.0000 | 1.0000/1.0000 | — |",
		"### Direct warm qualification by plan",
		"| 1.53.1 | r125-exact | yes | passed | 5040/5040 | 4 | 8.00/21.00/35.00 ms | 19.5 q/s | 0 | ds923plus: passed | — |",
		"### Spatial screen accuracy",
		"r100-exact",
		"1/2",
		"### Direct count error",
		"0.20/1.00/2",
		"### Direct product workload",
		"Non-empty success",
		"8.00/21.00/35.00 ms",
		"2.0 KiB",
		"### Direct UX violations",
		"query-17",
		"r125-exact capacity spatial",
		"72.00 m",
		"trail-missed @ 4928.0 m (inside 72.0 m)",
		"trail-extra @ 5088.0 m (outside 88.0 m)",
		"### Direct product limitations and mismatches",
		"proximity_asc",
		"exact line proximity is unsupported",
		"## Performance target",
		"exact path exceeds latency budget",
		"## GeoJSON direct performance target",
		"| geojson | 1.53.1 | r125-exact | ds923plus | passed | 50000/50000 | 5040/5000 | 4/4 | 19.5/4.0 q/s",
		"Its correctness column refers to the explicit GeoJSON UX contract",
	} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("direct UX markdown is missing %q:\n%s", wanted, markdown)
		}
	}
}

func TestMarkdownReportOmitsNewDirectTablesForOlderDirectReports(t *testing.T) {
	accuracy := GeoJSONDirectAccuracy{Status: "passed", Requests: 1}
	report := BenchmarkReport{Backends: []BackendReport{{
		Backend:            "geojson",
		MeilisearchVersion: "1.44.0",
		GeoJSONDirect: &GeoJSONDirectReport{
			SelectedID:      "r125-exact",
			SelectionStatus: "spatial_qualified",
			Accuracy:        &accuracy,
		},
	}}}

	markdown := markdownReport(report)
	if !strings.Contains(markdown, "## GeoJSON direct UX path") {
		t.Fatalf("older direct report lost its existing direct section:\n%s", markdown)
	}
	for _, unwanted := range []string{"### Direct product audits", "## GeoJSON direct performance target"} {
		if strings.Contains(markdown, unwanted) {
			t.Fatalf("older direct report unexpectedly contains %q:\n%s", unwanted, markdown)
		}
	}
}
