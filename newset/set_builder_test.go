package newset

import (
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/assert"
	"github.com/attic-labs/noms/ref"
)

// A Chunker that always produces chunks of the same size.
type identicalSizeChunker struct {
	chunkSize int
	cursor    int
}

func newIdenticalSizeChunker(chunkSize int) *identicalSizeChunker {
	return &identicalSizeChunker{chunkSize: chunkSize}
}

func (chunker *identicalSizeChunker) Add(r ref.Ref) bool {
	if chunker.cursor == chunker.chunkSize-1 {
		chunker.cursor = 0
		return true
	}
	chunker.cursor++
	return false
}

func (chunker *identicalSizeChunker) New() Chunker {
	return newIdenticalSizeChunker(chunker.chunkSize)
}

func TestIdenticalSizeChunker(t *testing.T) {
	assert := assert.New(t)
	r := ref.Ref{}

	chunker := newIdenticalSizeChunker(1)
	assert.True(chunker.Add(r))
	assert.True(chunker.Add(r))

	chunker = newIdenticalSizeChunker(3)
	assert.False(chunker.Add(r))
	assert.False(chunker.Add(r))
	assert.True(chunker.Add(r))
	assert.False(chunker.Add(r))
	assert.False(chunker.Add(r))
	assert.True(chunker.Add(r))
}

func TestChunkSize3Depth4(t *testing.T) {
	assert := assert.New(t)

	// Define the chunk size as 3 items, and aim for a tree with 4 layers of chunking (root, 3 chunk layers, leaves), implying we need to add 3^4 = 81 items.
	chunkSize := 3

	sb := NewSetBuilderWithChunker(newIdenticalSizeChunker(chunkSize))
	refs, ator := addRefs(&sb, 81)
	set := sb.Build()

	for _, r := range refs {
		assert.True(set.Has(r))
	}
	assert.False(set.Has(ator.Next()))

	// Test each level by hand. This could be factored nicely, but nice factoring might have bugs. It's clear this way we're testing the right thing.

	// Top layer is the first chunked layer.
	first := set.(chunkedSet)
	assert.Equal(uint64(81), first.Len())
	assert.Equal(refs[0], first.First())
	assert.Equal(chunkSize, len(first.children))

	for i := 0; i < chunkSize; i++ {
		// Second chunked layer:
		second := first.children[i].set.(chunkedSet)
		assert.Equal(uint64(27), second.Len())
		assert.Equal(chunkSize, len(second.children))
		assert.Equal(refs[27*i], second.First())

		for j := 0; j < chunkSize; j++ {
			// Third chunked layer:
			third := second.children[j].set.(chunkedSet)
			assert.Equal(uint64(9), third.Len())
			assert.Equal(chunkSize, len(third.children))
			assert.Equal(refs[27*i+9*j], third.First())

			for k := 0; k < chunkSize; k++ {
				// Fourth layer are the leaf nodes.
				fourth := third.children[k].set.(flatSet)
				assert.Equal(uint64(3), fourth.Len())
				assert.Equal(chunkSize, len(fourth.d))
				assert.Equal(refs[27*i+9*j+3*k], fourth.First())
				// Lastly, check the individual values of the leaf nodes.
				for m := 0; m < chunkSize; m++ {
					assert.Equal(refs[27*i+9*j+3*k+m], fourth.d[m])
				}
			}
		}
	}
}

