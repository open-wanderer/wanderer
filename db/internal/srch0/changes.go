package srch0

import (
	"encoding/json"
	"fmt"
)

type Change struct {
	ID          string `json:"id"`
	CaseID      string `json:"case_id"`
	BasisDigest string `json:"basis_digest"`
	Observed    Object `json:"observed"`
}

// ResolveObservation ersetzt wie scripts/srch0/expectations.mjs die vollständige
// Beobachtung. Basisfall, Basisdigest und Änderungsobjekt bleiben unverändert.
func ResolveObservation(c Case, changes []Change) (Object, error) {
	var active *Change
	for i := range changes {
		if changes[i].CaseID == c.ID {
			if active != nil {
				return nil, fmt.Errorf("%s: mehrere aktive Änderungen", c.ID)
			}
			active = &changes[i]
		}
	}
	observation := c.Observed
	if active != nil {
		if active.BasisDigest != c.Digest {
			return nil, fmt.Errorf("%s: Änderung gehört zu einem anderen Basisdigest", c.ID)
		}
		if active.Observed == nil {
			return nil, fmt.Errorf("%s: aktive Änderung enthält keine Beobachtung", c.ID)
		}
		observation = active.Observed
	}
	data, err := json.Marshal(observation)
	if err != nil {
		return nil, err
	}
	var result Object
	err = json.Unmarshal(data, &result)
	return result, err
}
