package s3

import (
	"bytes"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type MockBucket map[string][]byte

type MockS3 struct {
	// bucket: {key: value}
	data map[string]MockBucket
}

func NewMockS3() *MockS3 {
	return &MockS3{
		data: map[string]MockBucket{},
	}
}

func (self *MockS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	buckets := []*s3.Bucket{}
	for name := range self.data {
		bucket := s3.Bucket{Name: aws.String(name)}
		buckets = append(buckets, &bucket)
	}
	output := s3.ListBucketsOutput{
		Buckets: buckets,
	}
	return &output, nil
}

func (self *MockS3) DeleteBucket(*s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return &s3.DeleteBucketOutput{}, nil
}

func (self *MockS3) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	self.data[*input.Bucket] = MockBucket{}
	return &s3.CreateBucketOutput{}, nil
}

func (self *MockS3) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	// TODO: implement error for missing bucket
	bucket := self.data[*input.Bucket]
	contents := []*s3.Object{}
	for key, value := range bucket {
		if strings.HasPrefix(key, *input.Prefix) {
			object := s3.Object{
				Key:  aws.String(key),
				Size: aws.Int64(int64(len(value))),
			}
			contents = append(contents, &object)
		}
	}

	output := s3.ListObjectsOutput{
		Contents:    contents,
		IsTruncated: aws.Bool(false),
	}
	return &output, nil
}

func (self *MockS3) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	bucket := self.data[*input.Bucket]
	object := bucket[*input.Key]
	body := ioutil.NopCloser(bytes.NewReader(object))
	output := s3.GetObjectOutput{
		Body: body,
	}
	return &output, nil
}

func (self *MockS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	content, _ := ioutil.ReadAll(input.Body)
	self.data[*input.Bucket][*input.Key] = content
	return &s3.PutObjectOutput{}, nil
}

func (self *MockS3) DeleteObjects(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	return &s3.DeleteObjectsOutput{}, nil
}

func (self *MockS3) DeleteObject(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}
