package value

import (
	"encoding/base64"
	"fmt"
	"io"
	"strconv"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/types"
)

type valueWriter struct {
	ind int
	w   io.Writer
}

func (w *valueWriter) write(s string) {
	n, err := io.WriteString(w.w, s)
	d.Chk.NoError(err)
	d.Chk.Equal(len(s), n)
}

func (w *valueWriter) indent() {
	w.ind++
}

func (w *valueWriter) outdent() {
	w.ind--
}

func (w *valueWriter) newLine() {
	w.write("\n")
	for i := 0; i < w.ind; i++ {
		w.write("  ")
	}
}

func (w *valueWriter) Write(v types.Value) {
	switch v.Type().Kind() {
	case types.BoolKind:
		w.write(strconv.FormatBool(bool(v.(types.Bool))))
	case types.Uint8Kind:
		w.write(strconv.FormatUint(uint64(v.(types.Uint8)), 10))
	case types.Uint16Kind:
		w.write(strconv.FormatUint(uint64(v.(types.Uint16)), 10))
	case types.Uint32Kind:
		w.write(strconv.FormatUint(uint64(v.(types.Uint32)), 10))
	case types.Uint64Kind:
		w.write(strconv.FormatUint(uint64(v.(types.Uint64)), 10))
	case types.Int8Kind:
		w.write(strconv.FormatInt(int64(v.(types.Int8)), 10))
	case types.Int16Kind:
		w.write(strconv.FormatInt(int64(v.(types.Int16)), 10))
	case types.Int32Kind:
		w.write(strconv.FormatInt(int64(v.(types.Int32)), 10))
	case types.Int64Kind:
		w.write(strconv.FormatInt(int64(v.(types.Int64)), 10))
	case types.Float32Kind:
		w.write(strconv.FormatFloat(float64(v.(types.Float32)), 'g', -1, 32))
	case types.Float64Kind:
		w.write(strconv.FormatFloat(float64(v.(types.Float64)), 'g', -1, 64))

	case types.StringKind:
		w.write(strconv.Quote(v.(types.String).String()))

	case types.BlobKind:
		blob := v.(types.Blob)
		encoder := base64.NewEncoder(base64.StdEncoding, w.w)
		_, err := io.Copy(encoder, blob.Reader())
		d.Chk.NoError(err)
		encoder.Close()

	case types.ListKind:
		w.write("[")
		v.(types.List).IterAll(func(v types.Value, i uint64) {
			if i > 0 {
				w.write(", ")
			}
			w.Write(v)
		})
		w.write("]")

	case types.MapKind:
		w.write("{")
		i := uint64(0)
		v.(types.Map).IterAll(func(key, val types.Value) {
			if i > 0 {
				w.write(", ")
			}
			w.Write(key)
			w.write(": ")
			w.Write(val)
			i++
		})
		w.write("}")

	case types.RefKind:
		w.write(v.(types.RefBase).TargetRef().String())

	case types.SetKind:
		w.write("{")
		i := uint64(0)
		v.(types.Set).IterAll(func(v types.Value) {
			if i > 0 {
				w.write(", ")
			}
			w.Write(v)
			i++
		})
		w.write("}")

	case types.TypeKind:
		w.writeTypeAsValue(v.(types.Type))

	case types.UnresolvedKind:
		w.writeUnresolved(v, true)

	case types.PackageKind:
		panic("not implemented")

	case types.ValueKind, types.EnumKind, types.StructKind:
		panic("unreachable")
	}
}

func (w *valueWriter) writeUnresolved(v types.Value, printStructName bool) {
	t := v.Type()
	pkg := types.LookupPackage(t.PackageRef())
	typeDef := pkg.Types()[t.Ordinal()]
	switch typeDef.Kind() {
	case types.StructKind:
		v := v.(types.Struct)
		desc := typeDef.Desc.(types.StructDesc)
		i := 0
		if printStructName {
			w.write(typeDef.Name())
			w.write(" ")
		}
		w.write("{")

		writeField := func(f types.Field, v types.Value) {
			if i > 0 {
				w.write(", ")
			}
			w.write(f.Name)
			w.write(": ")
			w.Write(v)
			i++
		}

		for _, f := range desc.Fields {
			if fv, present := v.MaybeGet(f.Name); present {
				writeField(f, fv)
			}
		}
		if len(desc.Union) > 0 {
			f := desc.Union[v.UnionIndex()]
			fv := v.UnionValue()
			writeField(f, fv)
		}

		w.write("}")

	case types.EnumKind:
		v := v.(types.Enum)
		i := types.EnumPrimitiveValueFromType(v, t)
		w.write(typeDef.Desc.(types.EnumDesc).IDs[i])

	default:
		panic("unreachable")
	}
}

