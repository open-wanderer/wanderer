package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestAcceptAndUndoRequireActivityObjects(t *testing.T) {
	f := setupAddressingTestApp(t)
	actor := f.actor(t, "sender", false)
	var nilActivity *pub.Activity
	for _, test := range []struct {
		name   string
		typ    pub.ActivityVocabularyType
		object pub.Item
	}{
		{"accept missing", pub.AcceptType, nil},
		{"accept reference", pub.AcceptType, pub.IRI("https://remote.example/follow")},
		{"accept object", pub.AcceptType, &pub.Object{Type: pub.FollowType}},
		{"accept nil activity", pub.AcceptType, nilActivity},
		{"undo missing", pub.UndoType, nil},
		{"undo reference", pub.UndoType, pub.IRI("https://remote.example/follow")},
		{"undo follow object", pub.UndoType, &pub.Object{Type: pub.FollowType}},
		{"undo like object", pub.UndoType, &pub.Object{Type: pub.LikeType}},
		{"undo nil activity", pub.UndoType, nilActivity},
	} {
		t.Run(test.name, func(t *testing.T) {
			activity := pub.Activity{Type: test.typ, Object: test.object}
			if err := ProcessInboxActivity(f.app, actor, actor, activity); err == nil {
				t.Fatal("expected an error for an object without an embedded activity")
			}
		})
	}
}

func TestUndoHandlersRequireActivityObjects(t *testing.T) {
	f := setupAddressingTestApp(t)
	actor := f.actor(t, "sender", false)
	var nilActivity *pub.Activity
	for _, handler := range []struct {
		name string
		run  func(core.App, *core.Record, pub.Activity) error
	}{
		{"follow", processUnfollowActivity},
		{"like", processUnlikeActivity},
	} {
		t.Run(handler.name, func(t *testing.T) {
			for _, object := range []pub.Item{nil, nilActivity, pub.IRI("https://remote.example/activity"), &pub.Object{Type: pub.FollowType}} {
				if err := handler.run(f.app, actor, pub.Activity{Type: pub.UndoType, Object: object}); err == nil {
					t.Fatal("expected an error for an object without an embedded activity")
				}
			}
		})
	}
}
