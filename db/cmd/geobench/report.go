package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func writeReports(report BenchmarkReport, output string) (string, string, error) {
	jsonPath, markdownPath, err := reportPaths(output)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o750); err != nil {
		return "", "", err
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", "", err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(jsonPath, encoded, 0o640); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(markdownPath, []byte(markdownReport(report)), 0o640); err != nil {
		return "", "", err
	}
	return jsonPath, markdownPath, nil
}

func reportPaths(output string) (string, string, error) {
	if output == "" {
		directory, err := os.MkdirTemp("", "wanderer-geobench-results-")
		if err != nil {
			return "", "", err
		}
		return filepath.Join(directory, "report.json"), filepath.Join(directory, "report.md"), nil
	}
	abs, err := filepath.Abs(output)
	if err != nil {
		return "", "", err
	}
	if strings.EqualFold(filepath.Ext(abs), ".json") {
		return abs, strings.TrimSuffix(abs, filepath.Ext(abs)) + ".md", nil
	}
	return filepath.Join(abs, "report.json"), filepath.Join(abs, "report.md"), nil
}

func markdownReport(report BenchmarkReport) string {
	var builder strings.Builder
	builder.WriteString("# Wanderer geo benchmark\n\n")
	fmt.Fprintf(&builder, "Run: `%s`  \n", report.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(&builder, "Profile: `%s`; query workload: `%s`; index only: `%t`; product clients: `%d`; performance target: `%s`; fail on incorrect: `%t`; backend time budget: `%s`; overall time budget: `%s`  \n", report.Config.Profile, report.Config.QueryWorkload, report.Config.IndexOnly, report.Config.QueryClients, report.Config.PerformanceTarget, report.Config.FailOnIncorrect, report.Config.BackendTimeout, report.Config.OverallTimeout)
	if report.Config.GeoJSONDirect.Enabled || report.Config.FailOnUX {
		fmt.Fprintf(&builder, "Direct GeoJSON UX path: `%t`; fail on UX: `%t`  \n", report.Config.GeoJSONDirect.Enabled, report.Config.FailOnUX)
	}
	fmt.Fprintf(&builder, "Dataset: %d trails, %d vertices, %d segments, %d query points  \n", report.Dataset.Trails, report.Dataset.Vertices, report.Dataset.Segments, report.Dataset.QueryPoints)
	fmt.Fprintf(&builder, "Radii: `%v` meters; repetitions: %d; batch size: %d documents / %s; max pending batches: %d; Meili indexing threads: %d; GeoJSON simplification: %.1f m; GeoJSON max segment: %.1f m; GeoJSON shard mode: `%s`; requested hash shards: %d; capacity: %s; capacity arrival: `%s`; incremental updates: %d\n\n", report.Config.RadiiMeters, report.Config.Repetitions, report.Config.BatchSize, formatBytes(int64(report.Config.BatchBytes)), report.Config.MaxPendingBatches, report.Config.MeiliIndexThreads, report.Config.GeoJSONIndex.SimplifyToleranceMeters, report.Config.GeoJSONIndex.MaxSegmentLengthMeters, normalizedGeoJSONShardMode(report.Config.GeoJSONIndex.ShardMode), report.Config.GeoJSONIndex.Shards, geoJSONCapacityLabel(report.Config.GeoJSONIndex.MaxTrailsPerShard), emptyDash(report.Config.GeoJSONIndex.CapacityArrival), report.Config.IncrementalCount)
	if report.OracleDurationMS > 0 {
		fmt.Fprintf(&builder, "Exact-reference preparation: %.1f ms  \n\n", report.OracleDurationMS)
	}

	if len(report.Environment) > 0 {
		keys := make([]string, 0, len(report.Environment))
		for key := range report.Environment {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder.WriteString("Environment: ")
		for index, key := range keys {
			if index > 0 {
				builder.WriteString(", ")
			}
			fmt.Fprintf(&builder, "%s=`%s`", key, report.Environment[key])
		}
		builder.WriteString("\n\n")
	}

	if len(report.Reproducibility.Invocation) > 0 || report.Reproducibility.ExecutableSHA256 != "" || report.Reproducibility.HarnessSHA256 != "" || report.Reproducibility.DatasetSHA256 != "" || len(report.Reproducibility.DockerImages) > 0 || len(report.Reproducibility.Warnings) > 0 {
		builder.WriteString("## Reproducibility\n\n")
		if len(report.Reproducibility.Invocation) > 0 {
			encoded, _ := json.Marshal(report.Reproducibility.Invocation)
			builder.WriteString("Invocation:\n\n```json\n")
			builder.Write(encoded)
			builder.WriteString("\n```\n\n")
		}
		if report.Reproducibility.ExecutableSHA256 != "" {
			fmt.Fprintf(&builder, "Executable: `%s`  \n", report.Reproducibility.ExecutableSHA256)
		}
		if report.Reproducibility.HarnessSHA256 != "" {
			fmt.Fprintf(&builder, "Harness: `%s`  \n", report.Reproducibility.HarnessSHA256)
		}
		if report.Reproducibility.DatasetSHA256 != "" {
			fmt.Fprintf(&builder, "Dataset: `%s`  \n", report.Reproducibility.DatasetSHA256)
		}
		if report.Reproducibility.ExecutableSHA256 != "" || report.Reproducibility.HarnessSHA256 != "" || report.Reproducibility.DatasetSHA256 != "" {
			builder.WriteString("\n")
		}
		if len(report.Reproducibility.DockerImages) > 0 {
			builder.WriteString("| Docker image | Local image ID | Repository digest(s) | Resolution error |\n")
			builder.WriteString("|---|---|---|---|\n")
			for _, image := range report.Reproducibility.DockerImages {
				fmt.Fprintf(
					&builder,
					"| %s | %s | %s | %s |\n",
					escapeCell(image.Reference),
					emptyDash(image.ImageID),
					emptyDash(strings.Join(image.RepoDigests, "<br>")),
					emptyDash(image.Error),
				)
			}
			builder.WriteString("\n")
		}
		if len(report.Reproducibility.Warnings) > 0 {
			builder.WriteString("Metadata warnings:\n\n")
			for _, warning := range report.Reproducibility.Warnings {
				fmt.Fprintf(&builder, "- %s\n", strings.ReplaceAll(warning, "\n", " "))
			}
			builder.WriteString("\n")
		}
	}

	if len(report.Contracts) > 0 {
		builder.WriteString("## GeoJSON contract\n\n")
		builder.WriteString("| Meilisearch | Status | Time | Error |\n")
		builder.WriteString("|---|---:|---:|---|\n")
		for _, contract := range report.Contracts {
			fmt.Fprintf(&builder, "| %s | %s | %.0f ms | %s |\n", escapeCell(contract.MeilisearchVersion), contract.Status, contract.DurationMS, escapeCell(contract.Error))
		}
		builder.WriteString("\n")

		diagnosticCount := 0
		for _, contract := range report.Contracts {
			diagnosticCount += len(contract.Diagnostics)
		}
		if diagnosticCount > 0 {
			builder.WriteString("### Additive contract diagnostics\n\n")
			builder.WriteString("Diagnostics with status `passed` or `known_engine_limitation` do not change the top-level contract status or the aggregate Direct UX gate. A `diagnostic_error` is gating: the probe produced no conclusion and fails the contract, or classifies it as timed out when its deadline expired.\n\n")
			builder.WriteString("| Meilisearch | Diagnostic | Status | Cause | Error |\n")
			builder.WriteString("|---|---|---:|---|---|\n")
			for _, contract := range report.Contracts {
				for _, diagnostic := range contract.Diagnostics {
					fmt.Fprintf(
						&builder,
						"| %s | %s | %s | %s | %s |\n",
						escapeCell(contract.MeilisearchVersion),
						escapeCell(diagnostic.Name),
						escapeCell(diagnostic.Status),
						emptyDash(diagnostic.Cause),
						emptyDash(diagnostic.Error),
					)
				}
			}
			builder.WriteString("\n")
			builder.WriteString("| Meilisearch | Diagnostic case | Filter | Expected IDs | Found IDs | Missing IDs | Unexpected IDs |\n")
			builder.WriteString("|---|---|---|---|---|---|---|\n")
			for _, contract := range report.Contracts {
				for _, diagnostic := range contract.Diagnostics {
					for _, testCase := range diagnostic.Cases {
						fmt.Fprintf(
							&builder,
							"| %s | %s | `%s` | %s | %s | %s | %s |\n",
							escapeCell(contract.MeilisearchVersion),
							escapeCell(testCase.Name),
							escapeCell(testCase.Filter),
							formatContractIDs(testCase.ExpectedIDs),
							formatContractIDs(testCase.ActualIDs),
							formatContractIDs(testCase.MissingIDs),
							formatContractIDs(testCase.UnexpectedIDs),
						)
					}
				}
			}
			builder.WriteString("\n")
			countDiagnosticCount := 0
			for _, contract := range report.Contracts {
				for _, diagnostic := range contract.Diagnostics {
					countDiagnosticCount += len(diagnostic.Counts)
				}
			}
			if countDiagnosticCount > 0 {
				builder.WriteString("| Meilisearch | Diagnostic count | Filter | Expected | Found | Missing |\n")
				builder.WriteString("|---|---|---|---:|---:|---:|\n")
				for _, contract := range report.Contracts {
					for _, diagnostic := range contract.Diagnostics {
						for _, count := range diagnostic.Counts {
							fmt.Fprintf(
								&builder,
								"| %s | %s | `%s` | %d | %d | %d |\n",
								escapeCell(contract.MeilisearchVersion),
								escapeCell(count.Name),
								escapeCell(count.Filter),
								count.ExpectedCount,
								count.ActualCount,
								count.MissingCount,
							)
						}
					}
				}
				builder.WriteString("\n")
			}
		}
	}
	for _, backend := range report.Backends {
		if backend.ExecutionScope != "" {
			fmt.Fprintf(&builder, "%s execution scope: **%s**.\n\n", backend.Backend, backend.ExecutionScope)
		}
	}

	if len(report.Backends) > 0 {
		builder.WriteString("## Indexing\n\n")
		builder.WriteString("| Backend | Meilisearch | Status | Backend elapsed | Rebuild total | Index setup | Full prepare | Full submit | Full ready | Full total | Tasks | Payload | Engine peak RSS Δ | Client peak RSS Δ | DB used | Filesystem | Update total | Add total | Delete total | Error |\n")
		builder.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
		for _, backend := range report.Backends {
			fmt.Fprintf(
				&builder,
				"| %s | %s | %s | %.1f ms | %.1f ms | %.1f ms | %.1f ms | %.1f ms | %.1f ms | %.1f ms | %d | %s | %s | %s | %s | %s | %.1f ms | %.1f ms | %.1f ms | %s |\n",
				backend.Backend,
				emptyDash(backend.MeilisearchVersion),
				backend.Status,
				backend.ElapsedMS,
				backend.IndexSetupMS+phaseTotalMS(backend.FullIndex),
				backend.IndexSetupMS,
				backend.FullIndex.PrepareMS,
				backend.FullIndex.SubmitMS,
				backend.FullIndex.ReadyMS,
				phaseTotalMS(backend.FullIndex),
				backend.FullIndex.TaskCount,
				formatBytes(backend.FullIndex.SubmittedBytes),
				formatBytes(backend.FullIndex.EnginePeakDelta),
				formatBytes(backend.FullIndex.ClientPeakRSSDelta),
				formatBytes(backend.FullIndex.UsedDiskBytes),
				formatBytes(backend.FullIndex.FilesystemBytes),
				phaseTotalMS(backend.IncrementalIndex),
				phaseTotalMS(backend.AddIndex),
				phaseTotalMS(backend.DeleteIndex),
				escapeCell(backend.Error),
			)
		}
		builder.WriteString("\n")

		geoJSONProjectionRows := 0
		for _, backend := range report.Backends {
			if backend.GeoJSONIndex == nil {
				continue
			}
			if geoJSONProjectionRows == 0 {
				builder.WriteString("### GeoJSON index projection\n\n")
				builder.WriteString("| Meilisearch | Tolerance | Max segment | Mode | Capacity | Arrival | Actual shards | Shard documents | Assignment | Assignment SHA-256 | Source vertices | Pre-normalization vertices | Densified vertices | Antimeridian cuts | Indexed vertices | RDP reduction | Source segments | Pre-normalization segments | Indexed segments |\n")
				builder.WriteString("|---|---:|---:|---|---:|---|---:|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
			}
			projection := backend.GeoJSONIndex
			fmt.Fprintf(&builder, "| %s | %.1f m | %.1f m | %s | %s | %s | %d | %s | %s | %s | %d | %d | %d | %d | %d | %.2f%% | %d | %d | %d |\n", emptyDash(backend.MeilisearchVersion), projection.SimplifyToleranceMeters, projection.MaxSegmentLengthMeters, emptyDash(projection.ShardMode), geoJSONCapacityLabel(projection.MaxTrailsPerShard), emptyDash(projection.CapacityArrival), projection.Shards, escapeCell(fmt.Sprint(projection.ShardDocumentCounts)), escapeCell(projection.ShardAlgorithm), emptyDash(projection.ShardAssignmentSHA256), projection.SourceVertices, projection.PreNormalizationVertices, projection.DensifiedVertices, projection.AntimeridianCuts, projection.PostNormalizationVertices, projection.VertexReductionPercent, projection.SourceSegments, projection.PreNormalizationSegments, projection.PostNormalizationSegments)
			geoJSONProjectionRows++
		}
		if geoJSONProjectionRows > 0 {
			builder.WriteString("\nThe indexed `_geojson` projection may be simplified, then densifies spherical segments and splits antimeridian crossings into RFC 7946-compatible LineStrings. The separately indexed compact polyline and all exact/UX oracles continue to use the original dataset geometry. Wanderer's default trail-list response does not include that polyline.\n\n")
			for _, backend := range report.Backends {
				if backend.GeoJSONIndex != nil && backend.GeoJSONIndex.AssignmentScope != "" {
					fmt.Fprintf(&builder, "Capacity assignment scope: **%s**.\n\n", backend.GeoJSONIndex.AssignmentScope)
					break
				}
			}
		}

		shardRows := 0
		for _, backend := range report.Backends {
			if backend.GeoJSONIndex == nil {
				continue
			}
			for _, shard := range backend.GeoJSONIndex.ShardBuilds {
				if shardRows == 0 {
					builder.WriteString("### GeoJSON shard builds\n\n")
					builder.WriteString("| Meilisearch | Shard | Index | Documents | Indexed vertices | Indexed segments | Completed at | Tasks | Payload | Batch progress |\n")
					builder.WriteString("|---|---:|---|---:|---:|---:|---:|---:|---:|---|\n")
				}
				phase := shard.Phase
				progress := "-"
				if phase.LastBatch != nil {
					if phase.LastBatch.ProgressPercent == 0 && len(phase.LastBatch.ProgressSteps) == 0 && phase.LastBatch.Duration != "" {
						progress = "complete (" + phase.LastBatch.Duration + ")"
					} else {
						progress = fmt.Sprintf("%.1f%% %s", phase.LastBatch.ProgressPercent, strings.Join(phase.LastBatch.ProgressSteps, "; "))
					}
				}
				fmt.Fprintf(&builder, "| %s | %d | %s | %d | %d | %d | %.1f ms | %d | %s | %s |\n", emptyDash(backend.MeilisearchVersion), shard.Shard, escapeCell(shard.IndexUID), phase.DocumentCount, intAt(backend.GeoJSONIndex.ShardIndexedVertices, shard.Shard), intAt(backend.GeoJSONIndex.ShardIndexedSegments, shard.Shard), phase.ReadyMS, phase.TaskCount, formatBytes(phase.SubmittedBytes), escapeCell(progress))
				shardRows++
			}
		}
		if shardRows > 0 {
			builder.WriteString("\nShard tasks are disjoint and, with `max-pending-batches=0`, all are enqueued before the common completion barrier. `Completed at` is an offset from the common phase start, not an isolated shard duration. RSS and storage are measured once for the complete wall-clock phase; they are intentionally not attributed or summed per shard.\n\n")
		}

		batchRows := 0
		for _, backend := range report.Backends {
			if backend.FullIndex.LastBatch == nil {
				continue
			}
			if batchRows == 0 {
				builder.WriteString("### Meilisearch batch diagnostics\n\n")
				builder.WriteString("| Backend | Meilisearch | Batch | Tasks | Strategy | Progress | Steps | Duration | Write blocking |\n")
				builder.WriteString("|---|---|---:|---:|---|---:|---|---:|---:|\n")
			}
			batch := backend.FullIndex.LastBatch
			fmt.Fprintf(&builder, "| %s | %s | %d | %d | %s | %.1f%% | %s | %s | %.1f%% |\n", backend.Backend, emptyDash(backend.MeilisearchVersion), batch.UID, batch.TaskCount, escapeCell(batch.Strategy), batch.ProgressPercent, escapeCell(strings.Join(batch.ProgressSteps, "<br>")), emptyDash(batch.Duration), 100*batch.WriteBlockingRatio)
			batchRows++
		}
		if batchRows > 0 {
			builder.WriteString("\n")
		}

		geoJSONSweeps := 0
		for _, backend := range report.Backends {
			if backend.GeoJSONSweep == nil {
				continue
			}
			if geoJSONSweeps == 0 {
				builder.WriteString("## GeoJSON parameter sweep\n\n")
				builder.WriteString("| Meilisearch | Variant | Selected | Selection evidence | Status | Mode | Resolution | Padding | Max radius expansion | Candidate p95 | Candidates avg/max | Candidate recall | Candidate FN | Boundary-zone misses | Full p95 |\n")
				builder.WriteString("|---|---|---:|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
			}
			geoJSONSweeps++
			for _, variant := range backend.GeoJSONSweep.Variants {
				fullP95, _, _ := geoJSONVariantScore(variant)
				fullP95Cell := "—"
				if variant.Finalist && variant.Error == "" {
					fullP95Cell = fmt.Sprintf("%.2f ms", fullP95)
				}
				selected := ""
				selectionEvidence := ""
				if backend.GeoJSONSweep.SelectedID == variant.Plan.ID {
					selected = "yes"
					selectionEvidence = fmt.Sprintf("%s (%d samples)", backend.GeoJSONSweep.SelectionStatus, backend.GeoJSONSweep.SelectionSamples)
				}
				resolution := "default"
				if variant.Plan.Resolution > 0 {
					resolution = fmt.Sprintf("%d", variant.Plan.Resolution)
				}
				fmt.Fprintf(
					&builder,
					"| %s | %s | %s | %s | %s | %s | %s | %.2f m | %.2f m | %.2f ms | %.1f/%d | %s | %d | %d | %s |\n",
					emptyDash(backend.MeilisearchVersion),
					escapeCell(variant.Plan.ID),
					emptyDash(selected),
					emptyDash(selectionEvidence),
					variant.Status,
					variant.Plan.RadiusMode,
					resolution,
					variant.Plan.ExtraPaddingMeters,
					variant.MaxRadiusExpansionMeters,
					variant.CandidateLatency.P95,
					variant.AverageCandidates,
					variant.MaximumCandidates,
					formatScore(variant.CandidateRecall, variant.ExpectedInterior+variant.ExpectedBoundary > 0),
					variant.CandidateFalseNegatives,
					variant.BoundaryCandidateMisses,
					fullP95Cell,
				)
			}
		}
		if geoJSONSweeps > 0 {
			builder.WriteString("\nThe screen rotates variant order on one shared GeoJSON index. Only conservative variants with zero candidate false negatives and no truncated response become finalists; final selection uses the measured exact/product end-to-end p95. The complete ±boundary tolerance zone is required in the candidate set rather than silently removed from scoring. Fewer than 5,000 full-workload samples makes a selection provisional.\n\n")
		}

		writeGeoJSONDirectSection(&builder, report.Backends)

		builder.WriteString("## Search and accuracy\n\n")
		builder.WriteString("| Backend | Meilisearch | Scenario | Radius | p50 | p95 | DB load p50 | Exact p50 | DB rows avg | Candidates avg | Expected avg | Precision | Recall | Candidate recall |\n")
		builder.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
		for _, backend := range report.Backends {
			for _, query := range backend.Queries {
				fmt.Fprintf(
					&builder,
					"| %s | %s | %s | %.1f km | %.2f ms | %.2f ms | %.2f ms | %.2f ms | %.1f | %.1f | %.1f | %s | %s | %s |\n",
					backend.Backend,
					emptyDash(backend.MeilisearchVersion),
					query.Scenario,
					query.RadiusMeters/1000,
					query.TotalLatency.P50,
					query.TotalLatency.P95,
					query.DatabaseLatency.P50,
					query.ExactLatency.P50,
					query.AverageLoadedRows,
					query.AverageCandidates,
					query.AverageExpected,
					formatScore(query.Precision, query.TruePositives+query.FalsePositives > 0),
					formatScore(query.Recall, query.ExpectedMatches > 0),
					formatScore(query.CandidateRecall, query.ExpectedMatches > 0),
				)
			}
		}
		builder.WriteString("\n")

		builder.WriteString("## Product query pipeline\n\n")
		builder.WriteString("| Backend | Meilisearch | Correct cases | Incorrect warm samples | Clients | p50 | p95 | Throughput | Candidates avg | DB rows avg | DB loaded avg | Final response avg | Exact checks avg | App DB |\n")
		builder.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
		for _, backend := range report.Backends {
			workload := backend.ProductWorkload
			if workload == nil {
				continue
			}
			fmt.Fprintf(
				&builder,
				"| %s | %s | %d/%d | %d | %d | %.2f ms | %.2f ms | %.1f q/s | %.1f | %.1f | %s | %s | %.1f | %s |\n",
				backend.Backend,
				emptyDash(backend.MeilisearchVersion),
				workload.CorrectCases,
				workload.Cases,
				workload.IncorrectWarmSamples,
				workload.Clients,
				workload.TotalLatency.P50,
				workload.TotalLatency.P95,
				workload.ThroughputQPS,
				workload.AverageCandidates,
				workload.AverageLoadedRows,
				formatAverageBytes(workload.AverageEncodedBytes),
				formatAverageBytes(workload.AverageResponseBytes),
				workload.AverageExactChecks,
				formatBytes(workload.DatabaseBytes),
			)
		}
		builder.WriteString("\n")

		var failures int
		for _, backend := range report.Backends {
			if backend.ProductWorkload == nil {
				continue
			}
			for _, testCase := range backend.ProductWorkload.CaseReports {
				if testCase.Correct {
					continue
				}
				if failures == 0 {
					builder.WriteString("### Product correctness failures\n\n")
					builder.WriteString("| Backend | Meilisearch | Case | Query | Radius | Missing candidates | Count got/want | Error |\n")
					builder.WriteString("|---|---|---|---|---:|---:|---:|---|\n")
				}
				failures++
				fmt.Fprintf(&builder, "| %s | %s | %s | %s | %.1f km | %d | %d/%d | %s |\n",
					backend.Backend,
					emptyDash(backend.MeilisearchVersion),
					escapeCell(testCase.Name),
					escapeCell(testCase.QueryID),
					testCase.RadiusMeters/1000,
					testCase.MissingExpectedMatches,
					testCase.ActualTotalItems,
					testCase.ExpectedTotalItems,
					escapeCell(testCase.Error),
				)
			}
		}
		if failures > 0 {
			builder.WriteString("\n")
		}

		performanceReports := 0
		for _, backend := range report.Backends {
			performance := backend.Performance
			if performance == nil {
				continue
			}
			if performanceReports == 0 {
				builder.WriteString("## Performance target\n\n")
				builder.WriteString("| Backend | Meilisearch | Target | Status | Trails | Samples | Clients | Throughput | Full rebuild | Normal p50/p95/p99 | Broad p50/p95/p99 | Correct | Detail |\n")
				builder.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
			}
			performanceReports++
			fmt.Fprintf(&builder, "| %s | %s | %s | %s | %d/%d | %d/%d | %d/%d | %.1f/%.1f q/s | %.1f/%.1f min | %.0f/%.0f/%.0f ms | %.0f/%.0f/%.0f ms | %t | %s |\n",
				backend.Backend,
				emptyDash(backend.MeilisearchVersion),
				performance.Target,
				performance.Status,
				performance.MeasuredTrails,
				performance.RequiredTrails,
				performance.MeasuredSamples,
				performance.MinimumSamples,
				performance.MeasuredClients,
				performance.RequiredClients,
				performance.MeasuredThroughput,
				performance.MinimumThroughput,
				performance.FullRebuildMS/60_000,
				performance.RebuildBudgetMS/60_000,
				performance.NormalWorst.P50,
				performance.NormalWorst.P95,
				performance.NormalWorst.P99,
				performance.BroadWorst.P50,
				performance.BroadWorst.P95,
				performance.BroadWorst.P99,
				performance.CorrectnessPass,
				escapeCell(performance.Reason),
			)
		}
		if performanceReports > 0 {
			builder.WriteString("\nFor `ds923plus`, a 50,000-trail full build must finish within 30 minutes. Normal searches (up to 25 km) must stay within 250/750/1,500 ms p50/p95/p99 at four clients; broader searches use 500/2,000/4,000 ms. Throughput must reach 4 q/s and fewer than 5,000 warm samples cannot pass. This gate is not a substitute for measuring DSM-wide memory, swap, I/O, shadow rebuilds and concurrent mutations on the actual NAS.\n\n")
		}

		directPerformanceReports := 0
		for _, backend := range report.Backends {
			if backend.GeoJSONDirect == nil || backend.GeoJSONDirect.Performance == nil {
				continue
			}
			performance := backend.GeoJSONDirect.Performance
			if directPerformanceReports == 0 {
				builder.WriteString("## GeoJSON direct performance target\n\n")
				builder.WriteString("| Backend | Meilisearch | Plan | Target | Status | Trails | Samples | Clients | Throughput | Full rebuild | Normal p50/p95/p99 | Broad p50/p95/p99 | UX correct | Detail |\n")
				builder.WriteString("|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
			}
			directPerformanceReports++
			fmt.Fprintf(&builder, "| %s | %s | %s | %s | %s | %d/%d | %d/%d | %d/%d | %.1f/%.1f q/s | %.1f/%.1f min | %.0f/%.0f/%.0f ms | %.0f/%.0f/%.0f ms | %t | %s |\n",
				backend.Backend,
				emptyDash(backend.MeilisearchVersion),
				emptyDash(backend.GeoJSONDirect.SelectedID),
				performance.Target,
				performance.Status,
				performance.MeasuredTrails,
				performance.RequiredTrails,
				performance.MeasuredSamples,
				performance.MinimumSamples,
				performance.MeasuredClients,
				performance.RequiredClients,
				performance.MeasuredThroughput,
				performance.MinimumThroughput,
				performance.FullRebuildMS/60_000,
				performance.RebuildBudgetMS/60_000,
				performance.NormalWorst.P50,
				performance.NormalWorst.P95,
				performance.NormalWorst.P99,
				performance.BroadWorst.P50,
				performance.BroadWorst.P95,
				performance.BroadWorst.P99,
				performance.CorrectnessPass,
				escapeCell(performance.Reason),
			)
		}
		if directPerformanceReports > 0 {
			builder.WriteString("\nThe optional DS923+ capacity target uses the same scale, rebuild, concurrency, throughput and radius-latency budgets as the exact product path. Passing or failing it classifies that explicitly selected diagnostic profile; ADR 0002 does not make a particular NAS target a release prerequisite. Its correctness column refers to the explicit GeoJSON UX contract and warm-result stability; it does not claim exact spatial results or exact line-proximity sorting.\n\n")
		}
	}

	builder.WriteString("`point` is Wanderer's current start-point `_geo` baseline. Ordinary engine-query accuracy excludes trails within 0.5 m (or `1e-7 × radius`) of the boundary; the GeoJSON parameter gate instead requires the entire boundary zone as candidates. Selected GeoJSON, H3 and RTree plans load geometry from SQLite and refine exactly. Meilisearch hybrid product paths push ACL, federation, deterministic AND-text and metadata predicates into candidate acquisition, then reapply them after SQLite hydration before exact point-to-polyline distance, exact count, global stable sorting and pagination. RTree applies them in the finalizer only. The synthetic SQLite record encoding and deterministic text contract do not reproduce every production implementation detail; those boundaries remain visible in the report. A capacity-created Direct run provides G1 release evidence when its declared gates pass. Larger NAS profiles remain optional capacity diagnostics. Each Meilisearch backend ran in a fresh container and data directory. A `timed_out` phase is a measured lower bound; later backends still run with a fresh time budget.\n")
	return builder.String()
}

func geoJSONCapacityLabel(capacity int) string {
	if capacity <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d trails", capacity)
}

func writeGeoJSONDirectSection(builder *strings.Builder, backends []BackendReport) {
	if !hasGeoJSONDirectEvidence(backends) {
		return
	}

	builder.WriteString("## GeoJSON direct UX path\n\n")
	builder.WriteString("This section measures the approximate Direct product path. ADR 0002 accepts the r100/S50/5-km/no-padding profile; results for another diagnostic resolution do not alter that decision. The direct product workload treats Meilisearch's filtered, sorted and paginated response as final: it neither hydrates candidate IDs from SQLite nor runs point-to-polyline refinement. Candidate-plus-exact measurements remain separate fallback diagnostics. The neutral boundary band is reported separately; it never hides material misses, material outside hits, false-empty searches, unsupported sorts or truncated responses.\n\n")
	builder.WriteString("The correctness oracle is outside timed samples and independent of Meilisearch: the exhaustive ProductQuery evaluator applies ACL, text, metadata, exact point-to-polyline distance, count, supported sorting and pagination to every synthetic application document. Every eligible resolution is checked with a complete direct ID response plus the real product page. Every completed audit then receives its own complete repeated/concurrent warm workload; accuracy or relative-count-p95 failures retain performance diagnostics but cannot be selected. Those timed requests contain the geo filter and no exact refinement.\n\n")
	builder.WriteString("For historical report compatibility, a plan ID ending in `-exact` means that `_geoRadius` receives the requested radius without conservative padding. It does **not** mean exact route geometry or exact results. `geojson_direct.plan.index_simplification_meters` records the actual indexed projection tolerance even though a direct plan does not add that tolerance to its query radius.\n\n")

	builder.WriteString("### Selection and policy\n\n")
	builder.WriteString("| Meilisearch | Selected plan | Selection status | Resolution | Neutral boundary band | Min recall | Min material precision | Min top-10 recall | Min page recall | Min page precision | Selection rule | Error |\n")
	builder.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|---|\n")
	for _, backend := range backends {
		if !backendHasGeoJSONDirectEvidence(backend) {
			continue
		}
		direct := backend.GeoJSONDirect
		selectedID, selectionStatus, selectionRule, directError := "", "not_selected", "", ""
		resolution := "—"
		if direct != nil {
			selectedID = direct.SelectedID
			selectionStatus = direct.SelectionStatus
			selectionRule = direct.SelectionRule
			directError = direct.Error
			if direct.Plan != nil && direct.Plan.Resolution > 0 {
				resolution = fmt.Sprintf("%d", direct.Plan.Resolution)
			}
		}
		accuracy := geoJSONDirectPolicyAccuracy(backend)
		boundary, minimumRecall, minimumPrecision, minimumTop10, minimumPage, minimumPagePrecision := "—", "—", "—", "—", "—", "—"
		if accuracy != nil {
			boundary = fmt.Sprintf("±%.1f m", accuracy.BoundaryToleranceMeters)
			minimumRecall = formatScore(accuracy.MinimumRecall, true)
			minimumPrecision = formatScore(accuracy.MinimumPrecision, true)
			minimumTop10 = formatScore(accuracy.MinimumTop10Recall, true)
			minimumPage = formatScore(accuracy.MinimumComparablePageRecall, true)
			minimumPagePrecision = formatScore(accuracy.MinimumComparablePagePrecision, true)
		}
		fmt.Fprintf(
			builder,
			"| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			emptyDash(backend.MeilisearchVersion),
			emptyDash(selectedID),
			emptyDash(selectionStatus),
			resolution,
			boundary,
			minimumRecall,
			minimumPrecision,
			minimumTop10,
			minimumPage,
			minimumPagePrecision,
			emptyDash(selectionRule),
			emptyDash(directError),
		)
	}
	builder.WriteString("\nThe minimum recall is an aggregate gate for raw radius recall, clear-interior recall, nearest-result recall and non-empty-request success. Material precision, top-10 coverage and comparable-page recall/precision use their configured aggregate thresholds. Individual material misses, false-empty requests, worst-request recall and worst-page scores remain visible diagnostics. Semantic/ACL false positives and truncation remain unconditional hard failures. Recall over the complete mathematical radius and count error still include the neutral band, so boundary behavior remains visible.\n\n")

	auditTimingRows := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil || (backend.GeoJSONDirect.ProductOracleMS <= 0 && backend.GeoJSONDirect.PlanAuditWallMS <= 0) {
			continue
		}
		if auditTimingRows == 0 {
			builder.WriteString("### Untimed selection overhead\n\n")
			builder.WriteString("| Meilisearch | Exhaustive product oracle | All-plan product audit |\n")
			builder.WriteString("|---|---:|---:|\n")
		}
		auditTimingRows++
		fmt.Fprintf(builder, "| %s | %.1f ms | %.1f ms |\n",
			emptyDash(backend.MeilisearchVersion),
			backend.GeoJSONDirect.ProductOracleMS,
			backend.GeoJSONDirect.PlanAuditWallMS,
		)
	}
	if auditTimingRows > 0 {
		builder.WriteString("\nThese phases choose and verify the plan and are deliberately excluded from warm request latency.\n\n")
	}

	writeGeoJSONDirectProductAudits(builder, backends)
	writeGeoJSONDirectWarmPlans(builder, backends)

	spatialAuditRows := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil || backend.GeoJSONDirect.SpatialAudit == nil {
			continue
		}
		if spatialAuditRows == 0 {
			builder.WriteString("### Capacity-shard direct spatial audit\n\n")
			builder.WriteString("| Meilisearch | Status | Cases | Fanout | HTTP/subqueries | Request/response JSON | p95 | Engine p95 | Recall | Clear recall | Material precision | Material FN/FP | Boundary misses | Top-10 | Nearest | Non-empty success | Truncated | Error |\n")
			builder.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
		}
		spatialAuditRows++
		audit := backend.GeoJSONDirect.SpatialAudit
		accuracy := audit.Accuracy
		fmt.Fprintf(builder, "| %s | %s | %d | %d | %d/%d | %s/%s | %.2f ms | %.2f ms | %s | %s | %s | %d/%d | %d | %s | %s | %s | %d | %s |\n",
			emptyDash(backend.MeilisearchVersion),
			emptyDash(audit.Status),
			audit.Cases,
			audit.ShardFanout,
			audit.HTTPRequests,
			audit.ShardQueries,
			formatBytes(audit.RequestBytes),
			formatBytes(audit.ResponseBytes),
			audit.CandidateLatency.P95,
			audit.EngineProcessing.P95,
			formatScore(accuracy.Recall, accuracy.ExpectedHits > 0),
			formatScore(accuracy.ClearRecall, accuracy.ClearExpectedHits > 0),
			formatScore(accuracy.Precision, accuracy.ClearTruePositives+accuracy.MaterialFalsePositives > 0),
			accuracy.MaterialFalseNegatives,
			accuracy.MaterialFalsePositives,
			accuracy.BoundaryMisses,
			formatScore(accuracy.Top10Recall, accuracy.Top10Expected > 0),
			formatScore(accuracy.NearestRecall, accuracy.NearestExpected > 0),
			formatScore(accuracy.NonEmptyRequestSuccessRate, accuracy.RequestsWithClearExpected > 0),
			accuracy.TruncatedRequests,
			escapeCell(audit.Error),
		)
	}
	if spatialAuditRows > 0 {
		builder.WriteString("\nThis untimed audit executes every dataset query point at 0.5, 5, 25 and 100 km without ACL filtering. It verifies the complete federated ID/count result against the exact point-to-polyline oracle and is excluded from warm product latency and RSS.\n\n")
	}

	builder.WriteString("### Spatial screen accuracy\n\n")
	builder.WriteString("| Meilisearch | Variant | Selected | Status | Requests | Recall | Clear recall | Material precision | Material FN/FP | Boundary FN/allowed FP | Semantic FP | False empty | Non-empty success | Top-10 | Nearest | Worst request | Max miss/hit outside | Truncated |\n")
	builder.WriteString("|---|---|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, backend := range backends {
		if backend.GeoJSONSweep == nil {
			continue
		}
		selectedID := ""
		if backend.GeoJSONDirect != nil {
			selectedID = backend.GeoJSONDirect.SelectedID
		}
		for _, variant := range backend.GeoJSONSweep.Variants {
			accuracy := variant.DirectUX
			if accuracy == nil {
				continue
			}
			selected := ""
			if selectedID != "" && selectedID == variant.Plan.ID {
				selected = "yes"
			}
			fmt.Fprintf(
				builder,
				"| %s | %s | %s | %s | %d | %s | %s | %s | %d/%d | %d/%d | %d | %d | %s | %s | %s | %s | %.2f/%.2f m | %d |\n",
				emptyDash(backend.MeilisearchVersion),
				escapeCell(variant.Plan.ID),
				emptyDash(selected),
				emptyDash(accuracy.Status),
				accuracy.Requests,
				formatScore(accuracy.Recall, accuracy.ExpectedHits > 0),
				formatScore(accuracy.ClearRecall, accuracy.ClearExpectedHits > 0),
				formatScore(accuracy.Precision, accuracy.ClearTruePositives+accuracy.MaterialFalsePositives+accuracy.SemanticFalsePositives > 0),
				accuracy.MaterialFalseNegatives,
				accuracy.MaterialFalsePositives,
				accuracy.BoundaryMisses,
				accuracy.AllowedBoundaryFalsePositives,
				accuracy.SemanticFalsePositives,
				accuracy.FalseEmptyRequests,
				formatScore(accuracy.NonEmptyRequestSuccessRate, accuracy.RequestsWithClearExpected > 0),
				formatScore(accuracy.Top10Recall, accuracy.Top10Expected > 0),
				formatScore(accuracy.NearestRecall, accuracy.NearestExpected > 0),
				formatScore(accuracy.WorstRequestRecall, accuracy.RequestsWithExpected > 0),
				accuracy.MaxInsideMissMeters,
				accuracy.MaxOutsideHitMeters,
				accuracy.TruncatedRequests,
			)
		}
	}
	builder.WriteString("\nTop-10 is coverage of the ten nearest clear-interior reference trails, not a claim about Meilisearch ranking. “Worst request” is the minimum clear-interior recall of one query/radius pair.\n\n")

	builder.WriteString("### Direct count error\n\n")
	builder.WriteString("| Meilisearch | Source | Variant | Returned/expected | Absolute error mean/p95/max | Relative error mean/p95/max |\n")
	builder.WriteString("|---|---|---|---:|---:|---:|\n")
	for _, backend := range backends {
		if backend.GeoJSONSweep != nil {
			for _, variant := range backend.GeoJSONSweep.Variants {
				if variant.DirectUX != nil {
					writeGeoJSONDirectCountRow(builder, backend.MeilisearchVersion, "spatial screen", variant.Plan.ID, *variant.DirectUX)
				}
			}
		}
		if backend.GeoJSONDirect != nil && backend.GeoJSONDirect.ProductWorkload != nil {
			writeGeoJSONDirectCountRow(builder, backend.MeilisearchVersion, "product", backend.GeoJSONDirect.SelectedID, backend.GeoJSONDirect.ProductWorkload.Accuracy)
		}
		if backend.GeoJSONDirect != nil && backend.GeoJSONDirect.SpatialAudit != nil {
			writeGeoJSONDirectCountRow(builder, backend.MeilisearchVersion, "capacity spatial", backend.GeoJSONDirect.SelectedID, backend.GeoJSONDirect.SpatialAudit.Accuracy)
		}
	}
	builder.WriteString("\nCount error compares the Direct result count with the exact geometry reference. Relative error divides by `max(1, expected)` so empty-reference requests stay finite. The delivered `totalHits` and `totalPages` must separately match the completely enumerated Direct set without tolerance; against the geometry oracle, product-count p95 relative error must be at most 1%. Absolute and per-case exact-count values remain diagnostics.\n\n")
	writeGeoJSONDirectProductCountDiagnostics(builder, backends)

	builder.WriteString("### Direct product workload\n\n")
	builder.WriteString("| Meilisearch | Plan | Status | Cases/samples | Clients | Fanout | HTTP/query | Shard queries | Request | Meili response | Product response | p50/p95/p99 | Engine p95 | Throughput | Engine RSS base→peak (Δ) | Client RSS base→peak (Δ) | Unstable warm | Unsupported sorts | Comparable page R/P | Worst page R/P | Top-10 | Nearest | Material FN/FP | False empty | Non-empty success |\n")
	builder.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	productRows := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil || backend.GeoJSONDirect.ProductWorkload == nil {
			continue
		}
		productRows++
		workload := backend.GeoJSONDirect.ProductWorkload
		accuracy := workload.Accuracy
		fmt.Fprintf(
			builder,
			"| %s | %s | %s | %d/%d | %d | %d | %.1f | %.1f | %s | %s | %s | %.2f/%.2f/%.2f ms | %.2f ms | %.1f q/s | %s→%s (%s) | %s→%s (%s) | %d | %d | %s/%s (%d cases) | %s/%s | %s | %s | %d/%d | %d | %s |\n",
			emptyDash(backend.MeilisearchVersion),
			emptyDash(backend.GeoJSONDirect.SelectedID),
			emptyDash(workload.Status),
			workload.Cases,
			workload.Samples,
			workload.Clients,
			workload.ShardFanout,
			workload.AverageHTTPRequests,
			workload.AverageShardQueries,
			formatAverageBytes(workload.AverageRequestBytes),
			formatAverageBytes(workload.AverageWireResponseBytes),
			formatAverageBytes(workload.AverageResponseBytes),
			workload.TotalLatency.P50,
			workload.TotalLatency.P95,
			workload.TotalLatency.P99,
			workload.EngineProcessing.P95,
			workload.ThroughputQPS,
			formatBytes(workload.EngineBaselineBytes),
			formatBytes(workload.EnginePeakBytes),
			formatBytes(workload.EnginePeakDelta),
			formatBytes(workload.ClientBaselineRSSBytes),
			formatBytes(workload.ClientPeakRSSBytes),
			formatBytes(workload.ClientPeakRSSDelta),
			workload.UnstableWarmSamples,
			accuracy.UnsupportedSortCases,
			formatScore(accuracy.ComparablePageRecall, accuracy.ComparablePageExpected > 0),
			formatScore(accuracy.ComparablePagePrecision, accuracy.ComparablePageActual > 0),
			accuracy.ComparablePageCases,
			formatScore(accuracy.WorstComparablePageRecall, accuracy.ComparablePageCases > 0),
			formatScore(accuracy.WorstComparablePagePrecision, accuracy.ComparablePageCases > 0),
			formatScore(accuracy.Top10Recall, accuracy.Top10Expected > 0),
			formatScore(accuracy.NearestRecall, accuracy.NearestExpected > 0),
			accuracy.MaterialFalseNegatives,
			accuracy.MaterialFalsePositives+accuracy.SemanticFalsePositives,
			accuracy.FalseEmptyRequests,
			formatScore(accuracy.NonEmptyRequestSuccessRate, accuracy.RequestsWithClearExpected > 0),
		)
	}
	if productRows == 0 {
		builder.WriteString("| — | — | not measured | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | — |\n")
	}
	builder.WriteString("\nFor capacity shards, one HTTP request contains one subquery per shard and Meilisearch performs the exhaustive global count, sorting and page merge. Request/Meili-response sizes are the actual `/multi-search` JSON bodies (HTTP headers excluded); product response size excludes `_federation` metadata. Exact line-proximity sorting is unsupported on the direct path and is excluded from comparable-page recall instead of being silently repaired.\n\n")

	writeGeoJSONDirectViolations(builder, backends)
	writeGeoJSONDirectProductLimitations(builder, backends)
}

func writeGeoJSONDirectProductAudits(builder *strings.Builder, backends []BackendReport) {
	written := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil {
			continue
		}
		for _, audit := range backend.GeoJSONDirect.ProductAudits {
			if written == 0 {
				builder.WriteString("### Direct product audits\n\n")
				builder.WriteString("| Meilisearch | Plan | Resolution | Status | Cases | Calibrated product-page p95/p99 | Recall | Material precision | Comparable page R/P | Count p95 relative/limit | Error |\n")
				builder.WriteString("|---|---|---:|---|---:|---:|---:|---:|---:|---:|---|\n")
			}
			written++
			resolution := "default"
			if audit.Resolution > 0 {
				resolution = fmt.Sprintf("%d", audit.Resolution)
			}
			latency := "—"
			if audit.ProductPageLatency.P95 > 0 || audit.ProductPageLatency.P99 > 0 {
				latency = fmt.Sprintf("%.2f/%.2f ms", audit.ProductPageLatency.P95, audit.ProductPageLatency.P99)
			}
			accuracy := audit.Accuracy
			countQualification := "—"
			if audit.CountQualification.Cases > 0 {
				countQualification = fmt.Sprintf(
					"%s %.4f/%.4f",
					emptyDash(audit.CountQualification.Status),
					audit.CountQualification.CountError.P95Relative,
					audit.CountQualification.MaximumP95RelativeError,
				)
			}
			fmt.Fprintf(
				builder,
				"| %s | %s | %s | %s | %d | %s | %s | %s | %s/%s | %s | %s |\n",
				emptyDash(backend.MeilisearchVersion),
				emptyDash(audit.PlanID),
				resolution,
				emptyDash(audit.Status),
				audit.Cases,
				latency,
				formatScore(accuracy.Recall, accuracy.ExpectedHits > 0),
				formatScore(accuracy.Precision, accuracy.ClearTruePositives+accuracy.MaterialFalsePositives+accuracy.SemanticFalsePositives > 0),
				formatScore(accuracy.ComparablePageRecall, accuracy.ComparablePageCases > 0),
				formatScore(accuracy.ComparablePagePrecision, accuracy.ComparablePageCases > 0),
				countQualification,
				emptyDash(audit.Error),
			)
		}
	}
	if written > 0 {
		builder.WriteString("\nEach row combines the untimed exact product audit with the product-page calibration requests for that plan. An audit passes only when both the aggregate spatial UX contract and the relative count-p95 gate pass. Calibration orders diagnostics and breaks ties; final selection uses each audit passer's complete concurrent warm workload and, when configured, the active performance target below.\n\n")
	}
}

func writeGeoJSONDirectWarmPlans(builder *strings.Builder, backends []BackendReport) {
	written := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil {
			continue
		}
		for _, audit := range backend.GeoJSONDirect.ProductAudits {
			if audit.WarmWorkload == nil && audit.WarmError == "" && audit.Performance == nil {
				continue
			}
			if written == 0 {
				builder.WriteString("### Direct warm qualification by plan\n\n")
				builder.WriteString("| Meilisearch | Plan | Selected | Warm status | Samples | Clients | Warm p50/p95/p99 | Throughput | Unstable | Target | Error |\n")
				builder.WriteString("|---|---|---:|---|---:|---:|---:|---:|---:|---|---|\n")
			}
			written++
			selected := ""
			if backend.GeoJSONDirect.SelectedID == audit.PlanID {
				selected = "yes"
			}
			warmStatus, samples, clients, latency, throughput, unstable := "not_run", "—", "—", "—", "—", "—"
			if workload := audit.WarmWorkload; workload != nil {
				warmStatus = workload.Status
				samples = fmt.Sprintf("%d/%d", workload.Samples, workload.ExpectedSamples)
				clients = fmt.Sprintf("%d", workload.Clients)
				latency = fmt.Sprintf("%.2f/%.2f/%.2f ms", workload.TotalLatency.P50, workload.TotalLatency.P95, workload.TotalLatency.P99)
				throughput = fmt.Sprintf("%.1f q/s", workload.ThroughputQPS)
				unstable = fmt.Sprintf("%d", workload.UnstableWarmSamples)
			}
			target := "—"
			errorText := audit.WarmError
			if performance := audit.Performance; performance != nil {
				target = performance.Target + ": " + performance.Status
				if performance.Reason != "" {
					if errorText != "" {
						errorText += "; "
					}
					errorText += performance.Reason
				}
			}
			fmt.Fprintf(builder, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				emptyDash(backend.MeilisearchVersion),
				emptyDash(audit.PlanID),
				emptyDash(selected),
				emptyDash(warmStatus),
				samples,
				clients,
				latency,
				throughput,
				unstable,
				target,
				emptyDash(errorText),
			)
		}
	}
	if written > 0 {
		builder.WriteString("\nEvery completed product audit receives its own repeated workload, including spatial-accuracy and count-qualification failures so their latency, throughput and RSS remain measurable. Only plans that pass both the aggregate spatial UX contract and the relative count-p95 gate can be selected; warm stability and, when active, the performance target are additional gates.\n\n")
	}
}

