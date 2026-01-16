package util

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
)

func ActorFromUser(app core.App, u *core.Record) (*core.Record, error) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		return nil, err
	}
	priv, pub, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	privBytes := x509.MarshalPKCS1PrivateKey(priv)

	privEncrypted, err := security.Encrypt(privBytes, encryptionKey)
	if err != nil {
		return nil, err
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	settings, err := app.FindFirstRecordByData("settings", "user", u.Id)
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)

	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return nil, fmt.Errorf("ORIGIN environment variable not set")
	}
	id := fmt.Sprintf("%s/api/v1/activitypub/user/%s", origin, strings.ToLower(u.GetString("username")))

	url, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	domain := strings.TrimPrefix(url.Hostname(), "www.")

	record.Set("username", u.GetString("username"))
	record.Set("preferred_username", strings.ToLower(u.GetString("username")))
	record.Set("domain", domain)
	record.Set("summary", settings.GetString("bio"))
	record.Set("published", u.GetDateTime("created"))
	record.Set("iri", id)
	if u.GetString("avatar") != "" {
		record.Set("icon", fmt.Sprintf("%s/api/v1/files/users/%s/%s", origin, u.Id, u.GetString("avatar")))
	}
	record.Set("inbox", id+"/inbox")
	record.Set("outbox", id+"/outbox")
	record.Set("followers", id+"/followers")
	record.Set("following", id+"/following")
	record.Set("isLocal", true)
	record.Set("public_key", string(pubPem))
	record.Set("private_key", privEncrypted)
	record.Set("user", u.Id)
	record.Set("last_fetched", time.Now())

	err = app.Save(record)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func generateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	pub := &priv.PublicKey
	return priv, pub, nil
}

func SyncOutbox(app core.App, actor *core.Record) error {
	return fetchOutboxPage(app, actor, actor.GetString("outbox")+"?page=1")
}

func fetchOutboxPage(app core.App, actor *core.Record, pageURL string) error {
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Accept", `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var page pub.OrderedCollectionPage
	err = json.Unmarshal(body, &page)
	if err != nil {
		return err
	}

	for _, item := range page.OrderedItems {
		activity, err := pub.ToActivity(item)
		if err != nil {
			return err
		}
		if activity.Type != pub.CreateType {
			continue
		}
	}

	if page.Next != nil {
		return fetchOutboxPage(app, actor, page.Next.GetID().String())
	}

	return nil
}

func TrailFromActivity(activity pub.Activity, app core.App, actor *core.Record) (*core.Record, error) {
	t, err := pub.ToObject(activity.Object)
	if err != nil {
		return nil, err
	}

	iri := t.ID.String()
	var record *core.Record
	if actor.GetBool(("isLocal")) {
		trailUrl, parseErr := url.Parse(iri)
		if parseErr != nil {
			return nil, parseErr
		}
		trailId := path.Base(trailUrl.Path)
		record, err = app.FindRecordById("trails", trailId)
	} else {
		record, err = app.FindFirstRecordByData("trails", "iri", iri)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			collection, err := app.FindCollectionByNameOrId("trails")
			if err != nil {
				return nil, err
			}

			record = core.NewRecord(collection)
			record.Set("id", security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet))

		} else {
			return nil, err
		}
	} else {
		// this trail exists already
		// nothing more to do

		return record, nil
	}

	var distance, duration, elevation_gain, elevation_loss float64
	var diffculty, category string
	trailTags := []string{}
	tags, err := pub.ToItemCollection(t.Tag)
	if err != nil {
		return nil, err
	}

	for _, tag := range tags.Collection() {
		tagObj, err := pub.ToObject(tag)
		if err != nil {
			continue
		}
		content := tagObj.Content.First().Value.String()
		switch tagObj.Name.First().Value.String() {
		case "category":
			category = content
		case "difficulty":
			diffculty = content
		case "elevation_gain":
			elevation_gain, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "elevation_loss":
			elevation_loss, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "duration":
			duration, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "distance":
			distance, err = strconv.ParseFloat(content[:len(content)-1], 64)
		case "tag":
			existingTag, err := app.FindFirstRecordByData("tags", "name", content)
			if err != nil {
				if err == sql.ErrNoRows {
					collection, err := app.FindCollectionByNameOrId("tags")
					if err != nil {
						continue
					}
					existingTag = core.NewRecord(collection)
					existingTag.Set("name", content)
					err = app.Save(existingTag)
					if err != nil {
						continue
					}
				} else {
					continue
				}
			}

			trailTags = append(trailTags, existingTag.Id)
		}
		if err != nil {
			continue
		}
	}

	record.Set("name", t.Name.First().Value)
	record.Set("description", t.Content.First().Value)
	record.Set("location", t.Location.(*pub.Place).Name.First().Value)
	record.Set("lat", t.Location.(*pub.Place).Latitude)
	record.Set("lon", t.Location.(*pub.Place).Longitude)
	record.Set("distance", distance)
	record.Set("elevation_gain", elevation_gain)
	record.Set("elevation_loss", elevation_loss)
	record.Set("duration", duration)
	record.Set("difficulty", diffculty)
	record.Set("date", t.StartTime.Unix())
	record.Set("tags", trailTags)
	record.Set("public", true)
	record.Set("iri", t.ID.String())
	record.Set("author", actor.Id)

	categoryRecord, err := app.FindFirstRecordByData("categories", "name", category)
	if err == nil {
		record.Set("category", categoryRecord.Id)
	}

	if t.Attachment != nil {

		attachments, err := pub.ToItemCollection(t.Attachment)
		if err != nil {
			return nil, err
		}

		photoURLs := []string{}
		gpxURL := ""
		for _, a := range attachments.Collection() {
			attachment, err := pub.ToObject(a)
			if err != nil {
				continue
			}
			if attachment.Type == pub.DocumentType && attachment.MediaType == "application/xml+gpx" {
				gpxURL = attachment.URL.GetLink().String()
			} else if attachment.Type == pub.ImageType {
				photoURLs = append(photoURLs, attachment.URL.GetLink().String())
			}
		}

		if len(photoURLs) > 0 {
			photos := []*filesystem.File{}
			for i, purl := range photoURLs {
				photo, err := filesystem.NewFileFromURL(context.Background(), purl)
				if err != nil {
					continue
				}
				photos[i] = photo
			}

			record.Set("photos", photos)
		}

		if gpxURL != "" {
			gpx, err := filesystem.NewFileFromURL(context.Background(), gpxURL)
			if err != nil {
				return nil, err
			}

			record.Set("gpx", gpx)
		}
	}

	return record, app.Save(record)
}

