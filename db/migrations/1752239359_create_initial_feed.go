package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/util"
)

func init() {
	m.Register(func(app core.App) error {
		trails, err := app.FindAllRecords("trails")
		if err != nil {
			return err
		}

		for _, t := range trails {
			trailAuthor, err := app.FindRecordById("activitypub_actors", t.GetString("author"))
			if err != nil {
				return err
			}

			feed, err := util.InsertIntoFeed(app, trailAuthor.Id, trailAuthor.Id, t.Id, util.TrailFeed)
			if err != nil {
				return err
			}

			feed.SetRaw("created", t.GetDateTime("created"))
			err = app.Save(feed)
			if err != nil {
				return err
			}

			followers, err := app.FindRecordsByFilter("follows", "followee={:author}", "", -1, 0, dbx.Params{"author": trailAuthor.Id})
			if err != nil {
				return err
			}
			for _, f := range followers {
				feed, err = util.InsertIntoFeed(app, f.GetString("follower"), trailAuthor.Id, t.Id, util.TrailFeed)
				if err != nil {
					return err
				}

				feed.SetRaw("created", t.GetDateTime("created"))
				err = app.Save(feed)
				if err != nil {
					return err
				}
			}
		}

		lists, err := app.FindAllRecords("lists")
		if err != nil {
			return err
		}

		for _, t := range lists {
			listAuthor, err := app.FindRecordById("activitypub_actors", t.GetString("author"))
			if err != nil {
				return err
			}

			feed, err := util.InsertIntoFeed(app, listAuthor.Id, listAuthor.Id, t.Id, util.ListFeed)
			if err != nil {
				return err
			}

			feed.SetRaw("created", t.GetDateTime("created"))
			err = app.Save(feed)
			if err != nil {
				return err
			}

			followers, err := app.FindRecordsByFilter("follows", "followee={:author}", "", -1, 0, dbx.Params{"author": listAuthor.Id})
			if err != nil {
				return err
			}
			for _, f := range followers {
				feed, err = util.InsertIntoFeed(app, f.GetString("follower"), listAuthor.Id, t.Id, util.ListFeed)
				if err != nil {
					return err
				}

				feed.SetRaw("created", t.GetDateTime("created"))
				err = app.Save(feed)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}, func(app core.App) error {
		_, err := app.DB().
			NewQuery("DELETE FROM feed;").
			Execute()

		return err
	})
}