func hasGeoJSONDirectEvidence(backends []BackendReport) bool {
	for _, backend := range backends {
		if backendHasGeoJSONDirectEvidence(backend) {
			return true
		}
	}
	return false
}

func backendHasGeoJSONDirectEvidence(backend BackendReport) bool {
	if backend.GeoJSONDirect != nil {
		return true
	}
	if backend.GeoJSONSweep != nil {
		for _, variant := range backend.GeoJSONSweep.Variants {
			if variant.DirectUX != nil {
				return true
			}
		}
	}
	return false
}

func geoJSONDirectPolicyAccuracy(backend BackendReport) *GeoJSONDirectAccuracy {
	if backend.GeoJSONDirect != nil {
		if backend.GeoJSONDirect.Accuracy != nil {
			return backend.GeoJSONDirect.Accuracy
		}
		if backend.GeoJSONDirect.ProductWorkload != nil {
			return &backend.GeoJSONDirect.ProductWorkload.Accuracy
		}
	}
	if backend.GeoJSONSweep != nil {
		for _, variant := range backend.GeoJSONSweep.Variants {
			if variant.DirectUX != nil {
				return variant.DirectUX
			}
		}
	}
	return nil
}

func writeGeoJSONDirectCountRow(builder *strings.Builder, version, source, variant string, accuracy GeoJSONDirectAccuracy) {
	error := accuracy.CountError
	fmt.Fprintf(
		builder,
		"| %s | %s | %s | %d/%d | %.2f/%.2f/%d | %.4f/%.4f/%.4f |\n",
		emptyDash(version),
		escapeCell(source),
		emptyDash(variant),
		accuracy.ReturnedHits,
		accuracy.ExpectedHits,
		error.MeanAbsolute,
		error.P95Absolute,
		error.MaxAbsolute,
		error.MeanRelative,
		error.P95Relative,
		error.MaxRelative,
	)
}