func ObjectFromTrail(app core.App, trail *core.Record, mentions *pub.ItemCollection) (*pub.Object, error) {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return nil, fmt.Errorf("ORIGIN not set")
	}

	trailAuthor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return nil, err
	}
	errs := app.ExpandRecord(trail, []string{"tags"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand tags: %v", errs)
	}
	errs = app.ExpandRecord(trail, []string{"category"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand category: %v", errs)
	}

	category := ""
	categoryRecord := trail.ExpandedOne("category")
	if categoryRecord != nil {
		category = categoryRecord.GetString("name")
	}

	tagRecords := trail.ExpandedAll("tags")

	tags := pub.ItemCollection{
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "category")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, category)),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "difficulty")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, trail.GetString("difficulty"))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "elevation_gain")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", trail.GetFloat("elevation_gain")))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "elevation_loss")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", trail.GetFloat("elevation_loss")))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "distance")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", trail.GetFloat("distance")))),
		},
		pub.Object{
			Type:    pub.NoteType,
			Name:    pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "duration")),
			Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, fmt.Sprintf("%fm", trail.GetFloat("duration")))),
		},
	}

	if mentions != nil {
		for _, m := range *mentions {
			tags.Append(m)
		}
	}

	for _, v := range tagRecords {
		hashtag := pub.ObjectNew(pub.NoteType)
		hashtag.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "tag"))
		hashtag.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, v.GetString("name")))

		tags.Append(hashtag)
	}

	photos := trail.GetStringSlice("photos")

	gpx := ""
	if trail.GetString("gpx") != "" {
		gpx = fmt.Sprintf("%s/api/v1/files/trails/%s/%s", origin, trail.Id, trail.GetString("gpx"))
	}

	attachments := make(pub.ItemCollection, max(len(photos), 2))
	for i := range min(len(photos), 3) {
		iri := fmt.Sprintf("%s/api/v1/files/trails/%s/%s", origin, trail.Id, photos[i])

		attachments[i] = pub.Image{
			Type:      pub.ImageType,
			MediaType: "image/jpeg",
			URL:       pub.IRI(iri),
		}
	}
	if gpx != "" {
		attachments.Append(pub.Document{
			Type:      pub.DocumentType,
			MediaType: "application/xml+gpx",
			URL:       pub.IRI(gpx),
		})
	}

	activityURL := fmt.Sprintf("%s/trail/view/@%s/%s", origin, trailAuthor.GetString("preferred_username"), trail.Id)
	activityContent := fmt.Sprintf("<h1>%s</h1>%s<p><a href=\"%s\">%s</a></p>", trail.GetString("name"), trail.GetString("description"), activityURL, activityURL)

	trailObject := pub.ObjectNew(pub.NoteType)

	trailObject.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, trail.GetString("name")))
	trailObject.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, activityContent))
	trailObject.Location = pub.Place{
		Type:      pub.PlaceType,
		Name:      pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, trail.GetString("location"))),
		Latitude:  trail.GetFloat("lat"),
		Longitude: trail.GetFloat("lon"),
	}
	trailObject.AttributedTo = pub.IRI(trailAuthor.GetString("iri"))
	trailObject.Published = trail.GetDateTime("created").Time()
	trailObject.ID = pub.IRI(fmt.Sprintf("%s/api/v1/trail/%s", origin, trail.Id))
	trailObject.URL = pub.IRI(activityURL)

	trailObject.StartTime = trail.GetDateTime("date").Time()
	trailObject.Attachment = attachments

	trailObject.Tag = tags
	return trailObject, nil
}

