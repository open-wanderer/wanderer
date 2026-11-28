package importer

import (
	"context"
	"testing"
	"time"
)

func TestFinishTimeStartsWhenTheSyncEnds(t *testing.T) {
	sync, endSync := context.WithCancel(context.Background())
	defer endSync()
	finish, cancel := finishContext(sync, 10*time.Millisecond)
	defer cancel()

	select {
	case <-finish.Done():
		t.Fatal("finish time ran out while the sync was still running")
	case <-time.After(100 * time.Millisecond):
	}
	endSync()
	select {
	case <-finish.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("finish time did not run out after the sync ended")
	}
}
