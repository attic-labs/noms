package newset

import (
	"github.com/attic-labs/noms/ref"
)

type Set interface {
	First() ref.Ref
	Len() int
	Has(r ref.Ref) bool
	Ref() ref.Ref
	Fmt(indent int) string
}
