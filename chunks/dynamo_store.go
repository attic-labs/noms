package chunks

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

const (
	dynamoMaxGetCount      = 100
	dynamoMaxPutCount      = 25
	dynamoMaxPutSize       = 400 * 1024 // 400K
	dynamoWriteUnitSize    = 1024       // 1K
	dynamoWriteConcurrency = 6
	dynamoReadConcurrency  = 1

	dynamoTableName = "noms"
	refAttr         = "ref"
	chunkAttr       = "chunk"
	compAttr        = "comp"
	noneValue       = "none"
	gzipValue       = "gzip"
)

var (
	dynamoStats              = flag.Bool("dynamo-stats", false, "On each DynamoStore close, print read and write stats. Can be quite verbose")
	dynamoWriteBufferSize    = dynamoWriteConcurrency * dynamoMaxPutCount
	dynamoRootKey            = []byte("root")
	valueNotExistsExpression = fmt.Sprintf("attribute_not_exists(%s)", chunkAttr)
	valueEqualsExpression    = fmt.Sprintf("%s = :prev", chunkAttr)
)

type ddbsvc interface {
	BatchGetItem(input *dynamodb.BatchGetItemInput) (*dynamodb.BatchGetItemOutput, error)
	BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error)
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

// DynamoStore implements ChunkStore by storing data to DynamoDB and, if needed, S3.
type DynamoStore struct {
	table          string
	namespace      []byte
	namespaceLen   int
	rootKey        []byte
	ddbsvc         ddbsvc
	stats          *statKeeper
	readQueue      chan readRequest
	writeQueue     chan Chunk
	updateRootChan chan struct{}
	finishedChan   chan struct{}
	requestWg      *sync.WaitGroup
	workerWg       *sync.WaitGroup
	unwrittenPuts  *unwrittenPutCache
}

// NewDynamoStore returns a new DynamoStore instance pointed at a DynamoDB table in the given region. All keys used to access items are prefixed with the given namespace. If key and secret are empty, the DynamoStore will attempt to inherit AWS credentials from the environment.
func NewDynamoStore(table, namespace, region, key, secret string) *DynamoStore {
	config := aws.NewConfig().WithRegion(region)
	if key != "" {
		config = config.WithCredentials(credentials.NewStaticCredentials(key, secret, ""))
	}

	sess := session.New(config)
	return newDynamoStoreFromDDBsvc(table, namespace, dynamodb.New(sess))
}

func newDynamoStoreFromDDBsvc(table, namespace string, ddb ddbsvc) *DynamoStore {
	store := &DynamoStore{
		table:          table,
		namespace:      []byte(namespace),
		ddbsvc:         ddb,
		stats:          newStatKeeper(dynamoWriteConcurrency + dynamoReadConcurrency),
		readQueue:      make(chan readRequest, readBufferSize),
		writeQueue:     make(chan Chunk, dynamoWriteBufferSize),
		updateRootChan: make(chan struct{}),
		finishedChan:   make(chan struct{}),
		requestWg:      &sync.WaitGroup{},
		workerWg:       &sync.WaitGroup{},
		unwrittenPuts:  newUnwrittenPutCache(),
	}
	store.stats.AddStat("writeTime")
	store.stats.AddStat("writeBatchCount")
	store.stats.AddStat("writeCount")
	store.stats.AddStat("writeTotal")
	store.stats.AddStat("writeCompTotal")
	store.stats.AddStat("readTime")
	store.stats.AddStat("readBatchCount")
	store.namespaceLen = len(store.namespace)
	store.rootKey = append(store.namespace, dynamoRootKey...)

	store.batchGetRequests()
	store.batchPutRequests()
	return store
}

func (s *DynamoStore) Get(r ref.Ref) Chunk {
	pending := s.unwrittenPuts.Get(r)
	if !pending.IsEmpty() {
		return pending
	}

	ch := make(chan Chunk)
	s.requestWg.Add(1)
	s.readQueue <- getRequest{r, ch}
	return <-ch
}

func (s *DynamoStore) Has(r ref.Ref) bool {
	pending := s.unwrittenPuts.Get(r)
	if !pending.IsEmpty() {
		return true
	}

	ch := make(chan bool)
	s.requestWg.Add(1)
	s.readQueue <- hasRequest{r, ch}
	return <-ch
}

