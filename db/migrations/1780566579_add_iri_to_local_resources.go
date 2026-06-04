package migrations

import (
	"fmt"
	"net/url"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {

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
			author, err := app.FindRecordById("activitypub_actors", c.GetString("author"))
			if err != nil {
				return err
			}
			authorIri, _ := url.Parse(author.GetString("iri"))
			authorDomain := author.GetString("domain")
			authorScheme := authorIri.Scheme
			iri = fmt.Sprintf("%s://%s/api/v1/comment/%s", authorScheme, authorDomain, c.Id)
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
			author, err := app.FindRecordById("activitypub_actors", l.GetString("author"))
			if err != nil {
				return err
			}
			authorIri, _ := url.Parse(author.GetString("iri"))
			authorDomain := author.GetString("domain")
			authorScheme := authorIri.Scheme
			iri = fmt.Sprintf("%s://%s/api/v1/list/%s", authorScheme, authorDomain, l.Id)
			l.Set("iri", iri)

			if err := app.Save(l); err != nil {
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
			author, err := app.FindRecordById("activitypub_actors", sl.GetString("author"))
			if err != nil {
				return err
			}
			authorIri, _ := url.Parse(author.GetString("iri"))
			authorDomain := author.GetString("domain")
			authorScheme := authorIri.Scheme
			iri = fmt.Sprintf("%s://%s/api/v1/summit-log/%s", authorScheme, authorDomain, sl.Id)
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
			author, err := app.FindRecordById("activitypub_actors", t.GetString("author"))
			if err != nil {
				return err
			}
			authorIri, _ := url.Parse(author.GetString("iri"))
			authorDomain := author.GetString("domain")
			authorScheme := authorIri.Scheme
			iri = fmt.Sprintf("%s://%s/api/v1/trail/%s", authorScheme, authorDomain, t.Id)
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
			author, err := app.FindRecordById("activitypub_actors", wp.GetString("author"))
			if err != nil {
				return err
			}
			authorIri, _ := url.Parse(author.GetString("iri"))
			authorDomain := author.GetString("domain")
			authorScheme := authorIri.Scheme
			iri = fmt.Sprintf("%s://%s/api/v1/waypoint/%s", authorScheme, authorDomain, wp.Id)
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
