package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type inboxActionCase struct {
	object                 pub.Item
	collection, actorField string
	targetLookup           func(string) bool
}

func newInboxActionCase(t *testing.T, w *knownKeyWorld, kind pub.ActivityVocabularyType) inboxActionCase {
	t.Helper()
	w.save(t, "settings", map[string]any{"user": w.recipient.GetString("user"), "notifications": map[string]any{
		"new_follower": map[string]any{"web": true, "email": false},
		"trail_like":   map[string]any{"web": true, "email": false},
	}})
	if kind == pub.FollowType {
		iri := w.recipient.GetString("iri")
		return inboxActionCase{pub.IRI(iri), "follows", "follower", func(query string) bool {
			return strings.HasPrefix(query, "SELECT") && strings.Contains(query, "FROM `activitypub_actors` WHERE `iri`="+iri+" ")
		}}
	}
	iri := "https://example.com/api/v1/trail/morningwalk0001"
	w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
	return inboxActionCase{pub.IRI(iri), "trail_like", "actor", func(query string) bool {
		return strings.HasPrefix(query, "SELECT") && strings.Contains(query, "FROM `trails` WHERE `iri`='"+iri+"' ")
	}}
}

func waitInboxStage(t *testing.T, stage <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-stage:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not start", name)
	}
}

func assertInboxAction(t *testing.T, w *knownKeyWorld, c inboxActionCase, actor *core.Record, count int, id string) {
	t.Helper()
	records, err := w.app.FindAllRecords(c.collection, dbx.HashExp{c.actorField: actor.Id})
	if err != nil || len(records) != count {
		t.Fatalf("%s=%d, want %d: %v", c.collection, len(records), count, err)
	}
	if id != "" && records[0].Id != id {
		t.Fatalf("later action's record changed: %s, want %s", records[0].Id, id)
	}
}

func assertInboxFollowAccepted(t *testing.T, w *knownKeyWorld, activity pub.Activity) {
	t.Helper()
	accepts, err := w.app.FindAllRecords("activitypub_activities", dbx.HashExp{"type": "Accept"})
	if err != nil {
		t.Fatal(err)
	}
	for _, accept := range accepts {
		object, err := json.Marshal(accept.Get("object"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(object), `"id":"`+activity.ID.String()+`"`) {
			return
		}
	}
	t.Fatalf("no recorded Accept for %s", activity.ID)
}

func TestInboxEarlierOriginalKeepsRepeatedAction(t *testing.T) {
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
		t.Run(string(kind), func(t *testing.T) {
			w := newKnownKeyWorld(t)
			handler := knownKeyInboxRouter(t, w)
			actor := w.sender(t, "sender", "remote.example", true)
			c := newInboxActionCase(t, w, kind)
			first := pub.Activity{ID: "https://remote.example/activities/first", Type: kind, Object: c.object}
			undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &first}
			repeated := pub.Activity{ID: "https://remote.example/activities/repeated", Type: kind, Object: c.object}
			entered, resume := make(chan struct{}), make(chan struct{})
			var held atomic.Bool
			var release sync.Once
			defer release.Do(func() { close(resume) })
			hook := w.app.OnRecordCreate(c.collection).BindFunc(func(event *core.RecordEvent) error {
				if held.CompareAndSwap(false, true) {
					close(entered)
					<-resume
				}
				return event.Next()
			})
			defer w.app.OnRecordCreate(c.collection).Unbind(hook)
			firstDone := make(chan orderedInboxResult, 1)
			go func() {
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, first)
				firstDone <- orderedInboxResult{status, err}
			}()
			waitInboxStage(t, entered, "first insert")
			for _, activity := range []pub.Activity{undo, repeated} {
				if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, activity); status != http.StatusOK || err != nil {
					t.Fatalf("%s: %d, %v", activity.Type, status, err)
				}
			}
			release.Do(func() { close(resume) })
			orderedInboxResultOK(t, firstDone)
			assertInboxAction(t, w, c, actor, 1, "")
			if kind == pub.FollowType {
				assertInboxFollowAccepted(t, w, repeated)
				notifications, err := w.app.FindAllRecords("notifications", dbx.HashExp{"type": "new_follower", "author": actor.Id})
				if err != nil || len(notifications) != 1 {
					t.Fatalf("follow notifications=%d, want 1: %v", len(notifications), err)
				}
			}
		})
	}
}

