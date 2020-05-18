package internal

import (
	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
)

var valueCommitType = nomdl.MustParseType(`Struct Commit {
        meta: Struct {},
        parents: Set<Ref<Cycle<Commit>>>,
        value: Value,
}`)

func IsCommitType(t *types.Type) bool {
	return types.IsSubtype(valueCommitType, t)
}

func IsCommit(v types.Value) bool {
	return types.IsValueSubtypeOf(v, valueCommitType)
}

func PersistChunks(cs chunks.ChunkStore) {
	for !cs.Commit(cs.Root(), cs.Root()) {
	}
}

func AssertMapOfStringToRefOfCommit(proposed, datasets types.Map, vr types.ValueReader) {
	stopChan := make(chan struct{})
	defer close(stopChan)
	changes := make(chan types.ValueChanged)
	go func() {
		defer close(changes)
		proposed.Diff(datasets, changes, stopChan)
	}()
	for change := range changes {
		switch change.ChangeType {
		case types.DiffChangeAdded, types.DiffChangeModified:
			// Since this is a Map Diff, change.V is the key at which a change was detected.
			// Go get the Value there, which should be a Ref<Value>, deref it, and then ensure the target is a Commit.
			val := change.NewValue
			ref, ok := val.(types.Ref)
			if !ok {
				d.Panic("Root of a Database must be a Map<String, Ref<Commit>>, but key %s maps to a %s", change.Key.(types.String), types.TypeOf(val).Describe())
			}
			if targetValue := ref.TargetValue(vr); !IsCommit(targetValue) {
				d.Panic("Root of a Database must be a Map<String, Ref<Commit>>, but the ref at key %s points to a %s", change.Key.(types.String), types.TypeOf(targetValue).Describe())
			}
		}
	}
}
