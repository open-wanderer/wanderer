package federation

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const inboxRetractionLimit = 4096
const inboxRetractionLifetime = time.Minute

type inboxRetractionKey struct {
	dataDir, actor, kind, object string
}

type inboxRetractionItem struct {
	collection, id string
}

type inboxRetraction struct {
	activity pub.Activity
	marked   time.Time
	expires  time.Time
	sequence uint64
	once     bool
	itemIDs  []inboxRetractionItem
}

// Only accesses to the bounded marker table are serialized. Activity handlers,
// network requests and deliveries to separate inboxes always run independently.
var inboxRetractions = struct {
	sync.Mutex
	entries   map[inboxRetractionKey]inboxRetraction
	originals map[inboxRetractionKey]inboxOriginal
	sequence  uint64
}{entries: make(map[inboxRetractionKey]inboxRetraction), originals: make(map[inboxRetractionKey]inboxOriginal)}

type inboxOriginal struct {
	received, expires time.Time
}

// Remember Like and Follow arrivals before their handlers run. An older Undo
// must not remove an action which arrived later for the same actor and target.
func noteInboxOriginal(key inboxRetractionKey, received time.Time) {
	inboxRetractions.Lock()
	defer inboxRetractions.Unlock()
	now := time.Now()
	for k, entry := range inboxRetractions.originals {
		if !entry.expires.After(now) {
			delete(inboxRetractions.originals, k)
		}
	}
	if previous, ok := inboxRetractions.originals[key]; ok && previous.received.After(received) {
		return
	}
	if _, ok := inboxRetractions.originals[key]; !ok && len(inboxRetractions.originals) >= inboxRetractionLimit {
		var oldest inboxRetractionKey
		first := true
		for k, entry := range inboxRetractions.originals {
			if first || entry.received.Before(inboxRetractions.originals[oldest].received) {
				oldest, first = k, false
			}
		}
		delete(inboxRetractions.originals, oldest)
	}
	inboxRetractions.originals[key] = inboxOriginal{received: received, expires: now.Add(inboxRetractionLifetime)}
}

func inboxOriginalArrivedAfter(key inboxRetractionKey, received time.Time) bool {
	inboxRetractions.Lock()
	defer inboxRetractions.Unlock()
	entry, ok := inboxRetractions.originals[key]
	return ok && entry.expires.After(time.Now()) && entry.received.After(received)
}

func inboxOrderKey(app core.App, actor *core.Record, activity pub.Activity) (inboxRetractionKey, bool) {
	key := inboxRetractionKey{dataDir: app.DataDir(), actor: actor.GetString("iri")}
	object := activity.Object
	switch activity.Type {
	case pub.FollowType, pub.LikeType:
		key.kind = string(activity.Type)
	case pub.UndoType:
		original, ok := object.(*pub.Activity)
		if !ok || original == nil || (original.Type != pub.FollowType && original.Type != pub.LikeType) {
			return key, false
		}
		key.kind, object = string(original.Type), original.Object
	case pub.CreateType, pub.UpdateType, pub.DeleteType:
		key.kind = "object"
	default:
		return key, false
	}
	if object == nil {
		return key, false
	}
	key.object = object.GetID().String()
	if key.object == "" || key.actor == "" || (activity.Type == pub.DeleteType && (key.object == key.actor || util.ObjectKindFromIRI(key.object) == util.ObjectKindActor)) {
		return key, false
	}
	return key, true
}

func rememberInboxRetraction(key inboxRetractionKey, activity pub.Activity, item inboxRetractionItem, received time.Time) uint64 {
	inboxRetractions.Lock()
	defer inboxRetractions.Unlock()
	now := time.Now()
	for k, entry := range inboxRetractions.entries {
		if !entry.expires.After(now) {
			delete(inboxRetractions.entries, k)
		}
	}
	previous := inboxRetractions.entries[key]
	// An older Undo can finish its first lookup after a newer one has already
	// left a mark. Keep the later arrival for originals still in flight.
	if activity.Type == pub.UndoType && previous.once && previous.marked.After(received) {
		// The older Undo gets no sequence of its own, so its second pass cannot
		// finish the newer mark.
		return 0
	}
	inboxRetractions.sequence++
	// Retain only the fields needed by the retraction handlers, rather than
	// keeping an embedded original's content for the lifetime of the mark.
	compact := pub.Activity{Type: activity.Type, Actor: pub.IRI(key.actor), Object: pub.IRI(key.object)}
	if activity.Type == pub.UndoType {
		compact.Object = &pub.Activity{Type: pub.ActivityVocabularyType(key.kind), Actor: pub.IRI(key.actor), Object: pub.IRI(key.object)}
	}
	entry := inboxRetraction{activity: compact, marked: received, expires: now.Add(inboxRetractionLifetime), sequence: inboxRetractions.sequence, once: activity.Type == pub.UndoType}
	if !entry.once && !previous.once {
		entry.itemIDs = append([]inboxRetractionItem(nil), previous.itemIDs...)
	}
	if item.id != "" {
		found := false
		for _, id := range entry.itemIDs {
			found = found || id == item
		}
		if !found {
			entry.itemIDs = append(entry.itemIDs, item)
		}
	}
	inboxRetractions.entries[key] = entry
	if len(inboxRetractions.entries) > inboxRetractionLimit {
		var oldest inboxRetractionKey
		sequence := ^uint64(0)
		for k, value := range inboxRetractions.entries {
			if value.sequence < sequence {
				oldest, sequence = k, value.sequence
			}
		}
		delete(inboxRetractions.entries, oldest)
	}
	return entry.sequence
}

