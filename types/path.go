package types

import (
	"fmt"
	"math"
	"strings"

	"github.com/attic-labs/noms/d"
)

type Path []pathPart

type pathPart interface {
	Resolve(v Value) Value
	String() string
}

func NewPath() Path {
	return Path{}
}

func (p Path) AddField(name string) Path {
	return append(p, newFieldPart(name))
}

func (p Path) AddIndex(idx Value) Path {
	return append(p, newIndexPart(idx))
}

func (p Path) Resolve(v Value) (resolved Value) {
	resolved = v
	for _, part := range p {
		if resolved == nil {
			break
		}
		resolved = part.Resolve(resolved)
	}

	return
}

func (p Path) String() string {
	strs := make([]string, len(p))
	for _, part := range p {
		strs = append(strs, part.String())
	}

	return strings.Join(strs, "")
}

type fieldPart struct {
	name string
}

func newFieldPart(name string) fieldPart {
	return fieldPart{name}
}

func (fp fieldPart) Resolve(v Value) Value {
	if s, ok := v.(Struct); ok {
		if fv, ok := s.data[fp.name]; ok {
			return fv
		}
	}

	return nil
}

func (fp fieldPart) String() string {
	return fmt.Sprintf(".%s", fp.name)
}

type indexPart struct {
	idx Value
}

func newIndexPart(idx Value) indexPart {
	k := idx.Type().Kind()
	d.Chk.True(k == StringKind || k == BoolKind || k == NumberKind)
	return indexPart{idx}
}

func (ip indexPart) Resolve(v Value) Value {
	if l, ok := v.(List); ok {
		if n, ok := ip.idx.(Number); ok {
			f := float64(n)
			if f == math.Trunc(f) && f >= 0 {
				u := uint64(f)
				if u < l.Len() {
					return l.Get(u)
				}
			}
		}
	}

	if m, ok := v.(Map); ok {
		return m.Get(ip.idx)
	}

	return nil
}

func (ip indexPart) String() string {
	return fmt.Sprintf("[%s]", EncodedValue(ip.idx))
}
