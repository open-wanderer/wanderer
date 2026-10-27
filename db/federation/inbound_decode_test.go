package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestInboxActivityDecodingKeepsPublicAliasesAndOtherFields(t *testing.T) {
	body := []byte(`{"type":"Undo","actor":"https://peer.example/users/member","object":{"type":"Follow","actor":"https://peer.example/users/member","object":"https://local.example/users/reader","to":["as:Public"],"cc":["Public","https://peer.example/users/other"]},"to":["Public"],"cc":["as:Public"]}`)
	activity, err := UnmarshalInboxActivity(body)
	if err != nil {
		t.Fatal(err)
	}
	nested, err := pub.ToActivity(activity.Object)
	if err != nil {
		t.Fatal(err)
	}
	if nested.Actor.GetID().String() != "https://peer.example/users/member" || nested.Object.GetID().String() != "https://local.example/users/reader" {
		t.Fatal("nested activity fields changed")
	}
	for _, recipients := range []pub.ItemCollection{activity.To, activity.CC, nested.To, nested.CC} {
		if len(recipients) == 0 || recipients[0].GetID() != pub.PublicNS {
			t.Fatalf("public audience lost: %v", recipients)
		}
	}
	if len(nested.CC) != 2 || nested.CC[1].GetID().String() != "https://peer.example/users/other" {
		t.Fatal("other recipient changed")
	}
}

func TestInboxActivityDecodingRejectsTrailingDocuments(t *testing.T) {
	if _, err := UnmarshalInboxActivity([]byte(`{"type":"Create"} {"type":"Update"}`)); err == nil {
		t.Fatal("multiple documents accepted")
	}
}
