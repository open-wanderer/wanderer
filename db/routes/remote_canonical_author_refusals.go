package routes

import (
	"context"
	"sync"
)

type remoteAuthorRefusalsKey struct{}

// Keep dial refusals local to one author phase and its requesting viewer.
// Other failures still use the ordinary shared confirmation cache.
type remoteAuthorRefusals struct {
	sync.Mutex
	iris map[string]bool
}

func withRemoteAuthorRefusals(ctx context.Context) (context.Context, *remoteAuthorRefusals) {
	report := &remoteAuthorRefusals{iris: map[string]bool{}}
	return context.WithValue(ctx, remoteAuthorRefusalsKey{}, report), report
}

func (report *remoteAuthorRefusals) refused(iri string) bool {
	report.Lock()
	defer report.Unlock()
	return report.iris[iri]
}

func rememberRemoteAuthorRefusal(ctx context.Context, iri string) {
	report, _ := ctx.Value(remoteAuthorRefusalsKey{}).(*remoteAuthorRefusals)
	if report == nil {
		return
	}
	report.Lock()
	report.iris[iri] = true
	report.Unlock()
}
