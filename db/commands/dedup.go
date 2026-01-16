package commands

import (
	"crypto/sha1"
	"fmt"
	"log"
	"sort"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

func Dedup(app *pocketbase.PocketBase) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "dedup",
		Short: "Deduplicate trails by all matching fields",
		Run: func(cmd *cobra.Command, args []string) {
			records, err := app.FindAllRecords("trails")
			if err != nil {
				log.Fatalf("failed to fetch trails: %v", err)
			}

			// group by composite key
			trailsByKey := make(map[string][]*core.Record)
			for _, r := range records {
				key := makeKey(r)
				trailsByKey[key] = append(trailsByKey[key], r)
			}

			var duplicates []*core.Record
			for _, recs := range trailsByKey {
				if len(recs) <= 1 {
					continue
				}

				// sort by created date ascending
				sort.Slice(recs, func(i, j int) bool {
					return recs[i].GetDateTime("created").Time().Before(recs[j].GetDateTime("created").Time())
				})

				original := recs[0]
				dupes := recs[1:]

				// print header row for original
				// print original as header
				fmt.Printf("\nOriginal: id=%s, name=%s, distance=%.2f, elevation_gain=%.2f, elevation_loss=%.2f, lat=%.5f, lon=%.5f, duration=%.2f, location=%s, category=%s, author=%s, created=%s\n",
					original.Id,
					original.GetString("name"),
					original.GetFloat("distance"),
					original.GetFloat("elevation_gain"),
					original.GetFloat("elevation_loss"),
					original.GetFloat("lat"),
					original.GetFloat("lon"),
					original.GetFloat("duration"),
					original.GetString("location"),
					original.GetString("category"),
					original.GetString("author"),
					original.GetDateTime("created"),
				)

				// print duplicates indented
				for _, d := range dupes {
					fmt.Printf("  Duplicate: id=%s, name=%s, distance=%.2f, elevation_gain=%.2f, elevation_loss=%.2f, lat=%.5f, lon=%.5f, duration=%.2f, location=%s, category=%s, author=%s, created=%s\n",
						d.Id,
						d.GetString("name"),
						d.GetFloat("distance"),
						d.GetFloat("elevation_gain"),
						d.GetFloat("elevation_loss"),
						d.GetFloat("lat"),
						d.GetFloat("lon"),
						d.GetFloat("duration"),
						d.GetString("location"),
						d.GetString("category"),
						d.GetString("author"),
						d.GetDateTime("created"),
					)
					duplicates = append(duplicates, d)
				}
			}

			if dryRun {
				fmt.Printf("\n[Dry Run] Found %d duplicates (no deletions performed)\n", len(duplicates))
				return
			}

			// delete duplicates
			for _, d := range duplicates {
				if err := app.Delete(d); err != nil {
					fmt.Printf("Failed to delete duplicate %s: %v\n", d.Id, err)
				} else {
					fmt.Printf("Deleted duplicate %s\n", d.Id)
				}
			}
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show duplicates without deleting them")

	return cmd
}

// makeKey creates a composite key string for duplicate detection
func makeKey(r *core.Record) string {
	data := fmt.Sprintf("%s|%f|%f|%f|%f|%f|%f|%s|%s|%s",
		r.GetString("name"),
		r.GetFloat("distance"),
		r.GetFloat("elevation_gain"),
		r.GetFloat("elevation_loss"),
		r.GetFloat("lat"),
		r.GetFloat("lon"),
		r.GetFloat("duration"),
		r.GetString("location"),
		r.GetString("category"),
		r.GetString("author"),
	)
	h := sha1.Sum([]byte(data))
	return fmt.Sprintf("%x", h)
}
