package pluginsystem

import "testing"

func TestInstanceLockLifecycle(t *testing.T) {
	if _, ok := TryLockInstance(""); ok {
		t.Fatal("empty instance ID locked")
	}
	release, ok := TryLockInstance(t.Name())
	if !ok || !InstanceBusy(t.Name()) {
		t.Fatal("instance did not become busy")
	}
	if _, ok := TryLockInstance(t.Name()); ok {
		t.Fatal("instance locked twice")
	}
	release()
	if InstanceBusy(t.Name()) {
		t.Fatal("released instance stayed busy")
	}
	next, ok := TryLockInstance(t.Name())
	if !ok {
		t.Fatal("could not lock instance again")
	}
	defer next()
	release()
	if !InstanceBusy(t.Name()) {
		t.Fatal("old release unlocked the next operation")
	}
}