func TestInboxUndoKeepsActionCommittedDuringLookup(t *testing.T) {
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
		for _, pass := range []int32{1, 2} {
			t.Run(fmt.Sprintf("%s/pass=%d", kind, pass), func(t *testing.T) {
				w := newKnownKeyWorld(t)
				handler := knownKeyInboxRouter(t, w)
				actor := w.sender(t, "sender", "remote.example", true)
				c := newInboxActionCase(t, w, kind)
				// Finish the sender's key refresh before holding a target lookup.
				warmup := pub.Activity{ID: "https://remote.example/activities/warmup", Type: pub.LikeType, Object: pub.IRI("https://example.com/api/v1/trail/missing00000001")}
				if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, warmup); status != http.StatusOK || err != nil {
					t.Fatalf("warm-up: %d, %v", status, err)
				}
				first := pub.Activity{ID: "https://remote.example/activities/first", Type: kind, Object: c.object}
				undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &first}
				next := pub.Activity{ID: "https://remote.example/activities/next", Type: kind, Object: c.object}
				entered, resume := make(chan struct{}), make(chan struct{})
				var release sync.Once
				defer release.Do(func() { close(resume) })
				var armed atomic.Bool
				var calls atomic.Int32
				armed.Store(true)
				db := w.app.ConcurrentDB().(*dbx.DB)
				previous := db.QueryLogFunc
				db.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
					if previous != nil {
						previous(ctx, elapsed, query, rows, err)
					}
					if armed.Load() && c.targetLookup(query) && calls.Add(1) == pass {
						close(entered)
						<-resume
					}
				}
				defer func() { db.QueryLogFunc = previous }()
				undone := make(chan orderedInboxResult, 1)
				go func() {
					status, err := w.deliverOrderedActivity(handler, actor, w.recipient, undo)
					undone <- orderedInboxResult{status, err}
				}()
				waitInboxStage(t, entered, "undo lookup")
				armed.Store(false)
				if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, next); status != http.StatusOK || err != nil {
					t.Fatalf("next action: %d, %v", status, err)
				}
				release.Do(func() { close(resume) })
				orderedInboxResultOK(t, undone)
				assertInboxAction(t, w, c, actor, 1, "")
			})
		}
	}
}

// Wait until the removal reaches its write: an OnRecordDelete hook for a check
// before the write, or the writer wait of the removal's transaction.
func waitInboxRemoval(t *testing.T, db *sql.DB, waits int64, hook <-chan struct{}) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-hook:
			return
		case <-tick.C:
			if db.Stats().WaitCount > waits {
				return
			}
		case <-deadline.C:
			t.Fatal("removal did not reach the held writer")
		}
	}
}

