package pluginsystem

import "testing"

func TestPluginIssueRecordID(t *testing.T) {
	valid := pluginIssueRecordID(LocalPluginIssue{ID: "komoot", Dir: "/plugins/komoot"})
	if valid != "komoot" {
		t.Fatalf("pluginIssueRecordID(valid) = %q, want komoot", valid)
	}

	first := pluginIssueRecordID(LocalPluginIssue{ID: "@@@", Dir: "/plugins/@@@"})
	second := pluginIssueRecordID(LocalPluginIssue{ID: "***", Dir: "/plugins/***"})
	if first == second {
		t.Fatalf("invalid plugin issue ids collided: %q", first)
	}
	for _, got := range []string{first, second} {
		if !pluginIDPattern.MatchString(got) {
			t.Fatalf("pluginIssueRecordID() = %q, not a valid plugin id", got)
		}
	}
}
