package federation

import "testing"

func TestSummitLogCreateSkipsLoopbackPhoto(t *testing.T) {
	f := attachmentApp(t)
	server, hits := attachmentServer(t, attachmentSmallBody)
	activity := attachmentSummitLogActivity(t, f, attachmentImage(server.URL+"/internal.png"))
	if err := ProcessCreateOrUpdateActivity(f.app, f.y, f.owner.alice, activity); err != nil {
		t.Fatalf("summit log with an unreachable photo must still be stored: %v", err)
	}
	if count := hits.total(); count != 0 {
		t.Errorf("loopback server received %d requests, want 0", count)
	}
	stored := attachmentStored(t, f.app, "summit_logs", ownerRemoteLogIRI)
	if photos := stored.GetStringSlice("photos"); len(photos) != 0 {
		t.Errorf("stored photos = %v, want none", photos)
	}
}
