package federation

import (
	"fmt"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestInboxRetractionMarkersExpireAndStayBounded(t *testing.T) {
	inboxRetractions.Lock()
	previous, sequence := inboxRetractions.entries, inboxRetractions.sequence
	inboxRetractions.entries = make(map[inboxRetractionKey]inboxRetraction)
	inboxRetractions.Unlock()
	defer func() {
		inboxRetractions.Lock()
		inboxRetractions.entries, inboxRetractions.sequence = previous, sequence
		inboxRetractions.Unlock()
	}()
	key := inboxRetractionKey{actor: "https://remote.example/users/sender", object: "https://remote.example/trails/walk", kind: "object"}
	deletion := pub.Activity{Type: pub.DeleteType, Object: pub.IRI(key.object)}
	before := time.Now()
	rememberInboxRetraction(key, deletion, inboxRetractionItem{}, time.Now())
	first, ok := takeInboxRetraction(key, time.Time{})
	if !ok || first.once || first.expires.Before(before.Add(time.Minute)) || first.expires.After(time.Now().Add(time.Minute)) {
		t.Fatalf("deletion lifetime: %+v", first)
	}
	if _, ok := takeInboxRetraction(key, time.Time{}); !ok {
		t.Fatal("delete mark was consumed")
	}
	rememberInboxRetractionItem(key, inboxRetractionItem{collection: "trails", id: "morningwalk0001"})
	updated, _ := takeInboxRetraction(key, time.Time{})
	if updated.expires != first.expires || len(updated.itemIDs) != 1 {
		t.Fatal("remembering the item changed the deletion window")
	}
	inboxRetractions.Lock()
	expired := inboxRetractions.entries[key]
	expired.expires = time.Now().Add(-time.Second)
	inboxRetractions.entries[key] = expired
	inboxRetractions.Unlock()
	if _, ok := takeInboxRetraction(key, time.Time{}); ok {
		t.Fatal("expired mark was returned")
	}
	key.kind = string(pub.LikeType)
	undo := pub.Activity{Type: pub.UndoType, Object: &pub.Activity{Type: pub.LikeType, Object: pub.IRI(key.object)}}
	firstUndo := rememberInboxRetraction(key, undo, inboxRetractionItem{}, time.Now())
	nextUndo := rememberInboxRetraction(key, undo, inboxRetractionItem{}, time.Now())
	finishInboxUndo(key, firstUndo)
	if entry, ok := takeInboxRetraction(key, time.Time{}); !ok || entry.sequence != nextUndo {
		t.Fatal("finishing an older undo removed a newer mark")
	}
	if _, ok := takeInboxRetraction(key, time.Time{}); ok {
		t.Fatal("undo mark was not consumed")
	}
	completedUndo := rememberInboxRetraction(key, undo, inboxRetractionItem{}, time.Now())
	finishInboxUndo(key, completedUndo)
	if _, ok := takeInboxRetraction(key, time.Time{}); ok {
		t.Fatal("successful second undo retained its mark")
	}

	key.kind = "object"
	rememberInboxRetraction(key, deletion, inboxRetractionItem{}, time.Now())
	for i := 0; i < inboxRetractionLimit; i++ {
		next := key
		next.object = fmt.Sprintf("https://remote.example/trails/%d", i)
		rememberInboxRetraction(next, deletion, inboxRetractionItem{}, time.Now())
	}
	if _, ok := takeInboxRetraction(key, time.Time{}); ok {
		t.Fatal("oldest mark was retained beyond the limit")
	}
	inboxRetractions.Lock()
	count := len(inboxRetractions.entries)
	inboxRetractions.Unlock()
	if count != inboxRetractionLimit {
		t.Fatalf("marker count: %d", count)
	}
}

func TestInboxRetractionKeysMatchHandlerIdentity(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	actor := core.NewRecord(core.NewBaseCollection("activitypub_actors"))
	actor.Set("iri", "https://remote.example/api/v1/activitypub/user/sender")
	object := pub.IRI("https://local.example/api/v1/trail/morningwalk0001")
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
		original := pub.Activity{ID: "https://remote.example/activities/original", Type: kind, Object: object}
		first, ok := inboxOrderKey(app, actor, original)
		original.ID = "https://remote.example/activities/another-id"
		undo, undoOK := inboxOrderKey(app, actor, pub.Activity{Type: pub.UndoType, Object: &original})
		if !ok || !undoOK || first != undo {
			t.Fatalf("%s original and undo keys differ", kind)
		}
	}
	for _, kind := range []pub.ActivityVocabularyType{pub.AcceptType, pub.AnnounceType} {
		if _, ok := inboxOrderKey(app, actor, pub.Activity{Type: kind, Object: object}); ok {
			t.Fatalf("%s acquired a retraction key", kind)
		}
	}
	for _, iri := range []string{actor.GetString("iri"), "https://remote.example/users/sender"} {
		actor.Set("iri", iri)
		if _, ok := inboxOrderKey(app, actor, pub.Activity{Type: pub.DeleteType, Object: pub.IRI(iri)}); ok {
			t.Fatal("actor deletion acquired a retraction key")
		}
	}
}

