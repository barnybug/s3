package s3

import (
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Filesystem struct {
	err    error
	conn   s3iface.S3API
	bucket string
	path   string
}

type S3File struct {
	conn   s3iface.S3API
	bucket string
	object *s3.Object
	path   string
	md5    []byte
}

func (self *S3File) Relative() string {
	return self.path
}

func (self *S3File) Size() int64 {
	return *self.object.Size
}

func (self *S3File) IsDirectory() bool {
	return strings.HasSuffix(self.path, "/") && *self.object.Size == 0
}

func (self *S3File) MD5() []byte {
	if self.md5 == nil {
		etag := *self.object.ETag
		v := etag[1 : len(etag)-1]
		self.md5, _ = hex.DecodeString(v)
	}
	return self.md5
}

func (self *S3File) Reader() (io.ReadCloser, error) {
	input := s3.GetObjectInput{
		Bucket: aws.String(self.bucket),
		Key:    self.object.Key,
	}
	output, err := self.conn.GetObject(&input)
	if err != nil {
		return nil, err
	}
	return output.Body, err
}

func (self *S3File) Delete() error {
	input := s3.DeleteObjectInput{
		Bucket: aws.String(self.bucket),
		Key:    self.object.Key,
	}
	_, err := self.conn.DeleteObject(&input)
	return err
}

func (self *S3File) String() string {
	return fmt.Sprintf("s3://%s/%s", self.bucket, *self.object.Key)
}

func (self *S3Filesystem) Error() error {
	return self.err
}

func (self *S3Filesystem) Files() <-chan File {
	ch := make(chan File, 1000)
	stripLen := strings.LastIndex(self.path, "/") + 1
	if stripLen == -1 {
		stripLen = 0
	}
	go func() {
		defer close(ch)
		truncated := true
		marker := ""
		for truncated {
			input := s3.ListObjectsInput{
				Bucket: aws.String(self.bucket),
				Prefix: aws.String(self.path),
				Marker: aws.String(marker),
			}
			output, err := self.conn.ListObjects(&input)
			if err != nil {
				self.err = err
				return
			}
			last_key := ""
			for _, c := range output.Contents {
				key := c
				relpath := (*key.Key)[stripLen:]
				ch <- &S3File{self.conn, self.bucket, key, relpath, nil}
				last_key = *c.Key
			}
			truncated = *output.IsTruncated
			if output.NextMarker != nil {
				marker = *output.NextMarker
			}
			if marker == "" {
				// Response may not include NextMarker.
				marker = last_key
			}
		}
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
	var fullpath string
	if self.path == "" || strings.HasSuffix(self.path, "/") {
		fullpath = filepath.Join(self.path, src.Relative())
	} else {
		fullpath = self.path
	}
	input := s3manager.UploadInput{
		ACL:    aws.String(acl),
		Bucket: aws.String(self.bucket),
		Key:    aws.String(fullpath),
	}

	switch t := src.(type) {
	case *S3File:
		// special case for S3File to preserve header information
		getObjectInput := s3.GetObjectInput{
			Bucket: aws.String(t.bucket),
			Key:    t.object.Key,
		}
		output, err := self.conn.GetObject(&getObjectInput)
		if err != nil {
			return err
		}
		defer output.Body.Close()
		input.Body = output.Body
		// transfer existing headers across
		input.ContentType = output.ContentType
		// input.LastModified = output.LastModified
		input.StorageClass = output.StorageClass
	default:
		reader, err := src.Reader()
		if err != nil {
			return err
		}
		input.Body = reader
		defer reader.Close()
		input.ContentType = aws.String(guessMimeType(src.Relative()))
	}

	u := s3manager.NewUploaderWithClient(self.conn)
	_, err := u.Upload(&input)
	return err
}

func (self *S3Filesystem) Delete(path string) error {
	fullpath := filepath.Join(self.path, path)
	input := s3.DeleteObjectInput{
		Bucket: aws.String(self.bucket),
		Key:    aws.String(fullpath),
	}
	_, err := self.conn.DeleteObject(&input)
	return err
}
