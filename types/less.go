package types

func valueLess(v1, v2 Value) bool {
	switch v2.Type().Kind() {
	case BoolKind, NumberKind, StringKind:
		return false
	default:
		return v1.Hash().Less(v2.Hash())
	}
}