func TestRealData(t *testing.T) {
	assert := assert.New(t)

	sb := NewSetBuilder()
	refs, ator := addRefs(&sb, 5000)
	set := sb.Build()

	for _, r := range refs {
		assert.True(set.Has(r))
	}
	assert.False(set.Has(ator.Next()))

	expected := "(chunked with 85 chunks)\nchunk 0 (start 00000000)\n    flat{00000000...(10 more)...0000000b}\nchunk 1 (start 0000000c)\n    flat{0000000c...(185 more)...000000c6}\nchunk 2 (start 000000c7)\n    flat{000000c7...(11 more)...000000d3}\nchunk 3 (start 000000d4)\n    flat{000000d4...(34 more)...000000f7}\nchunk 4 (start 000000f8)\n    flat{000000f8...(37 more)...0000011e}\nchunk 5 (start 0000011f)\n    flat{0000011f...(19 more)...00000133}\nchunk 6 (start 00000134)\n    flat{00000134...(15 more)...00000144}\nchunk 7 (start 00000145)\n    flat{00000145...(86 more)...0000019c}\nchunk 8 (start 0000019d)\n    flat{0000019d...(179 more)...00000251}\nchunk 9 (start 00000252)\n    flat{00000252...(63 more)...00000292}\nchunk 10 (start 00000293)\n    flat{00000293...(102 more)...000002fa}\nchunk 11 (start 000002fb)\n    flat{000002fb...(30 more)...0000031a}\nchunk 12 (start 0000031b)\n    flat{0000031b...(5 more)...00000321}\nchunk 13 (start 00000322)\n    flat{00000322...(32 more)...00000343}\nchunk 14 (start 00000344)\n    flat{00000344...(143 more)...000003d4}\nchunk 15 (start 000003d5)\n    flat{000003d5...(16 more)...000003e6}\nchunk 16 (start 000003e7)\n    flat{000003e7...(12 more)...000003f4}\nchunk 17 (start 000003f5)\n    flat{000003f5...(184 more)...000004ae}\nchunk 18 (start 000004af)\n    flat{000004af...(11 more)...000004bb}\nchunk 19 (start 000004bc)\n    flat{000004bc...(9 more)...000004c6}\nchunk 20 (start 000004c7)\n    flat{000004c7...(48 more)...000004f8}\nchunk 21 (start 000004f9)\n    flat{000004f9...(52 more)...0000052e}\nchunk 22 (start 0000052f)\n    flat{0000052f...(20 more)...00000544}\nchunk 23 (start 00000545)\n    flat{00000545...(25 more)...0000055f}\nchunk 24 (start 00000560)\n    flat{00000560...(13 more)...0000056e}\nchunk 25 (start 0000056f)\n    flat{0000056f...(65 more)...000005b1}\nchunk 26 (start 000005b2)\n    flat{000005b2...(34 more)...000005d5}\nchunk 27 (start 000005d6)\n    flat{000005d6...(81 more)...00000628}\nchunk 28 (start 00000629)\n    flat{00000629...(81 more)...0000067b}\nchunk 29 (start 0000067c)\n    flat{0000067c...(34 more)...0000069f}\nchunk 30 (start 000006a0)\n    flat{000006a0...(43 more)...000006cc}\nchunk 31 (start 000006cd)\n    flat{000006cd...(38 more)...000006f4}\nchunk 32 (start 000006f5)\n    flat{000006f5...(15 more)...00000705}\nchunk 33 (start 00000706)\n    flat{00000706...(38 more)...0000072d}\nchunk 34 (start 0000072e)\n    flat{0000072e...(64 more)...0000076f}\nchunk 35 (start 00000770)\n    flat{00000770...(0 more)...00000771}\nchunk 36 (start 00000772)\n    flat{00000772...(50 more)...000007a5}\nchunk 37 (start 000007a6)\n    flat{000007a6...(26 more)...000007c1}\nchunk 38 (start 000007c2)\n    flat{000007c2...(23 more)...000007da}\nchunk 39 (start 000007db)\n    flat{000007db...(198 more)...000008a2}\nchunk 40 (start 000008a3)\n    flat{000008a3...(16 more)...000008b4}\nchunk 41 (start 000008b5)\n    flat{000008b5...(18 more)...000008c8}\nchunk 42 (start 000008c9)\n    flat{000008c9...(61 more)...00000907}\nchunk 43 (start 00000908)\n    flat{00000908...(49 more)...0000093a}\nchunk 44 (start 0000093b)\n    flat{0000093b...(100 more)...000009a0}\nchunk 45 (start 000009a1)\n    flat{000009a1...(24 more)...000009ba}\nchunk 46 (start 000009bb)\n    flat{000009bb...(38 more)...000009e2}\nchunk 47 (start 000009e3)\n    flat{000009e3...(30 more)...00000a02}\nchunk 48 (start 00000a03)\n    flat{00000a03...(139 more)...00000a8f}\nchunk 49 (start 00000a90)\n    flat{00000a90...(4 more)...00000a95}\nchunk 50 (start 00000a96)\n    flat{00000a96...(1 more)...00000a98}\nchunk 51 (start 00000a99)\n    flat{00000a99...(228 more)...00000b7e}\nchunk 52 (start 00000b7f)\n    flat{00000b7f...(12 more)...00000b8c}\nchunk 53 (start 00000b8d)\n    flat{00000b8d...(33 more)...00000baf}\nchunk 54 (start 00000bb0)\n    flat{00000bb0...(75 more)...00000bfc}\nchunk 55 (start 00000bfd)\n    flat{00000bfd...(109 more)...00000c6b}\nchunk 56 (start 00000c6c)\n    flat{00000c6c...(28 more)...00000c89}\nchunk 57 (start 00000c8a)\n    flat{00000c8a...(0 more)...00000c8b}\nchunk 58 (start 00000c8c)\n    flat{00000c8c...(37 more)...00000cb2}\nchunk 59 (start 00000cb3)\n    flat{00000cb3...(219 more)...00000d8f}\nchunk 60 (start 00000d90)\n    flat{00000d90...(11 more)...00000d9c}\nchunk 61 (start 00000d9d)\n    flat{00000d9d...(18 more)...00000db0}\nchunk 62 (start 00000db1)\n    flat{00000db1...(84 more)...00000e06}\nchunk 63 (start 00000e07)\n    flat{00000e07...(14 more)...00000e16}\nchunk 64 (start 00000e17)\n    flat{00000e17...(41 more)...00000e41}\nchunk 65 (start 00000e42)\n    flat{00000e42...(2 more)...00000e45}\nchunk 66 (start 00000e46)\n    flat{00000e46...(23 more)...00000e5e}\nchunk 67 (start 00000e5f)\n    flat{00000e5f...(25 more)...00000e79}\nchunk 68 (start 00000e7a)\n    flat{00000e7a...(22 more)...00000e91}\nchunk 69 (start 00000e92)\n    flat{00000e92...(12 more)...00000e9f}\nchunk 70 (start 00000ea0)\n    flat{00000ea0...(59 more)...00000edc}\nchunk 71 (start 00000edd)\n    flat{00000edd...(90 more)...00000f38}\nchunk 72 (start 00000f39)\n    flat{00000f39...(274 more)...0000104c}\nchunk 73 (start 0000104d)\n    flat{0000104d...(15 more)...0000105d}\nchunk 74 (start 0000105e)\n    flat{0000105e...(52 more)...00001093}\nchunk 75 (start 00001094)\n    flat{00001094...(87 more)...000010ec}\nchunk 76 (start 000010ed)\n    flat{000010ed...(69 more)...00001133}\nchunk 77 (start 00001134)\n    flat{00001134...(371 more)...000012a8}\nchunk 78 (start 000012a9)\n    flat{000012a9...(44 more)...000012d6}\nchunk 79 (start 000012d7)\n    flat{000012d7...(13 more)...000012e5}\nchunk 80 (start 000012e6)\n    flat{000012e6...(10 more)...000012f1}\nchunk 81 (start 000012f2)\n    flat{000012f2...(30 more)...00001311}\nchunk 82 (start 00001312)\n    flat 00001312\nchunk 83 (start 00001313)\n    flat{00001313...(49 more)...00001345}\nchunk 84 (start 00001346)\n    flat{00001346...(64 more)...00001387}\n"
	assert.Equal(expected, set.Fmt(0))
}

// Add n ref items to a set builder, and return the refs that were added alongside the referrator used to generate them.
func addRefs(sb *SetBuilder, n int) ([]ref.Ref, referrator) {
	var refs []ref.Ref
	ator := newReferrator()
	for i := 0; i < n; i++ {
		ref := ator.Next()
		(*sb).AddItem(ref)
		refs = append(refs, ref)
	}
	return refs, ator
}
