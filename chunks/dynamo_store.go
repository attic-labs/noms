package chunks

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/session"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/attic-labs/noms/d"
	"github.com/attic-labs/noms/ref"
)

var (
	dynamoMaxValueSize       = 4 * 1024 * 1024
	tableKeyName             = "nomsKey"
	tableValueName           = "nomsValue"
	tableRootKeyValue        = "root"
	valueNotExistsExpression = fmt.Sprintf("attribute_not_exists(%s)", tableValueName)
	valueEqualsExpression    = fmt.Sprintf("%s = :prev", tableValueName)
)

type ddbsvc interface {
	GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
}

type DynamoStore struct {
	table      string
	ddbsvc     ddbsvc
	writeTime  int64
	writeCount int64
	readTime   int64
	readCount  int64
	mu         sync.Mutex
}

func NewDynamoStore(table, region, key, secret string) *DynamoStore {
	creds := defaults.CredChain(defaults.Config(), defaults.Handlers())

	if key != "" {
		creds = credentials.NewStaticCredentials(key, secret, "")
	}

	sess := session.New(&aws.Config{Region: aws.String(region), Credentials: creds})

	store := DynamoStore{
		table,
		dynamodb.New(sess),
		0,
		0,
		0,
		0,
		sync.Mutex{},
	}
	return &store
}

func (s *DynamoStore) Root() ref.Ref {
	result, err := s.ddbsvc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]*dynamodb.AttributeValue{
			tableKeyName: {S: aws.String(tableRootKeyValue)},
		},
	})
	d.Exp.NoError(err)

	if len(result.Item) == 0 {
		return ref.Ref{}
	}

	d.Chk.Equal(len(result.Item), 2)
	return ref.Parse(*result.Item[tableValueName].S)
}

func (s *DynamoStore) UpdateRoot(current, last ref.Ref) bool {
	putArgs := dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]*dynamodb.AttributeValue{
			tableKeyName:   {S: aws.String(tableRootKeyValue)},
			tableValueName: {S: aws.String(current.String())},
		},
	}

	if (last == ref.Ref{}) {
		putArgs.ConditionExpression = aws.String(valueNotExistsExpression)
	} else {
		putArgs.ConditionExpression = aws.String(valueEqualsExpression)
		putArgs.ExpressionAttributeValues = map[string]*dynamodb.AttributeValue{
			":prev": {S: aws.String(last.String())},
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

func (s *DynamoStore) Get(ref ref.Ref) Chunk {
	n := time.Now().UnixNano()
	defer func() {
		s.mu.Lock()
		s.readCount++
		s.readTime += time.Now().UnixNano() - n
		s.mu.Unlock()
	}()

	result, err := s.ddbsvc.GetItem(&dynamodb.GetItemInput{
		TableName: &s.table,
		Key: map[string]*dynamodb.AttributeValue{
			tableKeyName: {S: aws.String(ref.String())},
		},
	})

	s.mu.Lock()
	s.readCount++
	s.mu.Unlock()

	d.Chk.NoError(err)

	if len(result.Item) == 0 {
		return EmptyChunk
	}

	d.Chk.Equal(len(result.Item), 2)

	str := *result.Item[tableValueName].S
	return NewChunkWithRef(ref, []byte(str))
}

func (s *DynamoStore) Has(ref ref.Ref) bool {
	n := time.Now().UnixNano()
	defer func() {
		s.mu.Lock()
		s.readCount++
		s.readTime += time.Now().UnixNano() - n
		s.mu.Unlock()
	}()

	result, err := s.ddbsvc.GetItem(&dynamodb.GetItemInput{
		TableName: &s.table,
		Key: map[string]*dynamodb.AttributeValue{
			tableKeyName: {S: aws.String(ref.String())},
		},
	})

	d.Chk.NoError(err)

	return len(result.Item) > 0
}

func (s *DynamoStore) Put(c Chunk) {
	n := time.Now().UnixNano()
	defer func() {
		s.mu.Lock()
		s.writeCount++
		s.writeTime += time.Now().UnixNano() - n
		s.mu.Unlock()
	}()

	_, err := s.ddbsvc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: map[string]*dynamodb.AttributeValue{
			tableKeyName:   {S: aws.String(c.Ref().String())},
			tableValueName: {S: aws.String(string(c.Data()))},
		},
	})
	d.Chk.NoError(err)
}

func (s *DynamoStore) Close() error {
	fmt.Printf("Read count: %d, Read latency: %d\n", s.readCount, s.readTime/s.readCount/1e6)
	fmt.Printf("Write count: %d, Write latency: %d\n", s.writeCount, s.writeTime/s.writeCount/1e6)
	return nil
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
		flag.String(prefix+"dynamo-table", "noms-values", "dynamodb table to store the values of the chunkstore in"),
		flag.String(prefix+"aws-region", "us-west-2", "aws region to put the aws-based chunkstore in"),
		flag.Bool(prefix+"aws-auth-from-env", false, "creates the aws-based chunkstore from authorization found in the environment. This is typically used in production to get keys from IAM profile. If not specified, then -aws-key and aws-secret must be specified instead"),
		flag.String(prefix+"aws-key", "", "aws key to use to create the aws-based chunkstore"),
		flag.String(prefix+"aws-secret", "", "aws secret to use to create the aws-based chunkstore"),
	}
}

func (f DynamoStoreFlags) CreateStore() ChunkStore {
	if *f.dynamoTable == "" || *f.awsRegion == "" {
		return nil
	}

	if !*f.authFromEnv {
		if *f.awsKey == "" || *f.awsSecret == "" {
			return nil
		}
	}

	return NewDynamoStore(*f.dynamoTable, *f.awsRegion, *f.awsKey, *f.awsSecret)
}
