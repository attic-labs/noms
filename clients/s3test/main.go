package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"runtime"

	"github.com/attic-labs/noms/d"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	bucket = flag.String("bucket", "", "name of bucket")
	mode   = flag.String("mode", "", "either put or get")
	dir    = flag.String("dir", "/tmp/s3test", "dir to create files in")
	num    = flag.Int("num", 100, "number of random files to create")
	p      = flag.Int("p", 10, "amount of parallelism")
	size   = flag.Int("size", 1024, "size of chunks to create (in kb)")
	mon    = make(chan struct{}, 100)
)

func main() {
	runtime.GOMAXPROCS(32)
	flag.Parse()

	out := make(chan struct{}, *num)

	s3svc := s3.New(&aws.Config{Region: aws.String("us-west-2")})

	if *mode == "put" {
		in := make(chan struct{}, *num)
		for i := 0; i < *num; i++ {
			in <- struct{}{}
		}
		close(in)
		for i := 0; i < *p; i++ {
			go put(s3svc, in, out)
		}
	} else if *mode == "get" {
		in := make(chan string, *num)
		num := int64(*num)
		list, err := s3svc.ListObjects(&s3.ListObjectsInput{
			Bucket:  bucket,
			MaxKeys: &num,
		})
		d.Chk.NoError(err)
		for _, obj := range list.Contents {
			in <- *obj.Key
		}
		close(in)
		for i := 0; i < *p; i++ {
			go get(s3svc, in, out)
		}
	} else {
		log.Fatal("Invalid mode ", *mode)
	}

	for i := 0; i < *num; i++ {
		<-out
	}
}

func get(s3svc *s3.S3, in chan string, out chan struct{}) {
	for key := range in {
		fmt.Println("reading", key)
		mon <- struct{}{}
		fmt.Println("num concurrent reads: ", len(mon))
		res, err := s3svc.GetObject(&s3.GetObjectInput{
			Bucket: bucket,
			Key:    &key,
		})
		d.Chk.NoError(err)
		defer res.Body.Close()
		buf := &bytes.Buffer{}
		_, err = io.Copy(buf, res.Body)
		<-mon
		d.Chk.NoError(err)
		fmt.Println(buf.Len())
		out <- struct{}{}
	}
}

func put(s3svc *s3.S3, in chan struct{}, out chan struct{}) {
	for range in {
		buf := &bytes.Buffer{}
		_, err := io.CopyN(buf, rand.Reader, int64(*size*1024))
		d.Chk.NoError(err)

		hash := sha1.Sum(buf.Bytes())
		name := hex.EncodeToString(hash[:])

		fmt.Println("uploading ", name)
		_, err = s3svc.PutObject(&s3.PutObjectInput{
			Body:   bytes.NewReader(buf.Bytes()),
			Bucket: bucket,
			Key:    &name,
		})
		d.Chk.NoError(err)
		out <- struct{}{}
	}
}
