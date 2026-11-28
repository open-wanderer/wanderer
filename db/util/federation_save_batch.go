package util

// ValidFederationSaveBatchID bounds the process-local save leases supplied by
// clients. UUIDs fit this alphabet; short or oversized keys are rejected.
func ValidFederationSaveBatchID(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return false
		}
	}
	return true
}
