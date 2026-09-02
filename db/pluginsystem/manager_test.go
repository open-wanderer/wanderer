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

func TestMergePluginIssuesPreservesSetupErrorCodes(t *testing.T) {
	infos := []PluginInfo{{
		ID:     "cached",
		Path:   "/plugins/cached",
		Status: "available",
	}}
	issues := []LocalPluginIssue{
		{
			ID:             "cached",
			Name:           "Cached",
			Dir:            "/plugins/cached",
			SetupErrorCode: SetupErrorCodeManifestInvalid,
			Error:          "private cached diagnostic",
		},
		{
			ID:             "new",
			Name:           "New",
			Dir:            "/plugins/new",
			SetupErrorCode: SetupErrorCodeRuntimeEntrypointMissing,
			Error:          "private new diagnostic",
		},
	}

	got := mergePluginIssues(infos, map[string]int{"/plugins/cached": 0}, issues)
	if len(got) != 2 {
		t.Fatalf("got %d plugin infos, want 2", len(got))
	}
	if got[0].Status != "error" ||
		got[0].SetupErrorCode != SetupErrorCodeManifestInvalid ||
		got[0].Error != "private cached diagnostic" {
		t.Fatalf("cached plugin issue was not overlaid: %#v", got[0])
	}
	if got[1].ID != "new" ||
		got[1].SetupErrorCode != SetupErrorCodeRuntimeEntrypointMissing ||
		got[1].Error != "private new diagnostic" {
		t.Fatalf("new plugin issue was not appended: %#v", got[1])
	}
}
