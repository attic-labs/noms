package badger

import (
	"fmt"
	"strings"

	badger "gx/ipfs/QmdKhi5wUQyV9i3GcTyfUmpfTntWjXu8DcyT9HyNbznYrn/badger"

	ds "gx/ipfs/QmPpegoMqhAEqjncrzArm7KVWAkCm78rqL2DPuNjhPrshg/go-datastore"
	dsq "gx/ipfs/QmPpegoMqhAEqjncrzArm7KVWAkCm78rqL2DPuNjhPrshg/go-datastore/query"
	goprocess "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
)

type datastore struct {
	DB *badger.DB

	gcDiscardRatio float64
}

// Options are the badger datastore options, reexported here for convenience.
type Options struct {
	gcDiscardRatio float64

	badger.Options
}

var DefaultOptions = Options{
	gcDiscardRatio: 0.1,

	Options: badger.DefaultOptions,
}

// NewDatastore creates a new badger datastore.
//
// DO NOT set the Dir and/or ValuePath fields of opt, they will be set for you.
func NewDatastore(path string, options *Options) (*datastore, error) {
	// Copy the options because we modify them.
	var opt badger.Options
	var gcDiscardRatio float64
	if options == nil {
		opt = badger.DefaultOptions
		gcDiscardRatio = DefaultOptions.gcDiscardRatio
	} else {
		opt = options.Options
		gcDiscardRatio = options.gcDiscardRatio
	}

	opt.Dir = path
	opt.ValueDir = path

	kv, err := badger.Open(opt)
	if err != nil {
		if strings.HasPrefix(err.Error(), "manifest has unsupported version:") {
			err = fmt.Errorf("unsupported badger version, use github.com/ipfs/badgerds-upgrade to upgrade: %s", err.Error())
		}
		return nil, err
	}

	return &datastore{
		DB: kv,

		gcDiscardRatio: gcDiscardRatio,
	}, nil
}

func (d *datastore) Put(key ds.Key, value interface{}) error {
	val, ok := value.([]byte)
	if !ok {
		return ds.ErrInvalidType
	}

	txn := d.DB.NewTransaction(true)
	defer txn.Discard()

	err := txn.Set(key.Bytes(), val)
	if err != nil {
		return err
	}

	//TODO: Setting callback may potentially make this faster
	return txn.Commit(nil)
}

func (d *datastore) Get(key ds.Key) (value interface{}, err error) {
	txn := d.DB.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key.Bytes())
	if err == badger.ErrKeyNotFound {
		err = ds.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	val, err := item.Value()
	if err != nil {
		return nil, err
	}

	out := make([]byte, len(val))
	copy(out, val)
	return out, nil
}

func (d *datastore) Has(key ds.Key) (bool, error) {
	txn := d.DB.NewTransaction(false)
	defer txn.Discard()
	_, err := txn.Get(key.Bytes())
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (d *datastore) Delete(key ds.Key) error {
	txn := d.DB.NewTransaction(true)
	defer txn.Discard()
	err := txn.Delete(key.Bytes())
	if err != nil {
		return err
	}

	//TODO: callback may potentially make this faster
	return txn.Commit(nil)
}

func (d *datastore) Query(q dsq.Query) (dsq.Results, error) {
	return d.QueryNew(q)
}

func (d *datastore) QueryNew(q dsq.Query) (dsq.Results, error) {
	prefix := []byte(q.Prefix)
	opt := badger.DefaultIteratorOptions
	opt.PrefetchValues = !q.KeysOnly

	txn := d.DB.NewTransaction(false)

	it := txn.NewIterator(opt)
	it.Seek([]byte(q.Prefix))
	if q.Offset > 0 {
		for j := 0; j < q.Offset; j++ {
			it.Next()
		}
	}

	qrb := dsq.NewResultBuilder(q)

	qrb.Process.Go(func(worker goprocess.Process) {
		defer txn.Discard()
		defer it.Close()

		for sent := 0; it.ValidForPrefix(prefix); sent++ {
			if qrb.Query.Limit > 0 && sent >= qrb.Query.Limit {
				break
			}

			item := it.Item()

			k := string(item.Key())
			e := dsq.Entry{Key: k}

			var result dsq.Result
			if !q.KeysOnly {
				b, err := item.Value()
				if err != nil {
					result = dsq.Result{Error: err}
				} else {
					bytes := make([]byte, len(b))
					copy(bytes, b)
					e.Value = bytes
					result = dsq.Result{Entry: e}
				}
			} else {
				result = dsq.Result{Entry: e}
			}

			select {
			case qrb.Output <- result:
			case <-worker.Closing(): // client told us to close early
				return
			}
			it.Next()
		}

		return
	})

	go qrb.Process.CloseAfterChildren()

	// Now, apply remaining things (filters, order)
	qr := qrb.Results()
	for _, f := range q.Filters {
		qr = dsq.NaiveFilter(qr, f)
	}
	for _, o := range q.Orders {
		qr = dsq.NaiveOrder(qr, o)
	}

	return qr, nil
}

func (d *datastore) Close() error {
	return d.DB.Close()
}

func (d *datastore) IsThreadSafe() {}

type badgerBatch struct {
	db  *badger.DB
	txn *badger.Txn
}

func (d *datastore) Batch() (ds.Batch, error) {
	return &badgerBatch{
		db:  d.DB,
		txn: d.DB.NewTransaction(true),
	}, nil
}

func (b *badgerBatch) Put(key ds.Key, value interface{}) error {
	val, ok := value.([]byte)
	if !ok {
		return ds.ErrInvalidType
	}

	err := b.txn.Set(key.Bytes(), val)
	if err != nil {
		b.txn.Discard()
	}
	return err
}

func (b *badgerBatch) Delete(key ds.Key) error {
	err := b.txn.Delete(key.Bytes())
	if err != nil {
		b.txn.Discard()
	}
	return err
}

func (b *badgerBatch) Commit() error {
	//TODO: Setting callback may potentially make this faster
	return b.txn.Commit(nil)
}

func (d *datastore) CollectGarbage() error {
	err := d.DB.PurgeOlderVersions()
	if err != nil {
		return err
	}

	err = d.DB.RunValueLogGC(d.gcDiscardRatio)
	if err == badger.ErrNoRewrite {
		err = nil
	}
	return err
}
