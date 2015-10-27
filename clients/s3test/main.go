package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
        "math/rand"
	"runtime"
        "time"

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
	obj    = &bytes.Buffer{}
)

type randReader struct {
	s    rand.Source
}

func (r *randReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(r.s.Int63() & 0xff)
	}
	return len(p), nil
}

func getRandomReader() io.Reader {
	return &randReader{rand.NewSource(time.Now().UnixNano())}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	flag.Parse()

	out := make(chan struct{}, *num)

	s3svc := s3.New(&aws.Config{Region: aws.String("us-west-2")})

	n, err := io.CopyN(obj, getRandomReader(), 16*1024)
	d.Chk.NoError(err)
	d.Chk.EqualValues(16*1024, n)

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
		buf = nil
	}
}

func put(s3svc *s3.S3, in chan struct{}, out chan struct{}) {
	for range in {
		//buf := &bytes.Buffer{}
		//_, err := io.CopyN(buf, getRandomReader(), int64(*size*1024))
		//d.Chk.NoError(err)

		name := fmt.Sprintf("%v", time.Now().UnixNano())

		//fmt.Println("uploading ", name, obj.Len())
		_, err := s3svc.PutObject(&s3.PutObjectInput{
			Body:   bytes.NewReader(obj.Bytes()),
			Bucket: bucket,
			Key:    &name,
		})
		d.Chk.NoError(err)
		out <- struct{}{}
	}
}
