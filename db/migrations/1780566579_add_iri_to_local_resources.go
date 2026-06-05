package migrations

import (
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			return fmt.Errorf("ORIGIN not set")
		}

		// Comments
		comments, err := app.FindAllRecords("comments")
		if err != nil {
			return err
		}

		for _, c := range comments {
			iri := c.GetString("iri")
			if iri != "" {
				continue
			}
			iri = fmt.Sprintf("%s/api/v1/comment/%s", origin, c.Id)
			c.Set("iri", iri)

			if err := app.Save(c); err != nil {
				return err
			}
		}
		// ---

		// Lists
		lists, err := app.FindAllRecords("lists")
		if err != nil {
			return err
		}

		for _, l := range lists {
			iri := l.GetString("iri")
			if iri != "" {
				continue
			}

			iri = fmt.Sprintf("%s/api/v1/list/%s", origin, l.Id)
			l.Set("iri", iri)

			if err := app.UnsafeWithoutHooks().Save(l); err != nil {
				return err
			}
		}
		// ---

		// Summit Logs
		summitLogs, err := app.FindAllRecords("summit_logs")
		if err != nil {
			return err
		}

		for _, sl := range summitLogs {
			iri := sl.GetString("iri")
			if iri != "" {
				continue
			}
			iri = fmt.Sprintf("%s/api/v1/summit-log/%s", origin, sl.Id)
			sl.Set("iri", iri)

			if err := app.Save(sl); err != nil {
				return err
			}
		}
		// ---

		// Trails
		trails, err := app.FindAllRecords("trails")
		if err != nil {
			return err
		}

		for _, t := range trails {
			iri := t.GetString("iri")
			if iri != "" {
				continue
			}

			iri = fmt.Sprintf("%s/api/v1/trail/%s", origin, t.Id)
			t.Set("iri", iri)

			if err := app.UnsafeWithoutHooks().Save(t); err != nil {
				return err
			}
		}
		// ---

		// Waypoints

		wps, err := app.FindAllRecords("waypoints")
		if err != nil {
			return err
		}

		for _, wp := range wps {
			iri := wp.GetString("iri")
			if iri != "" {
				continue
			}

			iri = fmt.Sprintf("%s/api/v1/waypoint/%s", origin, wp.Id)
			wp.Set("iri", iri)

			if err := app.Save(wp); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {

		return nil
	})
}
