package routes

import (
	"fmt"
	"net/url"
	"strings"
)

const canonicalRemoteRecordLimit = 30

func canonicalRecordURL(iri, kind string) (*url.URL, error) {
	parsed, err := url.Parse(iri)
	if err != nil {
		return nil, err
	}
	pieces := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(iri) > 2048 || !remoteSyncIRIMatchesOrigin(iri, iri) || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.EscapedPath() != parsed.Path || len(pieces) != 4 || pieces[0] != "api" || pieces[1] != "v1" || pieces[2] != kind || pieces[3] == "" || pieces[3] == "." || pieces[3] == ".." {
		return nil, fmt.Errorf("invalid canonical %s reference", kind)
	}
	return parsed, nil
}
