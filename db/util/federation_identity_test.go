package util

import "testing"

func TestIRIHasRecordIdentityAcrossOriginChanges(t *testing.T) {
	for _, origin := range []string{"http://localhost:3000", "http://192.0.2.15:3000", "https://old.example", "https://www.old.example/prefix"} {
		for collection, kind := range map[string]string{"trails": "trail", "lists": "list", "summit_logs": "summit-log", "comments": "comment", "waypoints": "waypoint"} {
			if !IRIHasRecordIdentity(origin+"/api/v1/"+kind+"/own-record", collection, "own-record") {
				t.Fatalf("historical identity rejected: %s %s", origin, collection)
			}
		}
	}
	for _, iri := range []string{"https://remote.example/api/v1/trail/remote-id", "https://remote.example/api/v1/list/own-record", "https://remote.example/api/v1/trail/own-record/", "https://user@remote.example/api/v1/trail/own-record", "https://remote.example/api/v1/trail/own-record?x=1", "https://remote.example/api/v1/trail/own-record#fragment", "https://remote.example/api/v1/trail/own%2drecord", "file:///api/v1/trail/own-record", "ftp://remote.example/api/v1/trail/own-record", "//remote.example/api/v1/trail/own-record"} {
		if IRIHasRecordIdentity(iri, "trails", "own-record") {
			t.Fatalf("nonmatching record URL accepted: %s", iri)
		}
	}
	if IRIHasRecordIdentity("https://remote.example/api/v1/trail/own-record", "unknown", "own-record") || IRIHasRecordIdentity("https://remote.example/api/v1/trail/", "trails", "") {
		t.Fatal("unknown collection or empty record identity accepted")
	}
}
