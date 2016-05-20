package types

type setLeafSequence struct {
	data []Value // sorted by Ref()
	t    *Type
	vr   ValueReader
}

func newSetLeafSequence(t *Type, vr ValueReader, m ...Value) orderedSequence {
	return setLeafSequence{m, t, vr}
}

// sequence interface
func (sl setLeafSequence) getItem(idx int) sequenceItem {
	return sl.data[idx]
}

func (sl setLeafSequence) seqLen() int {
	return len(sl.data)
}

func (sl setLeafSequence) numLeaves() uint64 {
	return uint64(len(sl.data))
}

func (sl setLeafSequence) valueReader() ValueReader {
	return sl.vr
}

func (sl setLeafSequence) Chunks() (chunks []Ref) {
	for _, v := range sl.data {
		chunks = append(chunks, v.Chunks()...)
	}
	return
}

func (sl setLeafSequence) Type() *Type {
	return sl.t
}

// orderedSequence interface
func (sl setLeafSequence) getKey(idx int) Value {
	return sl.data[idx]
}

func (sl setLeafSequence) equalsAt(idx int, other interface{}) bool {
	entry := sl.data[idx]
	return entry.Equals(other.(Value))
}
