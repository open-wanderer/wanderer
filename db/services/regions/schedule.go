package regions

import (
	"fmt"
	"os"
	"time"

	"github.com/pocketbase/pocketbase/tools/cron"
)

// CronSchedule resolves the region-archive-build cron expression the same
// way main.go's registerCronJobs does (REGION_ARCHIVE_CRON_SCHEDULE env
// var, default "0 3 * * *" UTC). Exported so both the actual cron
// registration and the admin "Sync now" status endpoint (which needs to
// report the next scheduled run) resolve the identical value from one
// place rather than duplicating the env var name/default.
func CronSchedule() string {
	if v := os.Getenv("REGION_ARCHIVE_CRON_SCHEDULE"); v != "" {
		return v
	}
	return "0 3 * * *"
}

// NextScheduledSync returns the next UTC time the given cron expression
// will fire. PocketBase's cron package (app.Cron(), never retimezoned in
// this project — see main.go) runs in UTC on a 1-minute tick and exposes no
// direct "next run" helper, only Schedule.IsDue for a single instant, so
// this searches forward minute-by-minute from the next full minute (never
// "now" itself, so a currently-due minute is never reported as still
// pending). Bounded to a year of minutes so a pathological expression
// can't spin forever — in practice every valid 5-field cron expression
// resolves within, at most, about a month.
func NextScheduledSync(cronExpr string) (time.Time, error) {
	schedule, err := cron.NewSchedule(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	t := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
	const maxMinutes = 366 * 24 * 60
	for i := 0; i < maxMinutes; i++ {
		if schedule.IsDue(cron.NewMoment(t)) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no matching time found for cron expression %q within a year", cronExpr)
}
