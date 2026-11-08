package util

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func stringChunks(values []string, size int) [][]string {
	if size <= 0 || len(values) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func recordsByIDs(app core.App, collection string, ids []string) ([]*core.Record, error) {
	return recordsByFieldValues(app, collection, "id", ids, "", nil)
}

func recordsByFieldValues(app core.App, collection string, field string, values []string, extraFilter string, extraParams dbx.Params) ([]*core.Record, error) {
	values = UniqueNonEmptyStrings(values)
	if len(values) == 0 {
		return []*core.Record{}, nil
	}

	// Leave room for extra conditions within PocketBase's 200-expression limit.
	// Ordering and limits belong to the caller, after all chunks are collected.
	records := []*core.Record{}
	seen := map[string]bool{}
	for _, chunk := range stringChunks(values, 150) {
		params := dbx.Params{}
		for key, value := range extraParams {
			params[key] = value
		}
		filters := make([]string, 0, len(chunk))
		for i, value := range chunk {
			key := fmt.Sprintf("value_%d", i)
			params[key] = value
			filters = append(filters, field+"={:"+key+"}")
		}

		filter := "(" + strings.Join(filters, " || ") + ")"
		if strings.TrimSpace(extraFilter) != "" {
			filter += " && (" + extraFilter + ")"
		}
		chunkRecords, err := app.FindRecordsByFilter(collection, filter, "", -1, 0, params)
		if err != nil {
			return nil, err
		}
		for _, record := range chunkRecords {
			if !seen[record.Id] {
				seen[record.Id] = true
				records = append(records, record)
			}
		}
	}
	return records, nil
}

func recordFieldValues(records []*core.Record, field string) []string {
	values := make([]string, 0, len(records))
	for _, record := range records {
		values = append(values, record.GetString(field))
	}
	return values
}

func recordIDs(records []*core.Record) []string {
	values := make([]string, 0, len(records))
	for _, record := range records {
		values = append(values, record.Id)
	}
	return values
}
