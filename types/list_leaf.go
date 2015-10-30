package types

import (
	"runtime"
	"sync"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

type listLeaf struct {
	values []Value
	t      TypeRef
	ref    *ref.Ref
}

func newListLeaf(v ...Value) List {
	// Copy because Noms values are supposed to be immutable and Go allows v to be reused (thus mutable).
	values := make([]Value, len(v))
	copy(values, v)
	return newListLeafNoCopy(values, listTypeRef)
}

func newListLeafNoCopy(values []Value, t TypeRef) List {
	d.Chk.Equal(ListKind, t.Kind())
	return listLeaf{values, t, &ref.Ref{}}
}

func (l listLeaf) Len() uint64 {
	return uint64(len(l.values))
}

func (l listLeaf) Empty() bool {
	return l.Len() == uint64(0)
}

func (l listLeaf) Get(idx uint64) Value {
	return l.values[idx]
}

func (l listLeaf) Iter(f listIterFunc) {
	for i, v := range l.values {
		if f(v, uint64(i)) {
			break
		}
	}
}

func (l listLeaf) IterAll(f listIterAllFunc) {
	for i, v := range l.values {
		f(v, uint64(i))
	}
}

func (l listLeaf) IterAllP(concurrency int, f listIterAllFunc) {
	var limit chan int
	if concurrency == 0 {
		limit = make(chan int, runtime.NumCPU())
	} else {
		limit = make(chan int, concurrency)
	}

	l.iterInternal(limit, f, 0)
}

func (l listLeaf) iterInternal(sem chan int, lf listIterAllFunc, offset uint64) {
	wg := sync.WaitGroup{}

	for idx := uint64(0); idx < l.Len(); idx++ {
		wg.Add(1)

		sem <- 1
		go func(idx uint64) {
			defer wg.Done()
			v := l.values[idx]
			lf(v, idx+offset)
			<-sem
		}(idx)
	}

	wg.Wait()
}

func (l listLeaf) Map(mf MapFunc) []interface{} {
	return l.MapP(1, mf)
}

func (l listLeaf) MapP(concurrency int, mf MapFunc) []interface{} {
	var limit chan int
	if concurrency == 0 {
		limit = make(chan int, runtime.NumCPU())
	} else {
		limit = make(chan int, concurrency)
	}

	return l.mapInternal(limit, mf, 0)
}

func (l listLeaf) mapInternal(sem chan int, mf MapFunc, offset uint64) []interface{} {
	values := make([]interface{}, l.Len(), l.Len())
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for idx := uint64(0); idx < l.Len(); idx++ {
		wg.Add(1)

		sem <- 1
		go func(idx uint64) {
			defer wg.Done()
			v := l.values[idx]
			c := mf(v, idx+offset)
			<-sem
			mu.Lock()
			values[idx] = c
			mu.Unlock()
		}(idx)
	}

	wg.Wait()
	return values
}

func (l listLeaf) Slice(start uint64, end uint64) List {
	return newListLeafNoCopy(l.values[start:end], l.t)
}

func (l listLeaf) Set(idx uint64, v Value) List {
	values := make([]Value, len(l.values))
	copy(values, l.values)
	values[idx] = v
	return newListLeafNoCopy(values, l.t)
}

func (l listLeaf) Append(v ...Value) List {
	values := append(l.values, v...)
	return newListLeafNoCopy(values, l.t)
}

func (l listLeaf) Insert(idx uint64, v ...Value) List {
	values := make([]Value, len(l.values)+len(v))
	copy(values, l.values[:idx])
	copy(values[idx:], v)
	copy(values[idx+uint64(len(v)):], l.values[idx:])
	return newListLeafNoCopy(values, l.t)
}

func (l listLeaf) Remove(start uint64, end uint64) List {
	values := make([]Value, uint64(len(l.values))-(end-start))
	copy(values, l.values[:start])
	copy(values[start:], l.values[end:])
	return newListLeafNoCopy(values, l.t)
}

func (l listLeaf) RemoveAt(idx uint64) List {
	return l.Remove(idx, idx+1)
}

func (l listLeaf) Ref() ref.Ref {
	return EnsureRef(l.ref, l)
}

// BUG 141
func (l listLeaf) Release() {
	// TODO: Remove?
}

func (l listLeaf) Equals(other Value) bool {
	return other != nil && l.Ref() == other.Ref()
}

func (l listLeaf) Chunks() (chunks []ref.Ref) {
	for _, v := range l.values {
		chunks = appendValueToChunks(chunks, v)
	}
	return
}

func (l listLeaf) TypeRef() TypeRef {
	return l.t
}
