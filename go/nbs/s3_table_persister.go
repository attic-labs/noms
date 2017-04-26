// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package nbs

import (
	"bytes"
	"io"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	defaultS3PartSize = 5 * 1 << 20 // 5MiB, smallest allowed by S3
	maxS3Parts        = 10000
)

type s3TablePersister struct {
	s3         s3svc
	bucket     string
	partSize   int
	indexCache *indexCache
	readRl     chan struct{}
}

func (s3p s3TablePersister) Open(name addr, chunkCount uint32) chunkSource {
	return newS3TableReader(s3p.s3, s3p.bucket, name, chunkCount, s3p.indexCache, s3p.readRl)
}

type s3UploadedPart struct {
	idx  int64
	etag string
}

func (s3p s3TablePersister) Persist(mt *memTable, haver chunkReader) chunkSource {
	return s3p.persistTable(mt.write(haver))
}

func (s3p s3TablePersister) persistTable(name addr, data []byte, chunkCount uint32) chunkSource {
	if chunkCount == 0 {
		return emptyChunkSource{}
	}
	t1 := time.Now()
	s3p.multipartUpload(data, name.String())
	verbose.Log("Compacted table of %d Kb in %s", len(data)/1024, time.Since(t1))

	return s3p.newReaderFromIdxData(data, name)
}

func (s3p s3TablePersister) newReaderFromIdxData(idxData []byte, name addr) *s3TableReader {
	s3tr := &s3TableReader{s3: s3p.s3, bucket: s3p.bucket, h: name, readRl: s3p.readRl}
	index := parseTableIndex(idxData)
	if s3p.indexCache != nil {
		s3p.indexCache.put(name, index)
	}
	s3tr.tableReader = newTableReader(index, s3tr, s3BlockSize)
	return s3tr
}

func (s3p s3TablePersister) multipartUpload(data []byte, key string) {
	uploadID := s3p.startMultipartUpload(key)
	multipartUpload, err := s3p.uploadParts(data, key, uploadID)
	if err != nil {
		s3p.abortMultipartUpload(key, uploadID)
		d.PanicIfError(err) // TODO: Better error handling here
	}
	s3p.completeMultipartUpload(key, uploadID, multipartUpload)
}

func (s3p s3TablePersister) startMultipartUpload(key string) string {
	result, err := s3p.s3.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
	})
	d.PanicIfError(err)
	return *result.UploadId
}

func (s3p s3TablePersister) abortMultipartUpload(key, uploadID string) {
	_, abrtErr := s3p.s3.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
		Bucket:   aws.String(s3p.bucket),
		Key:      aws.String(key),
		UploadId: aws.String(uploadID),
	})
	d.PanicIfError(abrtErr)
}

func (s3p s3TablePersister) completeMultipartUpload(key, uploadID string, mpu *s3.CompletedMultipartUpload) {
	_, err := s3p.s3.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(s3p.bucket),
		Key:             aws.String(key),
		MultipartUpload: mpu,
		UploadId:        aws.String(uploadID),
	})
	d.PanicIfError(err)
}

