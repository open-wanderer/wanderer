package util

import (
	"fmt"
	"net/url"
	"strings"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// IRIHasRecordIdentity recognizes the stable path used when this instance
// creates an object. Remote copies receive a different database ID.
func IRIHasRecordIdentity(iri, collection, id string) bool {
	kind := map[string]string{"trails": "trail", "lists": "list", "summit_logs": "summit-log", "comments": "comment", "waypoints": "waypoint"}[collection]
	parsed, err := url.Parse(iri)
	return err == nil && kind != "" && id != "" && parsed.Host != "" && parsed.User == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.RawQuery == "" && parsed.Fragment == "" &&
		parsed.EscapedPath() == parsed.Path && strings.HasSuffix(parsed.Path, "/api/v1/"+kind+"/"+id)
}

func ValidateRemoteObjectAuthor(actor *core.Record, iri string, attribution pub.Item) error {
	object, objectErr := url.Parse(iri)
	signer, signerErr := url.Parse(actor.GetString("iri"))
	if actor.GetBool("is_local") || objectErr != nil || signerErr != nil || object.User != nil || signer.User != nil ||
		(object.Scheme != "http" && object.Scheme != "https") || (signer.Scheme != "http" && signer.Scheme != "https") ||
		object.Host == "" || !strings.EqualFold(object.Host, signer.Host) || IsLocalIRI(iri) {
		return fmt.Errorf("inbound object origin does not match its author")
	}
	if attribution != nil && ItemID(attribution) != actor.GetString("iri") {
		return fmt.Errorf("inbound object attribution does not match its author")
	}
	return nil
}

// RecordHasLocalIdentity includes objects created before an ORIGIN change.
// A local author alone does not establish provenance for a historical cache.
func RecordHasLocalIdentity(app core.App, record *core.Record) bool {
	if IsLocalIRI(record.GetString("iri")) {
		return true
	}
	if !IRIHasRecordIdentity(record.GetString("iri"), record.Collection().Name, record.Id) {
		return false
	}

	author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	return err == nil && author.GetBool("is_local")
}
