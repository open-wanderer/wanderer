package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// deleteMeiliDocuments measures a batched delete phase with the same task
// barrier, RSS sampling, and post-phase storage statistics as document
// indexing. Waiting for the final task is sufficient as Meilisearch executes
// index tasks FIFO; earlier tasks are audited once after the barrier.
func deleteMeiliDocuments(
	ctx context.Context,
	client meilisearch.ServiceManager,
	container *meiliContainer,
	ids []string,
	batchSize int,
) (PhaseReport, error) {
	report := PhaseReport{}
	if len(ids) == 0 {
		return report, nil
	}
	if batchSize < 1 {
		batchSize = 100
	}

	for index, id := range ids {
		if id == "" {
			return report, fmt.Errorf("delete Meilisearch document at index %d: ID is empty", index)
		}
	}

	monitor := startContainerRSSMonitor(container)
	clientMonitor := startRSSMonitor(os.Getpid())
	monitorsFinished := false
	finishMonitors := func() {
		if monitorsFinished {
			return
		}
		monitorsFinished = true
		report.EngineBaselineBytes, report.EnginePeakBytes, report.EnginePeakDelta = monitor.finish()
		report.ClientBaselineRSSBytes, report.ClientPeakRSSBytes, report.ClientPeakRSSDelta = clientMonitor.finish()
	}
	fail := func(err error, started time.Time) (PhaseReport, error) {
		report.ReadyMS = milliseconds(time.Since(started))
		finishMonitors()
		captureCtx, captureCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = populateMeiliPhaseStats(captureCtx, client, container, &report)
		captureCancel()
		return report, err
	}

	readyStarted := time.Now()
	tasks := make([]*meilisearch.TaskInfo, 0, (len(ids)+batchSize-1)/batchSize)
	for start := 0; start < len(ids); start += batchSize {
		if err := ctx.Err(); err != nil {
			report.SubmitMS = milliseconds(time.Since(readyStarted))
			return fail(err, readyStarted)
		}
		end := min(start+batchSize, len(ids))
		taskInfo, err := client.Index(benchmarkIndexUID).DeleteDocumentsWithContext(ctx, ids[start:end], nil)
		if err != nil {
			report.SubmitMS = milliseconds(time.Since(readyStarted))
			return fail(err, readyStarted)
		}
		if taskInfo == nil {
			report.SubmitMS = milliseconds(time.Since(readyStarted))
			return fail(fmt.Errorf("Meilisearch returned no delete task"), readyStarted)
		}
		tasks = append(tasks, taskInfo)
	}
	report.SubmitMS = milliseconds(time.Since(readyStarted))

	if _, err := waitForSuccessfulTask(ctx, client, tasks[len(tasks)-1]); err != nil {
		return fail(err, readyStarted)
	}
	report.ReadyMS = milliseconds(time.Since(readyStarted))
	finishMonitors()

	for _, taskInfo := range tasks[:len(tasks)-1] {
		task, err := client.GetTaskWithContext(ctx, taskInfo.TaskUID)
		if err != nil {
			return report, fmt.Errorf("audit delete task %d: %w", taskInfo.TaskUID, err)
		}
		if err := successfulTaskError(task); err != nil {
			return report, err
		}
	}

	if err := populateMeiliPhaseStats(ctx, client, container, &report); err != nil {
		return report, err
	}
	return report, nil
}
