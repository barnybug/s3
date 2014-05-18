package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/mitchellh/goamz/s3"
)

type S3Filesystem struct {
	conn   *s3.S3
	bucket *s3.Bucket
	path   string
}

type S3File struct {
	bucket *s3.Bucket
	key    *s3.Key
	path   string
	md5    []byte
}

func (self *S3File) Name() string {
	return self.path
}

func (self *S3File) Size() int64 {
	return self.key.Size
}

func (self *S3File) MD5() []byte {
	if self.md5 == nil {
		v := self.key.ETag[1 : len(self.key.ETag)-1]
		self.md5, _ = hex.DecodeString(v)
	}
	return self.md5
}

func (self *S3File) Reader() (io.ReadCloser, error) {
	resp, err := self.bucket.GetResponse(self.key.Key)
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}

func (self *S3File) String() string {
	return fmt.Sprintf("s3://%s/%s", self.bucket.Name, self.key.Key)
}

func (self *S3Filesystem) Files() <-chan File {
	ch := make(chan File, 1000)
	go func() {
		defer close(ch)
		iterateKeys(self.conn, self.bucket, self.path, func(key *s3.Key) {
			relpath := key.Key[len(self.path):]
			ch <- &S3File{self.bucket, key, relpath, nil}
		})
	}()
	return ch
}

func guessMimeType(filename string) string {
	ext := mime.TypeByExtension(filepath.Ext(filename))
	if ext == "" {
		ext = "application/binary"
	}
	return ext
}

func (self *S3Filesystem) Create(src File) error {
	var reader io.ReadCloser
	headers := http.Header{}
	perm := s3.Private
	if acl != "" {
		perm = s3.ACL(acl)
	}

	switch t := src.(type) {
	case *S3File:
		// special case for S3File to preserve header information
		resp, err := t.bucket.GetResponse(t.key.Key)
		if err != nil {
			return err
		}
		reader = resp.Body
		defer reader.Close()
		// transfer existing headers across
		headers["Content-Type"] = []string{resp.Header.Get("Content-Type")}
		headers["Last-Modified"] = []string{resp.Header.Get("Last-Modified")}
		headers["x-amz-storage-class"] = []string{t.key.StorageClass}
		if acl == "" {
			// TODO: add "GET Object ACL" to goamz
		}
	default:
		var err error
		reader, err = src.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()
		headers["Content-Type"] = []string{guessMimeType(src.Name())}
	}

	fullpath := filepath.Join(self.path, src.Name())
	err := self.bucket.PutReaderHeader(fullpath, reader, src.Size(), headers, perm)
	return err
}

func (self *S3Filesystem) Delete(path string) error {
	fullpath := filepath.Join(self.path, path)
	return self.bucket.Del(fullpath)
}
