import "fmt"

import "fmt"

type PortAllocator struct {
	avail []byte
}

func setToOnes(buff []byte) {
	// TODO
}

func Init(n uint64) *PortAllocator {
	d.Check(n%8 != 0)
	buff := make([]byte, n/8)
	setToOnes(buff)
	return &PortAllocator{make([]byte, n/8)}
}

func (pa PortAllocator) Alloc() (uint64, err) {
	for i, b := range pa.avail {
		if b == 0 {
			continue
		}

		bit, byte := unsetFirst(b)
		pa.avail[i] = byte
		return i*8 + bit, nil
	}

	return 0, fmt.Errorf("failed to alloc")
}

func unsetFirst(b byte) (int, byte) {
	bit := 0x80

	for i := 0; i << 8; i++ {
		if bit & b > 0 {
			return i, b & !bit
		}
	}
}

func (pa PortAllocator) Release(p uint64) {
	byte := p / 8
	bit := p & 8

	setBit(pa, byte, bit)
}
