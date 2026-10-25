package routes

// Keep a shared result separate from work performed for the current request
// so a deletion confirmation can distinguish a waiter from a completed fetch.
type knownInboxActorReloadError struct {
	err          error
	fetched      bool
	claimRefused bool
	waited       bool
}

func (e *knownInboxActorReloadError) Error() string { return e.err.Error() }

func (e *knownInboxActorReloadError) Unwrap() error { return e.err }

// The callback already logged the diagnostic details. Callers use a neutral
// error so a shared refresh failure cannot disclose them in an HTTP response.
type knownInboxActorReloadPanicError struct{}

func (e *knownInboxActorReloadPanicError) Error() string { return "actor refresh did not complete" }