func (s *DynamoStore) PutMany(chunks ...Chunk) (e BackpressureError) {
	for i, c := range chunks {
		if s.unwrittenPuts.Has(c) {
			continue
		}
		select {
		case s.writeQueue <- c:
			s.requestWg.Add(1)
			s.unwrittenPuts.Add(c)
		default:
			return BackpressureError(chunks[i:])
		}
	}
	return
}

func (s *DynamoStore) Put(c Chunk) {
	if !s.unwrittenPuts.Add(c) {
		return
	}

	s.requestWg.Add(1)
	s.writeQueue <- c
}

func (s *DynamoStore) batchGetRequests() {
	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()

		for done := false; !done; {
			select {
			case req := <-s.readQueue:
				s.sendGetRequests(req)
			case <-s.finishedChan:
				done = true
			}
			// Drain the readQueue before returning
			select {
			case req := <-s.readQueue:
				s.sendGetRequests(req)
			default:
				//drained!
			}
		}
	}()
}

func (s *DynamoStore) sendGetRequests(req readRequest) {
	n := time.Now().UnixNano()
	batch := readBatch{}
	refs := map[ref.Ref]bool{}

	addReq := func(req readRequest) {
		r := req.Ref()
		batch[r] = append(batch[r], req.Outstanding())
		refs[r] = true
	}

	addReq(req)
	for drained := false; !drained && len(refs) < dynamoMaxGetCount; {
		select {
		case req := <-s.readQueue:
			addReq(req)
		default:
			drained = true
		}
	}

	fullBatchSize := len(batch)
	defer func() {
		s.stats.Chan("readBatchCount") <- 1
		s.stats.Chan("readTime") <- time.Now().UnixNano() - n
		s.requestWg.Add(-fullBatchSize)
		batch.Close()
	}()

	requestItems := s.buildRequestItems(refs)
	for hasUnprocessedKeys := true; hasUnprocessedKeys; {
		out, err := s.ddbsvc.BatchGetItem(&dynamodb.BatchGetItemInput{
			RequestItems: requestItems,
		})

		if err == nil {
			s.processResponses(out.Responses[s.table], batch)
		} else if err.(awserr.Error).Code() != "ProvisionedThroughputExceededException" {
			d.Chk.NoError(err, "Errors from BatchGetItem() other than throughput exceeded are fatal")
		}

		hasUnprocessedKeys = len(out.UnprocessedKeys) != 0
		requestItems = out.UnprocessedKeys
	}
}

func (s *DynamoStore) buildRequestItems(refs map[ref.Ref]bool) map[string]*dynamodb.KeysAndAttributes {
	makeKeysAndAttrs := func() *dynamodb.KeysAndAttributes {
		out := &dynamodb.KeysAndAttributes{ConsistentRead: aws.Bool(true)} // This doubles the cost :-(
		for r := range refs {
			out.Keys = append(out.Keys, map[string]*dynamodb.AttributeValue{refAttr: {B: s.makeNamespacedKey(r)}})
		}
		return out
	}
	return map[string]*dynamodb.KeysAndAttributes{s.table: makeKeysAndAttrs()}
}

func (s *DynamoStore) processResponses(responses []map[string]*dynamodb.AttributeValue, batch readBatch) {
	for _, item := range responses {
		p := item[refAttr]
		d.Chk.NotNil(p)
		r := ref.FromSlice(s.removeNamespace(p.B))
		p = item[chunkAttr]
		d.Chk.NotNil(p)
		b := p.B
		if p = item[compAttr]; p != nil && *p.S == gzipValue {
			gr, err := gzip.NewReader(bytes.NewReader(b))
			d.Chk.NoError(err)
			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, gr)
			d.Chk.NoError(err)
			b = buf.Bytes()
		}
		c := NewChunkWithRef(r, b)
		for _, reqChan := range batch[r] {
			reqChan.Satisfy(c)
		}
		delete(batch, r)
	}
}

func (s *DynamoStore) batchPutRequests() {
	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()

		rateLimit := make(chan struct{}, dynamoWriteConcurrency)
		var chunks []Chunk
		for done := false; !done; {
			drainAndSend := false
			select {
			case c := <-s.writeQueue:
				if chunkItemSize(c) > dynamoMaxPutSize {
					s.writeLargeChunk(c)
					break
				}
				chunks = append(chunks, c)
				if len(chunks) == dynamoMaxPutCount {
					s.sendWriteRequests(chunks, rateLimit) // Takes ownership of chunks
					chunks = nil
				}
			case <-s.updateRootChan:
				drainAndSend = true
			case <-s.finishedChan:
				drainAndSend = true
				done = true
			}

			if drainAndSend {
				for drained := false; !drained; {
					select {
					case c := <-s.writeQueue:
						chunks = append(chunks, c)
					default:
						drained = true
					}
					if len(chunks) == dynamoMaxPutCount || (drained && chunks != nil) {
						s.sendWriteRequests(chunks, rateLimit) // Takes ownership of chunks
						chunks = nil
					}
				}
			}
		}
		d.Chk.Nil(chunks, "%d chunks were never sent to server", len(chunks))
	}()
}