func (s3p s3TablePersister) uploadParts(data []byte, key, uploadID string) (*s3.CompletedMultipartUpload, error) {
	sent, failed, done := make(chan s3UploadedPart), make(chan error), make(chan struct{})

	numParts := getNumParts(len(data), s3p.partSize)
	d.PanicIfTrue(numParts > maxS3Parts) // TODO: BUG 3433: handle > 10k parts
	var wg sync.WaitGroup
	wg.Add(numParts)
	sendPart := func(partNum int) {
		defer wg.Done()

		// Check if upload has been terminated
		select {
		case <-done:
			return
		default:
		}
		// Upload the desired part
		start, end := (partNum-1)*s3p.partSize, partNum*s3p.partSize
		if partNum == numParts { // If this is the last part, make sure it includes any overflow
			end = len(data)
		}
		etag, err := s3p.uploadPart(data[start:end], key, uploadID, int64(partNum))
		if err != nil {
			failed <- err
			return
		}
		// Try to send along part info. In the case that the upload was aborted, reading from done allows this worker to exit correctly.
		select {
		case sent <- s3UploadedPart{int64(partNum), etag}:
		case <-done:
			return
		}
	}
	for i := 1; i <= numParts; i++ {
		go sendPart(i)
	}
	go func() {
		wg.Wait()
		close(sent)
		close(failed)
	}()

	multipartUpload := &s3.CompletedMultipartUpload{}
	var firstFailure error
	for cont := true; cont; {
		select {
		case sentPart, open := <-sent:
			if open {
				multipartUpload.Parts = append(multipartUpload.Parts, &s3.CompletedPart{
					ETag:       aws.String(sentPart.etag),
					PartNumber: aws.Int64(sentPart.idx),
				})
			}
			cont = open

		case err := <-failed:
			if err != nil && firstFailure == nil { // nil err may happen when failed gets closed
				firstFailure = err
				close(done)
			}
		}
	}

	if firstFailure == nil {
		close(done)
	}
	sort.Sort(partsByPartNum(multipartUpload.Parts))
	return multipartUpload, firstFailure
}

func getNumParts(dataLen, partSize int) int {
	numParts := dataLen / partSize
	if numParts == 0 {
		numParts = 1
	}
	return numParts
}

type partsByPartNum []*s3.CompletedPart

func (s partsByPartNum) Len() int {
	return len(s)
}

func (s partsByPartNum) Less(i, j int) bool {
	return *s[i].PartNumber < *s[j].PartNumber
}

func (s partsByPartNum) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s3p s3TablePersister) CompactAll(sources chunkSources) chunkSource {
	plan := planCompaction(sources)
	if plan.chunkCount == 0 {
		return emptyChunkSource{}
	}
	t1 := time.Now()
	name := nameFromSuffixes(plan.suffixes())
	s3p.multipartCopyUpload(plan, name.String())
	verbose.Log("Compacted table of %d Kb in %s", plan.totalCompressedData/1024, time.Since(t1))

	return s3p.newReaderFromIdxData(plan.mergedIndex, name)
}

func (s3p s3TablePersister) multipartCopyUpload(plan compactionPlan, key string) {
	uploadID := s3p.startMultipartUpload(key)
	multipartUpload, err := s3p.uploadPartsCopy(plan, key, uploadID)
	if err != nil {
		s3p.abortMultipartUpload(key, uploadID)
		d.PanicIfError(err) // TODO: Better error handling here
	}
	s3p.completeMultipartUpload(key, uploadID, multipartUpload)
}

