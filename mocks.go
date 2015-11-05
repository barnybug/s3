package main

import "github.com/aws/aws-sdk-go/service/s3"

type MockS3 struct {
}

func (self *MockS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return &s3.ListBucketsOutput{}, nil
}

func (self *MockS3) DeleteBucket(*s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return &s3.DeleteBucketOutput{}, nil
}

func (self *MockS3) PutBucketAcl(*s3.PutBucketAclInput) (*s3.PutBucketAclOutput, error) {
	return &s3.PutBucketAclOutput{}, nil
}

func (self *MockS3) ListObjects(*s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	return &s3.ListObjectsOutput{}, nil
}

func (self *MockS3) GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

func (self *MockS3) DeleteObjects(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	return &s3.DeleteObjectsOutput{}, nil
}

func (self *MockS3) DeleteObject(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}
