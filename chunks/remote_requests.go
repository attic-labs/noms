package chunks

import "github.com/attic-labs/noms/hash"

type ReadRequest interface {
	Hash() hash.Hash
	Outstanding() OutstandingRequest
}

func NewGetRequest(r hash.Hash, ch chan Chunk) GetRequest {
	return GetRequest{r, ch}
}

type GetRequest struct {
	r  hash.Hash
	ch chan Chunk
}

func NewHasRequest(r hash.Hash, ch chan bool) HasRequest {
	return HasRequest{r, ch}
}

type HasRequest struct {
	r  hash.Hash
	ch chan bool
}

func (g GetRequest) Hash() hash.Hash {
	return g.r
}

func (g GetRequest) Outstanding() OutstandingRequest {
	return OutstandingGet(g.ch)
}

func (h HasRequest) Hash() hash.Hash {
	return h.r
}

func (h HasRequest) Outstanding() OutstandingRequest {
	return OutstandingHas(h.ch)
}

type OutstandingRequest interface {
	Satisfy(c Chunk)
	Fail()
}

type OutstandingGet chan Chunk
type OutstandingHas chan bool

func (r OutstandingGet) Satisfy(c Chunk) {
	r <- c
	close(r)
}

func (r OutstandingGet) Fail() {
	r <- EmptyChunk
	close(r)
}

func (h OutstandingHas) Satisfy(c Chunk) {
	h <- true
	close(h)
}

func (h OutstandingHas) Fail() {
	h <- false
	close(h)
}

// ReadBatch represents a set of queued Get/Has requests, each of which are blocking on a receive channel for a response.
type ReadBatch map[hash.Hash][]OutstandingRequest

// GetBatch represents a set of queued Get requests, each of which are blocking on a receive channel for a response.
type GetBatch map[hash.Hash][]chan Chunk
type HasBatch map[hash.Hash][]chan bool

// Close ensures that callers to Get() and Has() are failed correctly if the corresponding chunk wasn't in the response from the server (i.e. it wasn't found).
func (rb *ReadBatch) Close() error {
	for _, reqs := range *rb {
		for _, req := range reqs {
			req.Fail()
		}
	}
	return nil
}

// Put is implemented so that ReadBatch implements the ChunkSink interface.
func (rb *ReadBatch) Put(c Chunk) {
	for _, or := range (*rb)[c.Hash()] {
		or.Satisfy(c)
	}

	delete(*rb, c.Hash())
}

// PutMany is implemented so that ReadBatch implements the ChunkSink interface.
func (rb *ReadBatch) PutMany(chunks []Chunk) (e BackpressureError) {
	for _, c := range chunks {
		rb.Put(c)
	}
	return
}

// Close ensures that callers to Get() must receive nil if the corresponding chunk wasn't in the response from the server (i.e. it wasn't found).
func (gb *GetBatch) Close() error {
	for _, chs := range *gb {
		for _, ch := range chs {
			ch <- EmptyChunk
		}
	}
	return nil
}

// Put is implemented so that GetBatch implements the ChunkSink interface.
func (gb *GetBatch) Put(c Chunk) {
	for _, ch := range (*gb)[c.Hash()] {
		ch <- c
	}

	delete(*gb, c.Hash())
}

// PutMany is implemented so that GetBatch implements the ChunkSink interface.
func (gb *GetBatch) PutMany(chunks []Chunk) (e BackpressureError) {
	for _, c := range chunks {
		gb.Put(c)
	}
	return
}
