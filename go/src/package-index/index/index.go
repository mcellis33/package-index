package index

import "sync"

type Index interface {
	// Returns true if the package could be indexed or if it was already
	// present. Returns false if the package cannot be indexed because some of
	// its dependencies aren't indexed yet and need to be installed first.
	//
	// NB: empty string is a valid (albeit silly) package name, even though
	// the frontend protocol does not support it.
	Index(pkg string, deps map[string]struct{}) (ok bool)
	// Returns true if the package could be removed from the index. Returns
	// false if the package could not be removed from the index because some
	// other indexed package depends on it. It returns true if the package
	// wasn't indexed.
	Remove(pkg string) (ok bool)
	// Returns true if the package is indexed. Returns false if the package
	// isn't indexed.
	Query(pkg string) (ok bool)
}

type index struct {
	l sync.RWMutex
	m map[string]entry
}

// TODO: entry is not a flat struct, as it holds a map. This representation
// will fragment the heap. We could make entry flat, but we would need to put
// bounds on deps.
type entry struct {
	refCount int64
	deps     map[string]struct{}
}

func NewIndex() Index {
	return &index{
		m: make(map[string]entry),
	}
}

func (i *index) Index(pkg string, deps map[string]struct{}) bool {
	i.l.Lock()
	defer i.l.Unlock()
	if _, ok := i.m[pkg]; ok {
		return true
	}
	// Don't hold references to empty deps.
	if len(deps) == 0 {
		deps = nil
	} else {
		for d := range deps {
			if _, ok := i.m[d]; !ok {
				return false
			}
		}
		for d := range deps {
			depEntry := i.m[d]
			depEntry.refCount++
			i.m[d] = depEntry
		}
	}
	i.m[pkg] = entry{deps: deps}
	return true
}

func (i *index) Remove(pkg string) bool {
	i.l.Lock()
	defer i.l.Unlock()
	entry, ok := i.m[pkg]
	if !ok {
		return true
	}
	if entry.refCount > 0 {
		return false
	}
	delete(i.m, pkg)
	for d := range entry.deps {
		depEntry := i.m[d]
		depEntry.refCount--
		i.m[d] = depEntry
	}
	return true
}

func (i *index) Query(pkg string) bool {
	i.l.RLock()
	defer i.l.RUnlock()
	_, ok := i.m[pkg]
	return ok
}