func (s *DynamoStore) sendWriteRequests(chunks []Chunk, rateLimit chan struct{}) {
	rateLimit <- struct{}{}
	startTime := time.Now().UnixNano()
	go func() {
		requestItems := s.buildWriteRequests(chunks)
		for hasUnprocessedItems := true; hasUnprocessedItems; {
			out, err := s.ddbsvc.BatchWriteItem(&dynamodb.BatchWriteItemInput{
				RequestItems: requestItems,
			})

			if err != nil && err.(awserr.Error).Code() != "ProvisionedThroughputExceededException" {
				d.Chk.NoError(err, "Errors from BatchGetItem() other than throughput exceeded are fatal")
			}

			hasUnprocessedItems = len(out.UnprocessedItems) != 0
			requestItems = out.UnprocessedItems
		}

		s.stats.Chan("writeBatchCount") <- 1
		s.stats.Chan("writeTime") <- time.Now().UnixNano() - startTime

		s.unwrittenPuts.Clear(chunks)
		s.requestWg.Add(-len(chunks))
		<-rateLimit
	}()
}

func chunkItemSize(c Chunk) int {
	r := c.Ref()
	return len(refAttr) + len(r.DigestSlice()) + len(chunkAttr) + len(c.Data()) + len(compAttr) + len(noneValue)
}

func (s *DynamoStore) buildWriteRequests(chunks []Chunk) map[string][]*dynamodb.WriteRequest {
	totalChunkData, totalCompData := 0, 0
	chunkToItem := func(c Chunk) map[string]*dynamodb.AttributeValue {
		compression := noneValue
		chunkData := c.Data()
		compDataLen := len(chunkData)
		totalChunkData += len(chunkData)

		if chunkItemSize(c) > dynamoWriteUnitSize {
			compression = gzipValue
			buf := &bytes.Buffer{}
			gw := gzip.NewWriter(buf)
			_, err := io.Copy(gw, bytes.NewReader(chunkData))
			d.Chk.NoError(err)
			gw.Close()
			chunkData = buf.Bytes()
			compDataLen = buf.Len()
		}

		totalCompData += compDataLen
		return map[string]*dynamodb.AttributeValue{
			refAttr:   {B: s.makeNamespacedKey(c.Ref())},
			chunkAttr: {B: chunkData},
			compAttr:  {S: aws.String(compression)},
		}
	}
	var requests []*dynamodb.WriteRequest
	for _, c := range chunks {
		requests = append(requests, &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{Item: chunkToItem(c)},
		})
	}
	s.stats.Chan("writeCount") <- int64(len(chunks))
	s.stats.Chan("writeTotal") <- int64(totalChunkData)
	s.stats.Chan("writeCompTotal") <- int64(totalCompData)
	return map[string][]*dynamodb.WriteRequest{s.table: requests}
}

func (s *DynamoStore) writeLargeChunk(c Chunk) {
	d.Chk.Fail("Unsupported!")
}

func (s *DynamoStore) Close() error {
	close(s.finishedChan)
	s.requestWg.Wait()
	s.workerWg.Wait()

	close(s.readQueue)
	close(s.writeQueue)

	s.stats.Stop()
	if *dynamoStats {
		if s.stats.Has("readBatchCount") {
			fmt.Printf("Read batch count: %d, Read batch latency: %dms\n", s.stats.Get("readBatchCount"), s.stats.Get("readTime")/s.stats.Get("readBatchCount")/1e6)
		}
		if s.stats.Has("writeBatchCount") {
			fmt.Printf("Write batch count: %d, Write batch latency: %dms\n", s.stats.Get("writeBatchCount"), s.stats.Get("writeTime")/s.stats.Get("writeBatchCount")/1e6)
		}
		if s.stats.Has("writeCount") {
			writeTotal := float64(s.stats.Get("writeTotal"))
			writeCompTotal := float64(s.stats.Get("writeCompTotal"))
			writeCount := float64(s.stats.Get("writeCount"))
			fmt.Printf("Write chunk count: %.0f, Avg chunk size: %.3fK\n", writeCount, writeTotal/writeCount/1024.0)
			fmt.Printf("Avg compression ratio: %.2fx, Avg compressed chunk size: %.3fK\n", writeTotal/writeCompTotal, writeCompTotal/writeCount/1024.0)
		}
	}
	return nil
}