func writeGeoJSONDirectProductCountDiagnostics(builder *strings.Builder, backends []BackendReport) {
	written := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil || backend.GeoJSONDirect.ProductWorkload == nil {
			continue
		}
		workload := backend.GeoJSONDirect.ProductWorkload
		counts := workload.CountDiagnostics
		if counts.Cases == 0 {
			continue
		}
		if written == 0 {
			builder.WriteString("### Direct product count diagnostics\n\n")
			builder.WriteString("| Meilisearch | Plan | Status | Cases | Exact count | Within 1% | totalPages match | Absolute error mean/p95/max | Relative error mean/p95/max/limit |\n")
			builder.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|\n")
		}
		written++
		error := counts.CountError
		fmt.Fprintf(
			builder,
			"| %s | %s | %s | %d | %d/%d (%.4f) | %d/%d (%.4f) | %d/%d (%.4f) | %.2f/%.2f/%d | %.4f/%.4f/%.4f/%.4f |\n",
			emptyDash(backend.MeilisearchVersion),
			emptyDash(backend.GeoJSONDirect.SelectedID),
			emptyDash(counts.Status),
			counts.Cases,
			counts.ExactCountCases,
			counts.Cases,
			counts.ExactCountRate,
			counts.WithinOnePercentCases,
			counts.Cases,
			counts.WithinOnePercentRate,
			counts.TotalPagesMatchCases,
			counts.Cases,
			counts.TotalPagesMatchRate,
			error.MeanAbsolute,
			error.P95Absolute,
			error.MaxAbsolute,
			error.MeanRelative,
			error.P95Relative,
			error.MaxRelative,
			counts.MaximumP95RelativeError,
		)
	}
	if written == 0 {
		return
	}

	builder.WriteString("\n| Meilisearch | Plan | Case | Query | Radius | Count expected/returned/delta | FN total/boundary/material | FP total/boundary/material/semantic | totalPages expected/actual |\n")
	builder.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|\n")
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil || backend.GeoJSONDirect.ProductWorkload == nil {
			continue
		}
		for _, testCase := range backend.GeoJSONDirect.ProductWorkload.CaseReports {
			fmt.Fprintf(
				builder,
				"| %s | %s | %s | %s | %.1f km | %d/%d/%+d | %d/%d/%d | %d/%d/%d/%d | %d/%d |\n",
				emptyDash(backend.MeilisearchVersion),
				emptyDash(backend.GeoJSONDirect.SelectedID),
				escapeCell(testCase.Name),
				escapeCell(testCase.QueryID),
				testCase.RadiusMeters/1000,
				testCase.ExpectedTotalItems,
				testCase.ActualTotalItems,
				testCase.SignedCountDelta,
				testCase.FalseNegatives,
				testCase.BoundaryFalseNegatives,
				testCase.MaterialFalseNegatives,
				testCase.FalsePositives,
				testCase.BoundaryFalsePositives,
				testCase.MaterialFalsePositives,
				testCase.SemanticFalsePositives,
				testCase.ExpectedTotalPages,
				testCase.ActualTotalPages,
			)
		}
	}
	builder.WriteString("\nThe signed delta is `returned - expected`. Relative error divides absolute error by `max(1, expected)`. The relative p95 value is the normative 1% count gate; exact-count rate, absolute error, per-case deltas and boundary/material classifications remain diagnostics.\n\n")
}

