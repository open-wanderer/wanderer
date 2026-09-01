package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"time"
)

func runRTreeBenchmark(
	ctx context.Context,
	workRoot string,
	dataset Dataset,
	updates []Trail,
	config RunConfig,
	oracle *accuracyOracle,
) BackendReport {
	report := BackendReport{Backend: "rtree", Status: "failed"}
	path := filepath.Join(workRoot, "rtree", "geobench.sqlite")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		report.Error = err.Error()
		return report
	}
	backend, err := newRTreeBackend(path)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	defer backend.close()

	prepareMonitor := startRSSMonitor(os.Getpid())
	started := time.Now()
	prepared, err := prepareRTreeTrailsContext(ctx, dataset.Trails)
	report.FullIndex.PrepareMS = milliseconds(time.Since(started))
	report.FullIndex.ClientBaselineRSSBytes, report.FullIndex.ClientPeakRSSBytes, report.FullIndex.ClientPeakRSSDelta = prepareMonitor.finish()
	if err != nil {
		report.Error = fmt.Sprintf("prepare full index: %v", err)
		return report
	}

	monitor := startRSSMonitor(os.Getpid())
	started = time.Now()
	err = backend.writeTrails(ctx, prepared, true)
	report.FullIndex.ReadyMS = milliseconds(time.Since(started))
	report.FullIndex.SubmitMS = report.FullIndex.ReadyMS
	writeBaseline, writePeak, _ := monitor.finish()
	mergeClientRSS(&report.FullIndex, writeBaseline, writePeak)
	if err != nil {
		report.Error = fmt.Sprintf("full index: %v", err)
		return report
	}
	if err := populateRTreePhaseStats(ctx, backend, &report.FullIndex); err != nil {
		report.Error = fmt.Sprintf("full index stats: %v", err)
		return report
	}
	if config.IndexOnly {
		report.Status = "ok"
		return report
	}

	runEngineQueries := config.QueryWorkload != "product"
	runProductQueries := config.QueryWorkload != "engine"
	var productDatabase *productStore
	var productDocuments []ProductQueryDocument
	if runProductQueries {
		applicationStarted := time.Now()
		productDocuments = generateProductDocuments(dataset)
		productDatabase, err = newProductStore(
			filepath.Join(workRoot, "rtree", "application.sqlite"),
			config.QueryClients,
		)
		if err == nil {
			err = productDatabase.replaceAll(ctx, productDocuments)
		}
		report.ApplicationDatabaseSetupMS = milliseconds(time.Since(applicationStarted))
		if err != nil {
			if productDatabase != nil {
				_ = productDatabase.close()
			}
			report.Error = fmt.Sprintf("prepare application database: %v", err)
			return report
		}
		defer productDatabase.close()
		report.ApplicationDatabaseBytes, err = productDatabase.diskBytes()
		if err != nil {
			report.Error = fmt.Sprintf("measure application database: %v", err)
			return report
		}
	}

	queryFunction := func(ctx context.Context, queryPoint QueryPoint, radius float64) (CandidateResult, error) {
		started := time.Now()
		ids, err := backend.queryCandidates(ctx, queryPoint.Point, radius)
		duration := time.Since(started)
		return CandidateResult{
			IDs:                ids,
			WallDuration:       duration,
			EngineProcessingMS: milliseconds(duration),
		}, err
	}
	if runEngineQueries {
		report.Queries, err = benchmarkQueries(
			ctx,
			dataset,
			config.RadiiMeters,
			config.Repetitions,
			backend.loadCandidateTrails,
			oracle,
			queryFunction,
		)
		if err != nil {
			report.Error = fmt.Sprintf("engine queries: %v", err)
			return report
		}
	}

	if runProductQueries {
		backend.db.SetMaxOpenConns(max(1, config.QueryClients))
		report.ProductWorkload, err = benchmarkProductWorkload(
			ctx,
			dataset,
			config.RadiiMeters,
			config.Repetitions,
			config.QueryClients,
			productDatabase,
			productDocuments,
			queryFunction,
		)
		backend.db.SetMaxOpenConns(1)
		if report.ProductWorkload != nil {
			report.ProductWorkload.SetupMS = report.ApplicationDatabaseSetupMS
			report.ProductWorkload.DatabaseBytes = report.ApplicationDatabaseBytes
		}
		if err != nil {
			report.Error = fmt.Sprintf("product queries: %v", err)
			return report
		}
	}

	prepared = nil
	debug.FreeOSMemory()
	prepareMonitor = startRSSMonitor(os.Getpid())
	started = time.Now()
	prepared, err = prepareRTreeTrailsContext(ctx, updates)
	report.IncrementalIndex.PrepareMS = milliseconds(time.Since(started))
	report.IncrementalIndex.ClientBaselineRSSBytes, report.IncrementalIndex.ClientPeakRSSBytes, report.IncrementalIndex.ClientPeakRSSDelta = prepareMonitor.finish()
	if err != nil {
		report.Error = fmt.Sprintf("prepare incremental index: %v", err)
		return report
	}

	monitor = startRSSMonitor(os.Getpid())
	started = time.Now()
	err = backend.writeTrails(ctx, prepared, false)
	report.IncrementalIndex.ReadyMS = milliseconds(time.Since(started))
	report.IncrementalIndex.SubmitMS = report.IncrementalIndex.ReadyMS
	writeBaseline, writePeak, _ = monitor.finish()
	mergeClientRSS(&report.IncrementalIndex, writeBaseline, writePeak)
	if err != nil {
		report.Error = fmt.Sprintf("incremental index: %v", err)
		return report
	}
	if err := populateRTreePhaseStats(ctx, backend, &report.IncrementalIndex); err != nil {
		report.Error = fmt.Sprintf("incremental index stats: %v", err)
		return report
	}
	if len(updates) > 0 {
		for _, index := range verificationIndices(len(updates)) {
			stored, err := backend.storedTrail(ctx, updates[index].ID)
			if err != nil {
				report.Error = fmt.Sprintf("verify incremental index: %v", err)
				return report
			}
			if !reflect.DeepEqual(stored, updates[index]) {
				report.Error = fmt.Sprintf("verify incremental index: trail %q differs after update", updates[index].ID)
				return report
			}
			ids, err := backend.queryCandidates(ctx, updates[index].Start(), 1)
			if err != nil {
				report.Error = fmt.Sprintf("verify incremental spatial index for trail %q: %v", updates[index].ID, err)
				return report
			}
			if _, found := ids[updates[index].ID]; !found {
				report.Error = fmt.Sprintf("verify incremental spatial index: updated trail %q was not found at its new start", updates[index].ID)
				return report
			}
		}
	}

	additions := deriveAddedTrails(dataset, config.IncrementalCount)
	debug.FreeOSMemory()
	prepareMonitor = startRSSMonitor(os.Getpid())
	started = time.Now()
	preparedAdditions, err := prepareRTreeTrailsContext(ctx, additions)
	report.AddIndex.PrepareMS = milliseconds(time.Since(started))
	report.AddIndex.ClientBaselineRSSBytes, report.AddIndex.ClientPeakRSSBytes, report.AddIndex.ClientPeakRSSDelta = prepareMonitor.finish()
	if err != nil {
		report.Error = fmt.Sprintf("prepare additions: %v", err)
		return report
	}
	monitor = startRSSMonitor(os.Getpid())
	started = time.Now()
	err = backend.writeTrails(ctx, preparedAdditions, false)
	report.AddIndex.ReadyMS = milliseconds(time.Since(started))
	report.AddIndex.SubmitMS = report.AddIndex.ReadyMS
	writeBaseline, writePeak, _ = monitor.finish()
	mergeClientRSS(&report.AddIndex, writeBaseline, writePeak)
	if err != nil {
		report.Error = fmt.Sprintf("add index: %v", err)
		return report
	}
	if err := populateRTreePhaseStats(ctx, backend, &report.AddIndex); err != nil {
		report.Error = fmt.Sprintf("add index stats: %v", err)
		return report
	}
	for _, index := range verificationIndices(len(additions)) {
		stored, err := backend.storedTrail(ctx, additions[index].ID)
		if err != nil || !reflect.DeepEqual(stored, additions[index]) {
			report.Error = fmt.Sprintf("verify addition for trail %q: stored geometry differs: %v", additions[index].ID, err)
			return report
		}
		ids, err := backend.queryCandidates(ctx, additions[index].Start(), 1)
		if err != nil {
			report.Error = fmt.Sprintf("verify added spatial index for trail %q: %v", additions[index].ID, err)
			return report
		}
		if _, found := ids[additions[index].ID]; !found {
			report.Error = fmt.Sprintf("verify added spatial index: trail %q was not found at its start", additions[index].ID)
			return report
		}
	}

	started = time.Now()
	deletionIDs := selectDeletionIDs(additions, len(additions))
	report.DeleteIndex.PrepareMS = milliseconds(time.Since(started))
	monitor = startRSSMonitor(os.Getpid())
	started = time.Now()
	err = backend.deleteTrails(ctx, deletionIDs)
	report.DeleteIndex.ReadyMS = milliseconds(time.Since(started))
	report.DeleteIndex.SubmitMS = report.DeleteIndex.ReadyMS
	writeBaseline, writePeak, _ = monitor.finish()
	mergeClientRSS(&report.DeleteIndex, writeBaseline, writePeak)
	if err != nil {
		report.Error = fmt.Sprintf("delete index: %v", err)
		return report
	}
	if err := populateRTreePhaseStats(ctx, backend, &report.DeleteIndex); err != nil {
		report.Error = fmt.Sprintf("delete index stats: %v", err)
		return report
	}
	if report.DeleteIndex.DocumentCount != int64(len(dataset.Trails)) {
		report.Error = fmt.Sprintf("verify deletion: index contains %d documents, want %d", report.DeleteIndex.DocumentCount, len(dataset.Trails))
		return report
	}
	for _, index := range verificationIndices(len(additions)) {
		if _, err := backend.storedTrail(ctx, additions[index].ID); err == nil {
			report.Error = fmt.Sprintf("verify deletion: trail %q still exists", additions[index].ID)
			return report
		}
		ids, err := backend.queryCandidates(ctx, additions[index].Start(), 1)
		if err != nil {
			report.Error = fmt.Sprintf("verify deleted spatial index for trail %q: %v", additions[index].ID, err)
			return report
		}
		if _, found := ids[additions[index].ID]; found {
			report.Error = fmt.Sprintf("verify deleted spatial index: trail %q is still a candidate", additions[index].ID)
			return report
		}
	}

	report.Status = completedBackendStatus(report)
	report.Performance = evaluatePerformanceTarget(config, report)
	return report
}

func populateRTreePhaseStats(ctx context.Context, backend *rtreeBackend, report *PhaseReport) error {
	diskBytes, err := backend.diskBytes()
	if err != nil {
		return err
	}
	usedBytes, err := backend.usedDatabaseBytes(ctx)
	if err != nil {
		return err
	}
	documents, err := backend.documentCount(ctx)
	if err != nil {
		return err
	}
	report.DiskBytes = diskBytes
	report.UsedDiskBytes = usedBytes
	report.FilesystemBytes = diskBytes
	report.DocumentCount = documents
	return nil
}
