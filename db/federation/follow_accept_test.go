package federation

import (
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"
)

// TestProcessAcceptActivityIRIOnlyObjectReturnsError verifies that ProcessAcceptActivity
// returns a non-nil error (and does NOT panic) when the Accept activity's Object field
// is set to a plain pub.IRI rather than a *pub.Activity struct.
// Many ActivityPub implementations send Accept with an IRI-only object.
func TestProcessAcceptActivityIRIOnlyObjectReturnsError(t *testing.T) {
	// Build an Accept activity whose Object is an IRI (not a *pub.Activity).
	acceptActivity := pub.ActivityNew(
		pub.IRI("https://remote.example.com/activity/accept/1"),
		pub.AcceptType,
		pub.IRI("https://remote.example.com/api/v1/activitypub/activity/abc"),
	)

	// Pass nil app and nil actor — the guard must return before any DB lookup.
	err := ProcessAcceptActivity(nil, nil, *acceptActivity)

	if err == nil {
		t.Fatal("ProcessAcceptActivity returned nil error — expected a non-nil error for IRI-only Accept object")
	}
	if !strings.Contains(err.Error(), "ProcessAcceptActivity: object is not *pub.Activity") {
		t.Errorf("error message = %q, want it to contain \"ProcessAcceptActivity: object is not *pub.Activity\"", err.Error())
	}
}