func writeGeoJSONDirectViolations(builder *strings.Builder, backends []BackendReport) {
	written := 0
	writeViolation := func(version, source string, violation GeoJSONDirectCaseViolation) {
		if written == 0 {
			builder.WriteString("### Direct UX violations\n\n")
			builder.WriteString("| Meilisearch | Variant/source | Query | Scenario | Radius | Returned/expected | FN/material | FP/material | Recall | Max miss/hit outside | Worst material trails | False empty | Error |\n")
			builder.WriteString("|---|---|---|---|---:|---:|---:|---:|---:|---:|---|---:|---|\n")
		}
		written++
		fmt.Fprintf(
			builder,
			"| %s | %s | %s | %s | %.1f km | %d/%d | %d/%d | %d/%d | %s | %.2f m / %.2f m | %s | %t | %s |\n",
			emptyDash(version),
			escapeCell(source),
			escapeCell(violation.QueryID),
			escapeCell(violation.Scenario),
			violation.RadiusMeters/1000,
			violation.Returned,
			violation.Expected,
			violation.FalseNegatives,
			violation.MaterialFalseNegative,
			violation.FalsePositives,
			violation.MaterialFalsePositive,
			formatScore(violation.Recall, violation.Expected > 0),
			violation.MaxInsideMissMeters,
			violation.MaxOutsideHitMeters,
			formatGeoJSONDirectTrailViolationDetails(violation),
			violation.FalseEmpty,
			emptyDash(violation.Error),
		)
	}
	for _, backend := range backends {
		if backend.GeoJSONSweep != nil {
			for _, variant := range backend.GeoJSONSweep.Variants {
				if variant.DirectUX == nil {
					continue
				}
				for _, violation := range variant.DirectUX.Violations {
					writeViolation(backend.MeilisearchVersion, variant.Plan.ID+" spatial", violation)
				}
			}
		}
		if backend.GeoJSONDirect != nil && backend.GeoJSONDirect.SpatialAudit != nil {
			for _, violation := range backend.GeoJSONDirect.SpatialAudit.Accuracy.Violations {
				writeViolation(backend.MeilisearchVersion, backend.GeoJSONDirect.SelectedID+" capacity spatial", violation)
			}
		}
		if backend.GeoJSONDirect != nil && backend.GeoJSONDirect.ProductWorkload != nil {
			for _, violation := range backend.GeoJSONDirect.ProductWorkload.Accuracy.Violations {
				writeViolation(backend.MeilisearchVersion, backend.GeoJSONDirect.SelectedID+" product", violation)
			}
		}
	}
	if written > 0 {
		builder.WriteString("\nOnly the bounded violation samples retained in `report.json` are expanded here. Aggregate counters above remain authoritative.\n\n")
	}
}

