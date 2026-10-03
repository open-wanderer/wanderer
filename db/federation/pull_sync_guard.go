package federation

import (
	"database/sql"
	"errors"
	"fmt"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

// CheckRemoteObjectOwnership refuses an empty or local IRI, an IRI not on the
// author's host, and an existing row stored under another author. It returns
// the existing row, or nil when the IRI is not stored yet.
func CheckRemoteObjectOwnership(app core.App, collection, objectIRI string, author *core.Record) (*core.Record, error) {
	return checkRemoteObjectOwnership(app, collection, objectIRI, author)
}

// CheckPulledObject checks an item pulled from a trail's origin. It refuses an
// empty or local IRI, an item or author IRI not on the trail's host, and an
// existing row that belongs to another trail. It returns the existing row, or
// nil when the IRI is not stored yet. It fetches and writes nothing.
func CheckPulledObject(app core.App, collection, objectIRI, authorIRI string, trail *core.Record) (*core.Record, error) {
	if objectIRI == "" {
		return nil, fmt.Errorf("pulled object has no iri")
	}
	if util.IsLocalIRI(objectIRI) {
		return nil, fmt.Errorf("refusing pulled object referencing local object %s", objectIRI)
	}
	trailIRI := trail.GetString("iri")
	if !sameHost(objectIRI, trailIRI) {
		return nil, fmt.Errorf("object %s is not on the trail origin's host", objectIRI)
	}
	if authorIRI != "" && !sameHost(authorIRI, trailIRI) {
		return nil, fmt.Errorf("author %s of object %s is not on the trail origin's host", authorIRI, objectIRI)
	}

	existing, err := app.FindFirstRecordByData(collection, "iri", objectIRI)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if existing.GetString("trail") != trail.Id {
		return nil, fmt.Errorf("object %s belongs to another trail", objectIRI)
	}
	return existing, nil
}

// CheckPulledListTrail checks a trail pulled from a list's origin. It returns
// the stored row (nil if unknown) and whether the caller may write it. Local
// trails and trails on another host than the list are never writable: an
// existing row may only be linked, an unknown one is skipped. A trail on the
// list's host is writable only if its author is on that host too; the caller
// must still run CheckRemoteObjectOwnership. It fetches and writes nothing.
func CheckPulledListTrail(app core.App, trailIRI, authorIRI, listIRI string) (existing *core.Record, writable bool, err error) {
	if trailIRI == "" {
		return nil, false, fmt.Errorf("pulled trail has no iri")
	}

	existing, err = app.FindFirstRecordByData("trails", "iri", trailIRI)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, false, err
		}
		existing = nil
	}

	if util.IsLocalIRI(trailIRI) || !sameHost(trailIRI, listIRI) {
		return existing, false, nil
	}
	if authorIRI == "" || !sameHost(authorIRI, listIRI) {
		return nil, false, fmt.Errorf("author %q of trail %s is not on the list origin's host", authorIRI, trailIRI)
	}
	return existing, true, nil
}
