package format

import (
	"context"
	"errors"
	"runtime"
)

// ParallelBatchCommits is the number of batch commits that can be in-flight before blocking.
// TODO(ipfs/go-ipfs#4299): Experiment with multiple datastores, storage
// devices, and CPUs to find the right value/formula.
var ParallelBatchCommits = runtime.NumCPU() * 2

// ErrNotCommited is returned when closing a batch that hasn't been successfully
// committed.
var ErrNotCommited = errors.New("error: batch not commited")

// ErrClosed is returned when operating on a batch that has already been closed.
var ErrClosed = errors.New("error: batch closed")

// NewBatch returns a node buffer (Batch) that buffers nodes internally and
// commits them to the underlying DAGService in batches. Use this if you intend
// to add or remove a lot of nodes all at once.
//
// If the passed context is canceled, any in-progress commits are aborted.
func NewBatch(ctx context.Context, ds DAGService) *Batch {
	ctx, cancel := context.WithCancel(ctx)
	return &Batch{
		ds:            ds,
		ctx:           ctx,
		cancel:        cancel,
		commitResults: make(chan error, ParallelBatchCommits),
		MaxSize:       8 << 20,

		// By default, only batch up to 128 nodes at a time.
		// The current implementation of flatfs opens this many file
		// descriptors at the same time for the optimized batch write.
		MaxNodes: 128,
	}
}

// Batch is a buffer for batching adds to a dag.
type Batch struct {
	ds DAGService

	ctx    context.Context
	cancel func()

	activeCommits int
	err           error
	commitResults chan error

	nodes []Node
	size  int

	MaxSize  int
	MaxNodes int
}

func (t *Batch) processResults() {
	for t.activeCommits > 0 {
		select {
		case err := <-t.commitResults:
			t.activeCommits--
			if err != nil {
				t.setError(err)
				return
			}
		default:
			return
		}
	}
}

func (t *Batch) asyncCommit() {
	numBlocks := len(t.nodes)
	if numBlocks == 0 {
		return
	}
	if t.activeCommits >= ParallelBatchCommits {
		select {
		case err := <-t.commitResults:
			t.activeCommits--

			if err != nil {
				t.setError(err)
				return
			}
		case <-t.ctx.Done():
			t.setError(t.ctx.Err())
			return
		}
	}
	go func(ctx context.Context, b []Node, result chan error, ds DAGService) {
		select {
		case result <- ds.AddMany(ctx, b):
		case <-ctx.Done():
		}
	}(t.ctx, t.nodes, t.commitResults, t.ds)

	t.activeCommits++
	t.nodes = make([]Node, 0, numBlocks)
	t.size = 0

	return
}

// Add adds a node to the batch and commits the batch if necessary.
func (t *Batch) Add(nd Node) error {
	if t.err != nil {
		return t.err
	}
	// Not strictly necessary but allows us to catch errors early.
	t.processResults()

	if t.err != nil {
		return t.err
	}

	t.nodes = append(t.nodes, nd)
	t.size += len(nd.RawData())

	if t.size > t.MaxSize || len(t.nodes) > t.MaxNodes {
		t.asyncCommit()
	}
	return t.err
}

// Commit commits batched nodes.
func (t *Batch) Commit() error {
	if t.err != nil {
		return t.err
	}

	t.asyncCommit()

loop:
	for t.activeCommits > 0 {
		select {
		case err := <-t.commitResults:
			t.activeCommits--
			if err != nil {
				t.setError(err)
				break loop
			}
		case <-t.ctx.Done():
			t.setError(t.ctx.Err())
			break loop
		}
	}

	return t.err
}

func (t *Batch) setError(err error) {
	t.err = err

	t.cancel()

	// Drain as much as we can without blocking.
loop:
	for {
		select {
		case <-t.commitResults:
		default:
			break loop
		}
	}

	// Be nice and cleanup. These can take a *lot* of memory.
	t.commitResults = nil
	t.ds = nil
	t.ctx = nil
	t.nodes = nil
	t.size = 0
	t.activeCommits = 0
}