func formatGeoJSONDirectTrailViolationDetails(violation GeoJSONDirectCaseViolation) string {
	parts := make([]string, 0, len(violation.MaterialFalseNegativeTrails)+len(violation.MaterialFalsePositiveTrails))
	for _, detail := range violation.MaterialFalseNegativeTrails {
		parts = append(parts, fmt.Sprintf(
			"FN %s @ %.1f m (inside %.1f m)",
			detail.TrailID,
			detail.DistanceMeters,
			detail.RadiusDeltaMeters,
		))
	}
	for _, detail := range violation.MaterialFalsePositiveTrails {
		parts = append(parts, fmt.Sprintf(
			"FP %s @ %.1f m (outside %.1f m)",
			detail.TrailID,
			detail.DistanceMeters,
			detail.RadiusDeltaMeters,
		))
	}
	if len(parts) == 0 {
		return "—"
	}
	return escapeCell(strings.Join(parts, "; "))
}

func writeGeoJSONDirectProductLimitations(builder *strings.Builder, backends []BackendReport) {
	written := 0
	for _, backend := range backends {
		if backend.GeoJSONDirect == nil || backend.GeoJSONDirect.ProductWorkload == nil {
			continue
		}
		workload := backend.GeoJSONDirect.ProductWorkload
		for _, testCase := range workload.CaseReports {
			if testCase.SortSupported && testCase.PageComparable && testCase.CorrectSemantics &&
				testCase.MaterialFalseNegatives == 0 && !testCase.FalseEmpty && testCase.Error == "" &&
				testCase.PageRecall >= workload.Accuracy.MinimumComparablePageRecall &&
				testCase.PagePrecision >= workload.Accuracy.MinimumComparablePagePrecision {
				continue
			}
			if written == 0 {
				builder.WriteString("### Direct product limitations and mismatches\n\n")
				builder.WriteString("| Meilisearch | Plan | Case | Query | Sort | Sort supported | Page comparable | Page R/P | Count got/want | Material FN | False empty | Non-spatial semantics | Error |\n")
				builder.WriteString("|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|\n")
			}
			written++
			fmt.Fprintf(
				builder,
				"| %s | %s | %s | %s | %s | %t | %t | %s/%s | %d/%d | %d | %t | %t | %s |\n",
				emptyDash(backend.MeilisearchVersion),
				emptyDash(backend.GeoJSONDirect.SelectedID),
				escapeCell(testCase.Name),
				escapeCell(testCase.QueryID),
				escapeCell(testCase.Sort),
				testCase.SortSupported,
				testCase.PageComparable,
				formatScore(testCase.PageRecall, testCase.PageComparable),
				formatScore(testCase.PagePrecision, testCase.PageComparable),
				testCase.ActualTotalItems,
				testCase.ExpectedTotalItems,
				testCase.MaterialFalseNegatives,
				testCase.FalseEmpty,
				testCase.CorrectSemantics,
				emptyDash(testCase.Error),
			)
		}
	}
	if written > 0 {
		builder.WriteString("\nUnsupported sorts are limitations, not failed comparable pages. In particular, line `proximity` would require the exact geometry calculation that this direct path is intended to avoid.\n\n")
	}
}

func formatAverageBytes(bytes float64) string {
	if bytes <= 0 {
		return "—"
	}
	return formatBytes(int64(bytes + 0.5))
}

func formatBytes(bytes int64) string {
	if bytes <= 0 {
		return "—"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func phaseTotalMS(phase PhaseReport) float64 {
	return phase.PrepareMS + phase.ReadyMS
}

func intAt(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func formatScore(value float64, defined bool) string {
	if !defined {
		return "—"
	}
	return fmt.Sprintf("%.4f", value)
}

func formatContractIDs(ids []string) string {
	if len(ids) == 0 {
		return "∅"
	}
	return escapeCell(strings.Join(ids, ", "))
}

func emptyDash(value string) string {
	if value == "" {
		return "—"
	}
	return escapeCell(value)
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	if value == "" {
		return "—"
	}
	return value
}
