package types

type mapLeafSequence struct {
	data []mapEntry // sorted by entry.key.Ref()
	t    *Type
	vr   ValueReader
}

type mapEntry struct {
	key   Value
	value Value
}

func newMapLeafSequence(t *Type, vr ValueReader, data ...mapEntry) orderedSequence {
	return mapLeafSequence{data, t, vr}
}

// sequence interface
func (ml mapLeafSequence) getItem(idx int) sequenceItem {
	return ml.data[idx]
}

func (ml mapLeafSequence) seqLen() int {
	return len(ml.data)
}

func (ml mapLeafSequence) numLeaves() uint64 {
	return uint64(len(ml.data))
}

func (ml mapLeafSequence) valueReader() ValueReader {
	return ml.vr
}

func (ml mapLeafSequence) Chunks() (chunks []Ref) {
	for _, entry := range ml.data {
		chunks = append(chunks, entry.key.Chunks()...)
		chunks = append(chunks, entry.value.Chunks()...)
	}
	return
}

func (ml mapLeafSequence) Type() *Type {
	return ml.t
}

// orderedSequence interface
func (ml mapLeafSequence) getKey(idx int) Value {
	return ml.data[idx].key
}

func (ml mapLeafSequence) equalsAt(idx int, other interface{}) bool {
	entry := ml.data[idx]
	otherEntry := other.(mapEntry)
	return entry.key.Equals(otherEntry.key) && entry.value.Equals(otherEntry.value)
}

// Collection interface
func (ml mapLeafSequence) Len() uint64 {
	return uint64(len(ml.data))
}

func (ml mapLeafSequence) Empty() bool {
	return ml.Len() == uint64(0)
}