func TestInboxUndoChecksArrivalAfterWriterWait(t *testing.T) {
	for _, replay := range []bool{false, true} {
		for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
			t.Run(fmt.Sprintf("%s/replay=%t", kind, replay), func(t *testing.T) {
				w := newKnownKeyWorld(t)
				handler := knownKeyInboxRouter(t, w)
				actor := w.sender(t, "sender", "remote.example", true)
				c := newInboxActionCase(t, w, kind)
				first := pub.Activity{ID: "https://remote.example/activities/first", Type: kind, Object: c.object}
				undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &first}
				next := pub.Activity{ID: "https://remote.example/activities/next", Type: kind, Object: c.object}
				firstInsert, resumeInsert := make(chan struct{}), make(chan struct{})
				var releaseInsert sync.Once
				defer releaseInsert.Do(func() { close(resumeInsert) })
				var createCalls atomic.Int32
				nextInsert := make(chan struct{})
				createHook := w.app.OnRecordCreate(c.collection).BindFunc(func(event *core.RecordEvent) error {
					switch createCalls.Add(1) {
					case 1:
						if replay {
							close(firstInsert)
							<-resumeInsert
						}
					case 2:
						close(nextInsert)
					}
					return event.Next()
				})
				defer w.app.OnRecordCreate(c.collection).Unbind(createHook)
				firstDone := make(chan orderedInboxResult, 1)
				if replay {
					go func() {
						status, err := w.deliverOrderedActivity(handler, actor, w.recipient, first)
						firstDone <- orderedInboxResult{status, err}
					}()
					waitInboxStage(t, firstInsert, "first insert")
					if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, undo); status != http.StatusOK || err != nil {
						t.Fatalf("undo: %d, %v", status, err)
					}
				} else if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, first); status != http.StatusOK || err != nil {
					t.Fatalf("first action: %d, %v", status, err)
				}
				lookup, resumeLookup := make(chan struct{}), make(chan struct{})
				var releaseLookup sync.Once
				defer releaseLookup.Do(func() { close(resumeLookup) })
				var seen atomic.Bool
				readDB := w.app.ConcurrentDB().(*dbx.DB)
				previous := readDB.QueryLogFunc
				readDB.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
					if previous != nil {
						previous(ctx, elapsed, query, rows, err)
					}
					if c.targetLookup(query) && seen.CompareAndSwap(false, true) {
						close(lookup)
						<-resumeLookup
					}
				}
				defer func() { readDB.QueryLogFunc = previous }()
				removed := firstDone
				if replay {
					releaseInsert.Do(func() { close(resumeInsert) })
				} else {
					removed = make(chan orderedInboxResult, 1)
					go func() {
						status, err := w.deliverOrderedActivity(handler, actor, w.recipient, undo)
						removed <- orderedInboxResult{status, err}
					}()
				}
				waitInboxStage(t, lookup, "removal lookup")
				record, err := w.app.FindFirstRecordByData(c.collection, c.actorField, actor.Id)
				if err != nil {
					t.Fatal(err)
				}
				writerHeld, resumeWriter := make(chan struct{}), make(chan struct{})
				var releaseWriter sync.Once
				defer releaseWriter.Do(func() { close(resumeWriter) })
				writerDone := make(chan error, 1)
				var writerReleased atomic.Bool
				go func() {
					writerDone <- w.app.RunInTransaction(func(tx core.App) error {
						close(writerHeld)
						<-resumeWriter
						return nil
					})
				}()
				waitInboxStage(t, writerHeld, "writer transaction")
				deleteStarted, resumeDelete := make(chan struct{}), make(chan struct{})
				var releaseDelete sync.Once
				defer releaseDelete.Do(func() { close(resumeDelete) })
				deleteHook := w.app.OnRecordDelete(c.collection).BindFunc(func(event *core.RecordEvent) error {
					if !writerReleased.Load() {
						close(deleteStarted)
						<-resumeDelete
					}
					return event.Next()
				})
				defer w.app.OnRecordDelete(c.collection).Unbind(deleteHook)
				writeDB := w.app.NonconcurrentDB().(*dbx.DB).DB()
				waits := writeDB.Stats().WaitCount
				releaseLookup.Do(func() { close(resumeLookup) })
				waitInboxRemoval(t, writeDB, waits, deleteStarted)
				nextDone := make(chan orderedInboxResult, 1)
				go func() {
					status, err := w.deliverOrderedActivity(handler, actor, w.recipient, next)
					nextDone <- orderedInboxResult{status, err}
				}()
				waitInboxStage(t, nextInsert, "next insert")
				writerReleased.Store(true)
				releaseWriter.Do(func() { close(resumeWriter) })
				if err := <-writerDone; err != nil {
					t.Fatal(err)
				}
				orderedInboxResultOK(t, nextDone)
				releaseDelete.Do(func() { close(resumeDelete) })
				orderedInboxResultOK(t, removed)
				assertInboxAction(t, w, c, actor, 1, record.Id)
				if kind == pub.FollowType {
					assertInboxFollowAccepted(t, w, next)
					notifications, err := w.app.FindAllRecords("notifications", dbx.HashExp{"type": "new_follower", "author": actor.Id})
					if err != nil || len(notifications) != 1 {
						t.Fatalf("follow notifications=%d, want 1: %v", len(notifications), err)
					}
				}
			})
		}
	}
}

type inboxBodyPause struct {
	io.ReadCloser
	started, resume chan struct{}
	once            sync.Once
}

func (body *inboxBodyPause) Read(data []byte) (int, error) {
	body.once.Do(func() { close(body.started); <-body.resume })
	return body.ReadCloser.Read(data)
}

