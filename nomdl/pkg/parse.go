package pkg

import (
	"io"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/types"
)

// Parsed represents a parsed Noms type package, which has some additional metadata beyond that which is present in a types.Package.
type Parsed struct {
	Filename   string
	Types      []*types.Type
	AliasNames map[string]string
}

// ParseNomDL reads a Noms package specification from r and returns a Package. Errors will be annotated with packageName and thrown.
func ParseNomDL(filename string, r io.Reader, includePath string) []*types.Type {
	i := runParser(filename, r)
	i.Filename = filename
	// name -> Parsed
	imports := resolveImports(i.Aliases, includePath)

	// Replace all variable references with the actual type.
	resolveReferences(&i, imports)

	return i.Types
}

type intermediate struct {
	Filename string
	// Aliases maps from Name to Target, where target is the non resolved filename.
	Aliases map[string]string
	Types   []*types.Type
}

func runParser(filename string, r io.Reader) intermediate {
	got, err := ParseReader(filename, r)
	d.Exp.NoError(err)
	return got.(intermediate)
}

func indexOf(t *types.Type, ts []*types.Type) int16 {
	for i, tt := range ts {
		if tt.Name() == t.Name() {
			return int16(i)
		}
	}
	return -1
}

func findType(n string, ts []*types.Type) *types.Type {
	for _, t := range ts {
		if n == t.Name() {
			return t
		}
	}
	d.Exp.Fail("Undefined reference %s", n)
	return nil
}

// resolveReferences replaces references with the actual Type
func resolveReferences(i *intermediate, aliases map[string][]*types.Type) {
	var rec func(t *types.Type) *types.Type
	resolveFields := func(fields types.TypeMap) {
		for name, t := range fields {
			fields[name] = rec(t)
		}
	}
	rec = func(t *types.Type) *types.Type {
		switch t.Kind() {
		case UnresolvedKind:
			desc := t.Desc.(UnresolvedDesc)
			if desc.Namespace == "" {
				return findType(desc.Name, i.Types)
			}
			ts, ok := aliases[desc.Namespace]
			d.Exp.True(ok, "No such namespace: %s", desc.Namespace)
			return findType(desc.Name, ts)
		case types.ListKind:
			return types.MakeListType(rec(t.Desc.(types.CompoundDesc).ElemTypes[0]))
		case types.SetKind:
			return types.MakeSetType(rec(t.Desc.(types.CompoundDesc).ElemTypes[0]))
		case types.RefKind:
			return types.MakeRefType(rec(t.Desc.(types.CompoundDesc).ElemTypes[0]))
		case types.MapKind:
			elemTypes := t.Desc.(types.CompoundDesc).ElemTypes
			return types.MakeMapType(rec(elemTypes[0]), rec(elemTypes[1]))
		case types.StructKind:
			resolveFields(t.Desc.(types.StructDesc).Fields)
		}
		return t
	}

	for idx, t := range i.Types {
		i.Types[idx] = rec(t)
	}
}
