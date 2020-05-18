// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

// Package spec provides builders and parsers for spelling Noms databases,
// datasets and values.
package aws

import (
	"errors"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/spec/lite"
)

var GetAWSSession func() *session.Session = func() *session.Session {
	return session.Must(session.NewSession(aws.NewConfig().WithRegion("us-west-2")))
}

type awsProtocol struct{}

var pattern = regexp.MustCompile("^[^/]+/[^/]+/.*$")

func (t *awsProtocol) Parse(name string) (string, error) {
	var err error
	if !pattern.MatchString(name) {
		err = errors.New("aws spec must match pattern aws:" + pattern.String())
	}
	return name, err
}

func (t *awsProtocol) NewChunkStore(sp spec.Spec) (chunks.ChunkStore, error) {
	parts := strings.SplitN(sp.DatabaseName, "/", 3) // table/bucket/ns
	d.PanicIfFalse(len(parts) >= 3)                  // parse should have ensured this was true
	sess := GetAWSSession()
	return nbs.NewAWSStore(parts[0], parts[2], parts[1], s3.New(sess), dynamodb.New(sess), 1<<28), nil
}

func (t *awsProtocol) NewDatabase(sp spec.Spec) (datas.Database, error) {
	return datas.NewDatabase(sp.NewChunkStore()), nil
}

func init() {
	spec.ExternalProtocols["aws"] = &awsProtocol{}
}