func (s *DynamoStore) Root() ref.Ref {
	result, err := s.ddbsvc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]*dynamodb.AttributeValue{
			refAttr: {B: s.rootKey},
		},
	})
	d.Exp.NoError(err)

	if len(result.Item) == 0 {
		return ref.Ref{}
	}

	itemLen := len(result.Item)
	d.Chk.True(itemLen == 2 || itemLen == 3)
	if itemLen == 3 {
		d.Chk.NotNil(result.Item[compAttr])
		d.Chk.NotNil(result.Item[compAttr].S)
		d.Chk.Equal(noneValue, *result.Item[compAttr].S)
	}
	return ref.FromSlice(result.Item[chunkAttr].B)
}

func (s *DynamoStore) UpdateRoot(current, last ref.Ref) bool {
	s.updateRootChan <- struct{}{}
	s.requestWg.Wait()

	putArgs := dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]*dynamodb.AttributeValue{
			refAttr:   {B: s.rootKey},
			chunkAttr: {B: current.DigestSlice()},
			compAttr:  {S: aws.String(noneValue)},
		},
	}

	if last.IsEmpty() {
		putArgs.ConditionExpression = aws.String(valueNotExistsExpression)
	} else {
		putArgs.ConditionExpression = aws.String(valueEqualsExpression)
		putArgs.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{
			":prev": {B: last.DigestSlice()},
		}
	}

	_, err := s.ddbsvc.PutItem(&putArgs)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ConditionalCheckFailedException" {
				return false
			}
			d.Chk.NoError(awsErr)
		} else {
			d.Chk.NoError(err)
		}
	}

	return true
}

func (s *DynamoStore) makeNamespacedKey(r ref.Ref) []byte {
	// This is semantically `return append(s.namespace, r.DigestSlice()...)`, but it seemed like we'd be doing this a LOT, and we know how much space we're going to need anyway. So, pre-allocate a slice and then copy into it.
	refSlice := r.DigestSlice()
	key := make([]byte, s.namespaceLen+len(refSlice))
	copy(key, s.namespace)
	copy(key[s.namespaceLen:], refSlice)
	return key
}

func (s *DynamoStore) removeNamespace(namespaced []byte) []byte {
	return namespaced[len(s.namespace):]
}

type DynamoStoreFlags struct {
	dynamoTable *string
	awsRegion   *string
	authFromEnv *bool
	awsKey      *string
	awsSecret   *string
}

func DynamoFlags(prefix string) DynamoStoreFlags {
	return DynamoStoreFlags{
		flag.String(prefix+"dynamo-table", dynamoTableName, "dynamodb table to store the values of the chunkstore in. You probably don't want to change this."),
		flag.String(prefix+"aws-region", "us-west-2", "aws region to put the aws-based chunkstore in"),
		flag.Bool(prefix+"aws-auth-from-env", false, "creates the aws-based chunkstore from authorization found in the environment. This is typically used in production to get keys from IAM profile. If not specified, then -aws-key and aws-secret must be specified instead"),
		flag.String(prefix+"aws-key", "", "aws key to use to create the aws-based chunkstore"),
		flag.String(prefix+"aws-secret", "", "aws secret to use to create the aws-based chunkstore"),
	}
}

func (f DynamoStoreFlags) CreateStore(ns string) ChunkStore {
	if f.check() {
		return NewDynamoStore(*f.dynamoTable, ns, *f.awsRegion, *f.awsKey, *f.awsSecret)
	}
	return nil
}

func (f DynamoStoreFlags) Shutter() {}

func (f DynamoStoreFlags) CreateFactory() (factree Factory) {
	if f.check() {
		factree = f
	}
	return
}

func (f DynamoStoreFlags) check() bool {
	if *f.dynamoTable == "" || *f.awsRegion == "" {
		return false
	}
	if !*f.authFromEnv {
		if *f.awsKey == "" || *f.awsSecret == "" {
			return false
		}
	}
	return true
}