// A successful second Undo pass has already found and removed its original.
// Only its own marker is finished; a newer retraction keeps its own window.
func finishInboxUndo(key inboxRetractionKey, sequence uint64) {
	inboxRetractions.Lock()
	defer inboxRetractions.Unlock()
	if entry, ok := inboxRetractions.entries[key]; ok && entry.once && entry.sequence == sequence {
		delete(inboxRetractions.entries, key)
	}
}

// A handler that observes an item adds its ID without extending the deletion
// window. No network or database operation runs while this table is locked.
func rememberInboxRetractionItem(key inboxRetractionKey, item inboxRetractionItem) {
	if item.id == "" {
		return
	}
	inboxRetractions.Lock()
	defer inboxRetractions.Unlock()
	entry, ok := inboxRetractions.entries[key]
	if !ok || entry.once || !entry.expires.After(time.Now()) {
		return
	}
	for _, existing := range entry.itemIDs {
		if existing == item {
			return
		}
	}
	entry.itemIDs = append(append([]inboxRetractionItem(nil), entry.itemIDs...), item)
	inboxRetractions.entries[key] = entry
}

// An Undo mark applies only to actions that arrived before the Undo. Later
// actions leave it available for an earlier delivery still waiting to run.
func takeInboxRetraction(key inboxRetractionKey, received time.Time) (inboxRetraction, bool) {
	inboxRetractions.Lock()
	defer inboxRetractions.Unlock()
	entry, exists := inboxRetractions.entries[key]
	if !exists {
		return inboxRetraction{}, false
	}
	if !entry.expires.After(time.Now()) {
		delete(inboxRetractions.entries, key)
		return inboxRetraction{}, false
	}
	if entry.once {
		if !received.Before(entry.marked) {
			return inboxRetraction{}, false
		}
		delete(inboxRetractions.entries, key)
	}
	// The cached slice is immutable after insertion.
	return entry, true
}

func inboxRetractionItemID(app core.App, actor *core.Record, key inboxRetractionKey) inboxRetractionItem {
	var collection string
	switch util.ObjectKindFromIRI(key.object) {
	case util.ObjectKindTrail:
		collection = "trails"
	case util.ObjectKindList:
		collection = "lists"
	default:
		return inboxRetractionItem{}
	}
	record, err := app.FindFirstRecordByData(collection, "iri", key.object)
	if err != nil || record.GetString("author") != actor.Id {
		return inboxRetractionItem{}
	}
	return inboxRetractionItem{collection: collection, id: record.Id}
}

func replayInboxRetraction(app core.App, actor, recipient *core.Record, key inboxRetractionKey, entry inboxRetraction, received time.Time) error {
	if entry.activity.Type == pub.DeleteType {
		// A concurrent Delete may remove the object before another handler adds
		// its feed entry. Keep the item ID so that this late entry is removed.
		rememberInboxRetractionItem(key, inboxRetractionItemID(app, actor, key))
	}
	var err error
	if entry.activity.Type == pub.UndoType {
		// Replays use the Undo's arrival, rather than the earlier original's.
		err = processUndoActivity(app, actor, entry.activity, func() bool { return inboxOriginalArrivedAfter(key, entry.marked) })
	} else {
		err = dispatchInboxActivity(app, actor, recipient, entry.activity)
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, errUndoSuperseded) {
		err = nil
	}
	if entry.activity.Type == pub.DeleteType {
		if latest, ok := takeInboxRetraction(key, received); ok {
			for _, item := range latest.itemIDs {
				if cleanupErr := cleanRetractedInboxFeed(app, actor, key, item); err == nil {
					err = cleanupErr
				}
			}
		}
	}
	return err
}

func processOrderedInboxActivity(app core.App, actor, recipient *core.Record, activity pub.Activity, received time.Time) error {
	key, ordered := inboxOrderKey(app, actor, activity)
	if !ordered {
		return dispatchInboxActivity(app, actor, recipient, activity)
	}
	switch activity.Type {
	case pub.DeleteType:
		rememberInboxRetraction(key, activity, inboxRetractionItemID(app, actor, key), received)
		err := dispatchInboxActivity(app, actor, recipient, activity)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	case pub.UndoType:
		superseded := func() bool { return inboxOriginalArrivedAfter(key, received) }
		if superseded() {
			return nil
		}
		err := processUndoActivity(app, actor, activity, superseded)
		if errors.Is(err, errUndoSuperseded) {
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		sequence := rememberInboxRetraction(key, activity, inboxRetractionItem{}, received)
		if superseded() {
			return nil
		}
		err = processUndoActivity(app, actor, activity, superseded)
		if err == nil {
			finishInboxUndo(key, sequence)
		}
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, errUndoSuperseded) {
			return nil
		}
		return err
	default:
		if key.kind != "object" {
			noteInboxOriginal(key, received)
		}
		if _, marked := takeInboxRetraction(key, received); marked {
			return nil
		}
		err := dispatchInboxActivity(app, actor, recipient, activity)
		if entry, marked := takeInboxRetraction(key, received); marked {
			if replayErr := replayInboxRetraction(app, actor, recipient, key, entry, received); err == nil {
				err = replayErr
			}
		}
		return err
	}
}

// A stored item ID belongs to one collection, IRI and author. Do not let an
// unrelated record that subsequently reuses that ID lose its feed entries.
func cleanRetractedInboxFeed(app core.App, actor *core.Record, key inboxRetractionKey, item inboxRetractionItem) error {
	current, err := app.FindRecordById(item.collection, item.id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && (current.GetString("iri") != key.object || !inboundRecordCanDelete(app, current, actor)) {
		return nil
	}
	feedType := string(util.TrailFeed)
	if item.collection == "lists" {
		feedType = string(util.ListFeed)
	}
	entries, err := app.FindAllRecords("feed", dbx.HashExp{"item": item.id, "type": feedType})
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := app.Delete(entry); err != nil {
			return err
		}
	}
	return nil
}