func TestInboxOriginalArrivalsExpireAndStayBounded(t *testing.T) {
	inboxRetractions.Lock()
	previous := inboxRetractions.originals
	inboxRetractions.originals = make(map[inboxRetractionKey]inboxOriginal)
	inboxRetractions.Unlock()
	defer func() {
		inboxRetractions.Lock()
		inboxRetractions.originals = previous
		inboxRetractions.Unlock()
	}()
	key := inboxRetractionKey{dataDir: t.TempDir(), actor: "https://remote.example/users/sender", object: "https://local.example/trails/walk", kind: string(pub.LikeType)}
	received := time.Now()
	noteInboxOriginal(key, received)
	if !inboxOriginalArrivedAfter(key, received.Add(-time.Second)) || inboxOriginalArrivedAfter(key, received) {
		t.Fatal("arrival comparison did not retain a strictly later action")
	}
	noteInboxOriginal(key, received.Add(-time.Second))
	if !inboxOriginalArrivedAfter(key, received.Add(-time.Millisecond)) {
		t.Fatal("older delivery replaced a newer arrival")
	}
	other := key
	other.actor = "https://remote.example/users/another"
	if inboxOriginalArrivedAfter(other, received.Add(-time.Second)) {
		t.Fatal("arrival crossed actors")
	}
	inboxRetractions.Lock()
	expired := inboxRetractions.originals[key]
	if expired.expires.Before(received.Add(time.Minute)) || expired.expires.After(time.Now().Add(time.Minute)) {
		t.Error("unexpected arrival lifetime")
	}
	expired.expires = time.Now().Add(-time.Second)
	inboxRetractions.originals[key] = expired
	inboxRetractions.Unlock()
	if inboxOriginalArrivedAfter(key, received.Add(-time.Second)) {
		t.Fatal("expired arrival was returned")
	}
	noteInboxOriginal(other, received.Add(time.Millisecond))
	inboxRetractions.Lock()
	_, retained := inboxRetractions.originals[key]
	inboxRetractions.Unlock()
	if retained {
		t.Fatal("expired arrival retained on insert")
	}
	for index := 0; index < inboxRetractionLimit; index++ {
		next := key
		next.object = fmt.Sprintf("https://local.example/trails/%d", index)
		noteInboxOriginal(next, received.Add(time.Duration(index+2)*time.Millisecond))
	}
	inboxRetractions.Lock()
	count := len(inboxRetractions.originals)
	_, retained = inboxRetractions.originals[other]
	inboxRetractions.Unlock()
	if count != inboxRetractionLimit || retained {
		t.Fatalf("arrival count=%d, oldest retained=%t", count, retained)
	}
}

func TestInboxUndoMarkRemainsForEarlierArrival(t *testing.T) {
	key := inboxRetractionKey{dataDir: t.TempDir(), actor: "https://remote.example/users/sender", object: "https://local.example/trails/walk", kind: string(pub.LikeType)}
	received := time.Now()
	undo := pub.Activity{Type: pub.UndoType, Object: &pub.Activity{Type: pub.LikeType, Object: pub.IRI(key.object)}}
	sequence := rememberInboxRetraction(key, undo, inboxRetractionItem{}, received)
	if _, ok := takeInboxRetraction(key, received.Add(time.Millisecond)); ok {
		t.Fatal("later action consumed undo mark")
	}
	if _, ok := takeInboxRetraction(key, received); ok {
		t.Fatal("equal arrival consumed undo mark")
	}
	entry, ok := takeInboxRetraction(key, received.Add(-time.Millisecond))
	if !ok || entry.sequence != sequence {
		t.Fatal("undo mark lost before earlier action")
	}
	if _, ok := takeInboxRetraction(key, received.Add(-time.Millisecond)); ok {
		t.Fatal("earlier action did not consume undo mark")
	}
}