func TestInboxOlderUndoKeepsNewerMarkAfterSecondPass(t *testing.T) {
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
		t.Run(string(kind), func(t *testing.T) {
			w := newKnownKeyWorld(t)
			handler := knownKeyInboxRouter(t, w)
			actor := w.sender(t, "sender", "remote.example", true)
			c := newInboxActionCase(t, w, kind)
			first := pub.Activity{ID: "https://remote.example/activities/first", Type: kind, Object: c.object}
			older := pub.Activity{ID: "https://remote.example/activities/older-undo", Type: pub.UndoType, Object: &first}
			between := pub.Activity{ID: "https://remote.example/activities/between", Type: kind, Object: c.object}
			newer := pub.Activity{ID: "https://remote.example/activities/newer-undo", Type: pub.UndoType, Object: &first}
			insert, resumeInsert, saved, resumeSaved := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseInsert, releaseSaved sync.Once
			defer releaseInsert.Do(func() { close(resumeInsert) })
			defer releaseSaved.Do(func() { close(resumeSaved) })
			var heldCreate, heldSaved atomic.Bool
			createHook := w.app.OnRecordCreate(c.collection).BindFunc(func(event *core.RecordEvent) error {
				if heldCreate.CompareAndSwap(false, true) {
					close(insert)
					<-resumeInsert
				}
				return event.Next()
			})
			defer w.app.OnRecordCreate(c.collection).Unbind(createHook)
			savedHook := w.app.OnRecordAfterCreateSuccess(c.collection).BindFunc(func(event *core.RecordEvent) error {
				if heldSaved.CompareAndSwap(false, true) {
					close(saved)
					<-resumeSaved
				}
				return event.Next()
			})
			defer w.app.OnRecordAfterCreateSuccess(c.collection).Unbind(savedHook)
			firstDone := make(chan orderedInboxResult, 1)
			go func() {
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, first)
				firstDone <- orderedInboxResult{status, err}
			}()
			waitInboxStage(t, insert, "first insert")
			lookup, resumeLookup := make(chan struct{}), make(chan struct{})
			var releaseLookup sync.Once
			defer releaseLookup.Do(func() { close(resumeLookup) })
			var heldLookup atomic.Bool
			readDB := w.app.ConcurrentDB().(*dbx.DB)
			previous := readDB.QueryLogFunc
			readDB.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
				if previous != nil {
					previous(ctx, elapsed, query, rows, err)
				}
				if strings.HasPrefix(query, "SELECT") && strings.Contains(query, "FROM `"+c.collection+"`") && heldLookup.CompareAndSwap(false, true) {
					if err != nil || rows == nil || rows.Next() {
						t.Error("older Undo's first lookup was not empty")
					}
					close(lookup)
					<-resumeLookup
				}
			}
			defer func() { readDB.QueryLogFunc = previous }()
			olderDone := make(chan orderedInboxResult, 1)
			go func() {
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, older)
				olderDone <- orderedInboxResult{status, err}
			}()
			waitInboxStage(t, lookup, "older Undo lookup")
			bodyStarted, resumeBody := make(chan struct{}), make(chan struct{})
			var releaseBody sync.Once
			defer releaseBody.Do(func() { close(resumeBody) })
			pausedHandler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				request.Body = &inboxBodyPause{ReadCloser: request.Body, started: bodyStarted, resume: resumeBody}
				handler.ServeHTTP(response, request)
			})
			betweenDone := make(chan orderedInboxResult, 1)
			go func() {
				status, err := w.deliverOrderedActivity(pausedHandler, actor, w.recipient, between)
				betweenDone <- orderedInboxResult{status, err}
			}()
			waitInboxStage(t, bodyStarted, "middle action body")
			if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, newer); status != http.StatusOK || err != nil {
				t.Fatalf("newer Undo: %d, %v", status, err)
			}
			releaseInsert.Do(func() { close(resumeInsert) })
			waitInboxStage(t, saved, "first stored action")
			releaseLookup.Do(func() { close(resumeLookup) })
			orderedInboxResultOK(t, olderDone)
			assertInboxAction(t, w, c, actor, 0, "")
			releaseBody.Do(func() { close(resumeBody) })
			orderedInboxResultOK(t, betweenDone)
			assertInboxAction(t, w, c, actor, 0, "")
			releaseSaved.Do(func() { close(resumeSaved) })
			orderedInboxResultOK(t, firstDone)
			assertInboxAction(t, w, c, actor, 0, "")
		})
	}
}

func TestInboxRepeatedFollowReceivesAccept(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	c := newInboxActionCase(t, w, pub.FollowType)
	first := pub.Activity{ID: "https://remote.example/activities/first", Type: pub.FollowType, Object: c.object}
	repeated := pub.Activity{ID: "https://remote.example/activities/repeated", Type: pub.FollowType, Object: c.object}
	for _, activity := range []pub.Activity{first, repeated} {
		if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, activity); status != http.StatusOK || err != nil {
			t.Fatalf("%s: %d, %v", activity.ID, status, err)
		}
		assertInboxFollowAccepted(t, w, activity)
	}
	assertInboxAction(t, w, c, actor, 1, "")
	notifications, err := w.app.FindAllRecords("notifications", dbx.HashExp{"type": "new_follower", "author": actor.Id})
	if err != nil || len(notifications) != 1 {
		t.Fatalf("follow notifications=%d, want 1: %v", len(notifications), err)
	}
}
