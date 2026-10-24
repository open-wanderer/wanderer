package federation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	pub "github.com/go-ap/activitypub"
)

// UnmarshalInboxActivity expands standard public audience names before the
// ActivityStreams decoder reads arrays. Signature checks still use the original
// request bytes.
func UnmarshalInboxActivity(body []byte) (pub.Activity, error) {
	var activity pub.Activity
	var document any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&document); err != nil {
		return activity, err
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return activity, fmt.Errorf("invalid activity document")
	}
	normalizePublicAudience(document)
	normalized, err := json.Marshal(document)
	if err != nil {
		return activity, err
	}
	err = activity.UnmarshalJSON(normalized)
	return activity, err
}

func normalizePublicAudience(value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "to" || key == "cc" {
				switch audience := child.(type) {
				case string:
					value[key] = expandPublicAudience(audience)
				case []any:
					for i, recipient := range audience {
						if iri, ok := recipient.(string); ok {
							audience[i] = expandPublicAudience(iri)
						}
					}
				}
			}
			normalizePublicAudience(child)
		}
	case []any:
		for _, child := range value {
			normalizePublicAudience(child)
		}
	}
}

func expandPublicAudience(iri string) string {
	if iri == "Public" || iri == "as:Public" {
		return pub.PublicNS.String()
	}
	return iri
}
