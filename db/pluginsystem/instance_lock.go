package pluginsystem

import "sync"

var connectionLocks = struct {
	sync.Mutex
	held map[string]bool
}{held: map[string]bool{}}

// TryLockInstance reserves an instance in this backend process without waiting.
// The caller releases it when the operation ends.
func TryLockInstance(id string) (func(), bool) {
	connectionLocks.Lock()
	defer connectionLocks.Unlock()
	if id == "" || connectionLocks.held[id] {
		return func() {}, false
	}
	connectionLocks.held[id] = true
	var once sync.Once
	return func() {
		once.Do(func() {
			connectionLocks.Lock()
			delete(connectionLocks.held, id)
			connectionLocks.Unlock()
		})
	}, true
}

// InstanceBusy reports whether an operation holds the instance lock.
func InstanceBusy(id string) bool {
	connectionLocks.Lock()
	defer connectionLocks.Unlock()
	return connectionLocks.held[id]
}
