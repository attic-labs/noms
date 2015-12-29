package chunks

import (
	"testing"

	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/attic-labs/noms/Godeps/_workspace/src/github.com/stretchr/testify/suite"
	"github.com/attic-labs/noms/ref"
)

func TestDynamoStoreTestSuite(t *testing.T) {
	suite.Run(t, &DynamoStoreTestSuite{})
}

type DynamoStoreTestSuite struct {
	ChunkStoreTestSuite
	numPuts int
}

func (suite *DynamoStoreTestSuite) SetupTest() {
	ddb := createMockDDB()
	suite.Store = &DynamoStore{
		table:  "table",
		ddbsvc: ddb,
	}
	suite.putCountFn = func() int {
		return ddb.numPuts
	}
}

func (suite *DynamoStoreTestSuite) TearDownTest() {
}

type mockAWSError string

func (m mockAWSError) Error() string   { return string(m) }
func (m mockAWSError) Code() string    { return string(m) }
func (m mockAWSError) Message() string { return string(m) }
func (m mockAWSError) OrigErr() error  { return nil }

type mockDDB struct {
	data    map[string][]byte
	numPuts int
}

func createMockDDB() *mockDDB {
	return &mockDDB{map[string][]byte{}, 0}
}

func (m *mockDDB) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	key := input.Key[tableKeyName]
	var value []byte
	if key.S != nil && *key.S == tableRootKeyValue {
		value = m.data[tableRootKeyValue]
	} else {
		value = m.data[ref.FromData(key.B).String()]
	}

	item := map[string]*dynamodb.AttributeValue{}
	if value != nil {
		item[tableKeyName] = &dynamodb.AttributeValue{S: aws.String(tableRootKeyValue)}
		item[tableValueName] = &dynamodb.AttributeValue{B: value}
	}
	return &dynamodb.GetItemOutput{
		Item: item,
	}, nil
}

func (m *mockDDB) hasRoot() bool {
	return m.data[tableRootKeyValue] != nil
}

func (m *mockDDB) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	key := input.Item[tableKeyName]
	value := input.Item[tableValueName].B

	if key.S != nil && *key.S == tableRootKeyValue {
		initial := *(input.ConditionExpression) == valueNotExistsExpression

		if (initial && m.hasRoot()) || (!initial && string(m.data[tableRootKeyValue]) != string(input.ExpressionAttributeValues[":prev"].B)) {
			return nil, mockAWSError("ConditionalCheckFailedException")
		}

		m.data[tableRootKeyValue] = value
	} else {
		m.data[ref.FromData(key.B).String()] = value
		m.numPuts++
	}

	return &dynamodb.PutItemOutput{}, nil
}
