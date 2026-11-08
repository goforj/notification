package notification

import "sync"

// MemoryJournal arbitrates scoped IDs within this process only. It retains entries
// until forgotten and must not be used as a durable queue journal.
type MemoryJournal struct {
	mu      sync.Mutex
	entries map[string]*entry
}

// entry pins the original plan, route and adapter revisions across resubmissions.
type entry struct {
	fingerprint string
	busy        bool
	plan        Plan
	planned     bool
	route       Route
	revisions   map[string]string
	result      Result
	err         error
	complete    bool
}

// NewMemoryJournal creates empty process-local deduplication state.
func NewMemoryJournal() *MemoryJournal { return &MemoryJournal{entries: map[string]*entry{}} }

// Forget removes a completed scoped identity, explicitly ending its deduplication window.
// Active entries cannot be forgotten. Forget permits a future send to repeat delivery.
func (j *MemoryJournal) Forget(scope Scope, id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	key := identity(scope.Application, scope.Tenant, id)
	if e := j.entries[key]; e != nil && e.busy {
		return ErrInProgress
	}
	delete(j.entries, key)
	return nil
}

// reserve arbitrates duplicate submissions before planner or driver invocation.
func (j *MemoryJournal) reserve(scope Scope, r Request) (*entry, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	key := identity(scope.Application, scope.Tenant, r.ID)
	fp := fingerprint(scope, r)
	if e := j.entries[key]; e != nil {
		if e.fingerprint != fp {
			return nil, ErrConflict
		}
		if e.busy {
			return nil, ErrInProgress
		}
		e.busy = true
		return e, nil
	}
	e := &entry{fingerprint: fp, busy: true}
	j.entries[key] = e
	return e, nil
}

// release publishes a call's snapshots through the same mutex used by reserve.
func (j *MemoryJournal) release(e *entry) { j.mu.Lock(); defer j.mu.Unlock(); e.busy = false }