func (s3p s3TablePersister) uploadPartsCopy(plan compactionPlan, key, uploadID string) (*s3.CompletedMultipartUpload, error) {
	d.PanicIfTrue(len(plan.sources) > maxS3Parts) // TODO: BUG 3433: handle > 10k parts

	copies, manuals, buff := dividePlan(plan.sources, plan)
	var readWg sync.WaitGroup
	for _, man := range manuals {
		readWg.Add(1)
		go func(m manualPart) {
			defer readWg.Done()
			n, _ := m.r.Read(buff[m.buffStart:m.buffEnd])
			d.PanicIfTrue(int64(n) < m.buffEnd-m.buffStart)
		}(man)
	}
	readWg.Wait()

	type uploadFn func() (etag string, err error)
	sent, failed, done := make(chan s3UploadedPart), make(chan error), make(chan struct{})
	var uploadWg sync.WaitGroup
	sendPart := func(partNum int64, doUpload uploadFn) {
		defer uploadWg.Done()

		// Check if upload has been terminated
		select {
		case <-done:
			return
		default:
		}

		etag, err := doUpload()
		if err != nil {
			failed <- err
			return
		}
		// Try to send along part info. In the case that the upload was aborted, reading from done allows this worker to exit correctly.
		select {
		case sent <- s3UploadedPart{int64(partNum), etag}:
		case <-done:
			return
		}
	}

	numCopyParts := len(copies)
	numManualParts := getNumParts(len(buff), s3p.partSize) // TODO: What if this is too big?
	numParts := numCopyParts + numManualParts
	uploadWg.Add(numParts)
	for i, cp := range copies {
		partNum := int64(i + 1) // parts are 1-indexed
		go sendPart(partNum, func() (etag string, err error) {
			return s3p.uploadPartCopy(cp.name, cp.dataLen, key, uploadID, partNum)
		})
	}
	for i := numCopyParts + 1; i <= numParts; i++ {
		start, end := (i-numCopyParts-1)*s3p.partSize, (i-numCopyParts)*s3p.partSize
		if i == numParts { // If this is the last part, make sure it includes any overflow
			end = len(buff)
		}
		partNum := int64(i)
		go sendPart(partNum, func() (etag string, err error) {
			return s3p.uploadPart(buff[start:end], key, uploadID, partNum)
		})
	}

	go func() {
		uploadWg.Wait()
		close(sent)
		close(failed)
	}()

	multipartUpload := &s3.CompletedMultipartUpload{}
	var firstFailure error
	for cont := true; cont; {
		select {
		case sentPart, open := <-sent:
			if open {
				multipartUpload.Parts = append(multipartUpload.Parts, &s3.CompletedPart{
					ETag:       aws.String(sentPart.etag),
					PartNumber: aws.Int64(sentPart.idx),
				})
			}
			cont = open

		case err := <-failed:
			if err != nil && firstFailure == nil { // nil err may happen when failed gets closed
				firstFailure = err
				close(done)
			}
		}
	}

	if firstFailure == nil {
		close(done)
	}
	sort.Sort(partsByPartNum(multipartUpload.Parts))
	return multipartUpload, firstFailure
}

type copyPart struct {
	name    string
	dataLen int64
}

type manualPart struct {
	r                  io.Reader
	buffStart, buffEnd int64
}

func dividePlan(sources chunkSources, plan compactionPlan) (copies []copyPart, manuals []manualPart, buff []byte) {
	buffSize := uint64(len(plan.mergedIndex))
	var offset int64
	for i, src := range sources {
		dataLen := plan.chunkDataLens[i]
		if dataLen >= defaultS3PartSize { // Big enough to copy part!
			copies = append(copies, copyPart{src.hash().String(), int64(dataLen)})
		} else {
			manuals = append(manuals, manualPart{src.reader(), offset, offset + int64(dataLen)})
			offset += int64(dataLen)
			buffSize += dataLen
		}
	}
	buff = make([]byte, buffSize)
	copy(buff[buffSize-uint64(len(plan.mergedIndex)):], plan.mergedIndex)
	return
}

func (s3p s3TablePersister) uploadPartCopy(src string, dataLen int64, key, uploadID string, partNum int64) (etag string, err error) {
	res, err := s3p.s3.UploadPartCopy(&s3.UploadPartCopyInput{
		// TODO: Use url.PathEscape() once we're on go 1.8
		CopySource:      aws.String(url.QueryEscape(s3p.bucket + "/" + src)),
		CopySourceRange: aws.String(s3RangeHeader(0, dataLen)),
		Bucket:          aws.String(s3p.bucket),
		Key:             aws.String(key),
		PartNumber:      aws.Int64(int64(partNum)),
		UploadId:        aws.String(uploadID),
	})
	if err == nil {
		etag = *res.CopyPartResult.ETag
	}
	return
}

func (s3p s3TablePersister) uploadPart(data []byte, key, uploadID string, partNum int64) (etag string, err error) {
	res, err := s3p.s3.UploadPart(&s3.UploadPartInput{
		Bucket:     aws.String(s3p.bucket),
		Key:        aws.String(key),
		PartNumber: aws.Int64(int64(partNum)),
		UploadId:   aws.String(uploadID),
		Body:       bytes.NewReader(data),
	})
	if err == nil {
		etag = *res.ETag
	}
	return
}
