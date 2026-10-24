package util

import (
	"errors"
	"strings"

	pub "github.com/go-ap/activitypub"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// Parallel deliveries of a new remote trail or list all try to insert it, and
// the unique IRI index admits only one of them. The others continue with the
// stored copy, exactly as a later delivery would. The retry adds no ownership
// checks; those belong to the underlying importer and its caller.
func TrailFromActivity(activity pub.Activity, app core.App, actor *core.Record) (*core.Record, error) {
	record, _, err := TrailFromConcurrentActivity(activity, app, actor)
	return record, err
}

// TrailFromConcurrentActivity also reports whether a parallel delivery of the
// activity stored the trail while this one tried to insert it.
func TrailFromConcurrentActivity(activity pub.Activity, app core.App, actor *core.Record) (*core.Record, bool, error) {
	record, err := trailFromActivity(activity, app, actor)
	if remoteObjectStoredMeanwhile(app, "trails", record, err) {
		record, err = trailFromActivity(activity, app, actor)
		return record, err == nil, err
	}
	return record, false, err
}

func ListFromActivity(activity pub.Activity, app core.App, actor *core.Record) (*core.Record, error) {
	record, err := listFromActivity(activity, app, actor)
	if remoteObjectStoredMeanwhile(app, "lists", record, err) {
		return listFromActivity(activity, app, actor)
	}
	return record, err
}

func remoteObjectStoredMeanwhile(app core.App, collection string, record *core.Record, err error) bool {
	if err == nil || record == nil || !record.IsNew() || record.GetString("iri") == "" || !remoteObjectIRITaken(err, collection) {
		return false
	}
	_, findErr := app.FindFirstRecordByData(collection, "iri", record.GetString("iri"))
	return findErr == nil
}

func remoteObjectIRITaken(err error, collection string) bool {
	if err == nil {
		return false
	}
	var invalid validation.Errors
	if errors.As(err, &invalid) {
		var taken validation.Error
		if errors.As(invalid["iri"], &taken) && taken.Code() == "validation_not_unique" {
			return true
		}
	}
	message := strings.ToLower(err.Error())
	marker := "unique constraint failed: " + collection + ".iri"
	start := strings.Index(message, marker)
	if start < 0 {
		return false
	}
	remaining := strings.TrimSpace(message[start+len(marker):])
	return remaining == "" || strings.HasPrefix(remaining, "(")
}