func (w *valueWriter) WriteTagged(v types.Value) {
	t := v.Type()
	switch t.Kind() {
	case types.BoolKind, types.StringKind:
		w.Write(v)
	case types.Uint8Kind, types.Uint16Kind, types.Uint32Kind, types.Uint64Kind, types.Int8Kind, types.Int16Kind, types.Int32Kind, types.Int64Kind, types.Float32Kind, types.Float64Kind, types.BlobKind, types.ListKind, types.MapKind, types.RefKind, types.SetKind, types.TypeKind:
		w.writeTypeAsValue(t)
		w.write("(")
		w.Write(v)
		w.write(")")

	case types.UnresolvedKind:
		w.writeTypeAsValue(t)
		w.write("(")
		w.writeUnresolved(v, false)
		w.write(")")
	case types.PackageKind:
		panic("not implemented")

	case types.ValueKind, types.EnumKind, types.StructKind:
	default:
		panic("unreachable")
	}
}

func (w *valueWriter) writeTypeAsValue(t types.Type) {
	switch t.Kind() {
	case types.BlobKind, types.BoolKind, types.Float32Kind, types.Float64Kind, types.Int16Kind, types.Int32Kind, types.Int64Kind, types.Int8Kind, types.StringKind, types.TypeKind, types.Uint16Kind, types.Uint32Kind, types.Uint64Kind, types.Uint8Kind, types.ValueKind:
		w.write(types.KindToString[t.Kind()])
	case types.ListKind, types.RefKind, types.SetKind:
		w.write(types.KindToString[t.Kind()])
		w.write("<")
		w.writeTypeAsValue(t.Desc.(types.CompoundDesc).ElemTypes[0])
		w.write(">")
	case types.MapKind:
		w.write(types.KindToString[t.Kind()])
		w.write("<")
		w.writeTypeAsValue(t.Desc.(types.CompoundDesc).ElemTypes[0])
		w.write(", ")
		w.writeTypeAsValue(t.Desc.(types.CompoundDesc).ElemTypes[1])
		w.write(">")
	case types.EnumKind:
		w.write("enum ")
		w.write(t.Name())
		w.write(" {")
		for i, id := range t.Desc.(types.EnumDesc).IDs {
			if i > 0 {
				w.write(" ")
			}
			w.write(id)
		}
		w.write("}")
	case types.StructKind:
		w.write("struct ")
		w.write(t.Name())
		w.write(" {")
		desc := t.Desc.(types.StructDesc)
		writeField := func(f types.Field, i int) {
			if i > 0 {
				w.write(" ")
			}
			w.write(f.Name)
			w.write(": ")
			if f.Optional {
				w.write("optional ")
			}
			w.writeTypeAsValue(f.T)
		}
		for i, f := range desc.Fields {
			writeField(f, i)
		}
		if len(desc.Union) > 0 {
			w.write(" union {")
			for i, f := range desc.Union {
				writeField(f, i)
			}
			w.write("}")
		}
		w.write("}")
	case types.UnresolvedKind:
		w.writeUnresolvedTypeRef(t, true)
	case types.PackageKind:
		panic("not implemented")
	}
}

func (w *valueWriter) writeUnresolvedTypeRef(t types.Type, printStructName bool) {
	pkg := types.LookupPackage(t.PackageRef())
	typeDef := pkg.Types()[t.Ordinal()]
	switch typeDef.Kind() {
	case types.StructKind:
		w.write("Struct")
	case types.EnumKind:
		w.write("Enum")
	default:
		panic("unreachable")
	}
	fmt.Fprintf(w.w, "<%s, %s, %d>", typeDef.Name(), t.PackageRef(), t.Ordinal())
}