func ListFromActivity(activity pub.Activity, app core.App, actor *core.Record) (*core.Record, error) {
	l, err := pub.ToObject(activity.Object)
	if err != nil {
		return nil, err
	}

	iri := l.ID.String()
	var record *core.Record
	if actor.GetBool(("isLocal")) {
		listURL, parseErr := url.Parse(iri)
		if parseErr != nil {
			return nil, parseErr
		}
		listId := path.Base(listURL.Path)
		record, err = app.FindRecordById("lists", listId)
	} else {
		record, err = app.FindFirstRecordByData("lists", "iri", iri)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			collection, err := app.FindCollectionByNameOrId("lists")
			if err != nil {
				return nil, err
			}

			record = core.NewRecord(collection)
			record.Set("id", security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet))
		} else {
			return nil, err
		}
	} else {
		// this list exists already
		// nothing more to do

		return record, nil
	}

	record.Set("name", l.Name.First().Value)
	record.Set("description", l.Content.First().Value)
	record.Set("public", true)
	record.Set("iri", iri)
	record.Set("author", actor.Id)

	if l.Attachment != nil {

		avatarURL := ""
		attachments, err := pub.ToItemCollection(l.Attachment)
		if err != nil {
			return nil, err
		}

		for _, a := range attachments.Collection() {
			attachment, err := pub.ToObject(a)
			if err != nil {
				continue
			}
			if attachment.Type == pub.ImageType {
				avatarURL = attachment.URL.GetLink().String()
			}
		}

		if avatarURL != "" {
			avatar, err := filesystem.NewFileFromURL(context.Background(), avatarURL)

			if err != nil {
				return nil, err
			}

			record.Set("avatar", avatar)
		}
	}

	err = app.Save(record)

	return record, err
}

func ObjectFromList(app core.App, list *core.Record) (*pub.Object, error) {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return nil, fmt.Errorf("ORIGIN not set")
	}

	listAuthor, err := app.FindRecordById("activitypub_actors", list.GetString("author"))
	if err != nil {
		return nil, err
	}
	avatar := ""
	if list.GetString("avatar") != "" {
		avatar = fmt.Sprintf("%s/api/v1/files/lists/%s/%s", origin, list.Id, list.GetString("avatar"))
	}

	attachments := make(pub.ItemCollection, 2)
	if avatar != "" {
		attachments[0] = pub.Image{
			Type:      pub.ImageType,
			MediaType: "image/jpeg",
			URL:       pub.IRI(avatar),
		}
	}

	activityURL := fmt.Sprintf("%s/lists/@%s/%s", origin, listAuthor.GetString("preferred_username"), list.Id)
	activityContent := fmt.Sprintf("%s<p><a href=\"%s\">%s</a></p>", list.GetString("description"), activityURL, activityURL)

	listObject := pub.ObjectNew(pub.NoteType)
	listObject.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, list.GetString("name")))
	listObject.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, activityContent))

	listObject.AttributedTo = pub.IRI(listAuthor.GetString("iri"))
	listObject.Published = list.GetDateTime("created").Time()
	listObject.ID = pub.IRI(fmt.Sprintf("%s/api/v1/list/%s", origin, list.Id))
	listObject.URL = pub.IRI(activityURL)
	listObject.Attachment = attachments
	return listObject, nil
}

func ObjectFromComment(app core.App, comment *core.Record, mentions *pub.ItemCollection) (*pub.Object, error) {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return nil, fmt.Errorf("ORIGIN not set")
	}

	commentAuthor, err := app.FindRecordById("activitypub_actors", comment.GetString("author"))
	if err != nil {
		return nil, err
	}

	commentTrail, err := app.FindRecordById("trails", comment.GetString("trail"))
	if err != nil {
		return nil, err
	}
	commentTrailAuthor, err := app.FindRecordById("activitypub_actors", commentTrail.GetString("author"))
	if err != nil {
		return nil, err
	}

	trailURL := ""
	if commentTrailAuthor.GetBool("isLocal") {
		trailURL = fmt.Sprintf("https://%s/api/v1/trail/%s", commentTrailAuthor.GetString("domain"), comment.GetString("trail"))
	} else {
		trailURL = commentTrail.GetString("iri")
	}

	commentObject := pub.ObjectNew(pub.NoteType)
	commentObject.ID = pub.IRI(fmt.Sprintf("%s/api/v1/comment/%s", origin, comment.Id))
	commentObject.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, comment.GetString("text")))
	commentObject.Published = comment.GetDateTime("created").Time()
	commentObject.AttributedTo = pub.IRI(commentAuthor.GetString("iri"))
	commentObject.InReplyTo = pub.IRI(trailURL)

	if mentions != nil {
		commentObject.Tag = *mentions
	}

	return commentObject, nil
}

func TrailObjectFromIRI(iri string) (*pub.Object, error) {
	fetchURL := strings.Replace(iri, "api/v1/trail", "api/v1/activitypub/trail", 1)

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var object pub.Object
	err = json.Unmarshal(body, &object)
	if err != nil {
		return nil, err
	}

	return &object, nil
}
