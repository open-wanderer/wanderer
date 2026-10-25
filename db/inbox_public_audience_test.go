package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSignedInboxPreservesPublicAudienceForms(t *testing.T) {
	f := newTrailMentionInboxFixture(t)
	author := f.actor(t, "replyauthor", false)
	recipient := f.actor(t, "replyrecipient", true)
	trails, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("iri", "https://example.com/api/v1/trail/publicparent")
	trail.Set("author", recipient.Id)
	trail.Set("name", "Public parent")
	trail.Set("public", true)
	if err := f.app.Save(trail); err != nil {
		t.Fatal(err)
	}
	comments, err := f.app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatal(err)
	}
	sequence := 0
	for _, kind := range []string{"Create", "Update"} {
		for _, location := range []string{"object", "activity"} {
			for _, field := range []string{"to", "cc"} {
				for _, audience := range []string{"https://www.w3.org/ns/activitystreams#Public", "as:Public", "Public", "https://remote.example/followers"} {
					t.Run(kind+"/"+location+"/"+field+"/"+audience, func(t *testing.T) {
						sequence++
						iri := fmt.Sprintf("https://remote.example/api/v1/comment/reply%d", sequence)
						if kind == "Update" {
							comment := core.NewRecord(comments)
							comment.Set("iri", iri)
							comment.Set("author", author.Id)
							comment.Set("trail", trail.Id)
							comment.Set("text", "Previous reply")
							if err := f.app.Save(comment); err != nil {
								t.Fatal(err)
							}
						}
						object := map[string]any{"type": "Note", "id": iri, "attributedTo": author.GetString("iri"), "inReplyTo": trail.GetString("iri"), "content": "Current reply"}
						activity := map[string]any{"@context": "https://www.w3.org/ns/activitystreams", "type": kind, "id": iri + "/activity", "actor": author.GetString("iri"), "object": object}
						target := object
						if location == "activity" {
							target = activity
						}
						target[field] = []string{recipient.GetString("iri"), audience}
						body, err := json.Marshal(activity)
						if err != nil {
							t.Fatal(err)
						}
						result := f.process(author, recipient, body)
						if result.err != nil || result.status != http.StatusOK {
							t.Fatalf("signed inbox: status=%d error=%v", result.status, result.err)
						}
						comment, err := f.app.FindFirstRecordByData("comments", "iri", iri)
						if audience != "https://remote.example/followers" {
							if err != nil || comment.GetString("text") != "Current reply" {
								t.Fatalf("public reply missing: record=%v error=%v", comment, err)
							}
						} else if kind == "Create" {
							if err == nil {
								t.Fatal("directly addressed reply imported")
							}
						} else if err != nil || comment.GetString("text") != "Previous reply" {
							t.Fatalf("direct update changed public reply: record=%v error=%v", comment, err)
						}
					})
				}
			}
		}
	}
}
