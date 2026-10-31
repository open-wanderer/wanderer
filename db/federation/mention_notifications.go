package federation

import (
	"sync"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

const mentionClaimLimit = 4096
const mentionClaimLifetime = time.Hour

type mentionClaimKey struct {
	dataDir, activity string
}

// A peer delivers one Create or Update to the user inboxes it addresses and to
// the instance inbox. Each delivery stores the object, but only the first one
// notifies the accounts it mentions.
var mentionClaims = struct {
	sync.Mutex
	expires map[mentionClaimKey]time.Time
}{expires: make(map[mentionClaimKey]time.Time)}

// claimMentionNotifications reports whether this delivery of the activity is
// the first and so notifies its mentions. An activity without an id cannot be
// recognised again and always notifies.
func claimMentionNotifications(app core.App, activity pub.Activity) bool {
	id := activity.GetID().String()
	if id == "" {
		return true
	}
	key := mentionClaimKey{dataDir: app.DataDir(), activity: id}

	mentionClaims.Lock()
	defer mentionClaims.Unlock()
	now := time.Now()
	for k, expires := range mentionClaims.expires {
		if !expires.After(now) {
			delete(mentionClaims.expires, k)
		}
	}
	if _, claimed := mentionClaims.expires[key]; claimed {
		return false
	}
	if len(mentionClaims.expires) >= mentionClaimLimit {
		var oldest mentionClaimKey
		first := true
		for k, expires := range mentionClaims.expires {
			if first || expires.Before(mentionClaims.expires[oldest]) {
				oldest, first = k, false
			}
		}
		delete(mentionClaims.expires, oldest)
	}
	mentionClaims.expires[key] = now.Add(mentionClaimLifetime)
	return true
}
