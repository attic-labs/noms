package types

import (
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"math/big"
)

type Number struct {
	n *big.Float
}

func NewNumber(n interface{}) Number {
	switch t := n.(type) {
	case int:
		return Number{new(big.Float).SetInt64(int64(n.(int)))}
	case int64:
		return Number{new(big.Float).SetInt64(n.(int64))}
	case uint64:
		return Number{new(big.Float).SetUint64(n.(uint64))}
	case float64:
		return Number{big.NewFloat(n.(float64))}
	case *big.Float:
		return Number{new(big.Float).Set(n.(*big.Float))}
	default:
		d.Chk.Fail("unknown type in NewNumber - %T", t)
		return Number{new(big.Float).SetInf(true)}
	}
}

// Value interface
func (v Number) Equals(other Value) bool {
	if other, ok := other.(Number); ok {
		return 0 == v.n.Cmp(other.n)
	}
	return false
}

func (v Number) Less(other Value) bool {
	if v2, ok := other.(Number); ok {
		return -1 == v.n.Cmp(v2.n)
	}
	return NumberKind < other.Type().Kind()
}

func (v Number) Ref() ref.Ref {
	return getRef(v)
}

func (v Number) ChildValues() []Value {
	return nil
}

func (v Number) Chunks() []Ref {
	return nil
}

func (v Number) Type() *Type {
	return NumberType
}

func (v Number) ToUint64() uint64 {
	n, accuracy := v.n.Uint64()
	d.Chk.True(accuracy == big.Exact, "number conversion to uint64 not exact")
	return n
}

func (v Number) ToFloat64() float64 {
	n, accuracy := v.n.Float64()
	d.Chk.True(accuracy == big.Exact, "number conversion to float64 not exact")
	return n
}
