package s3

import "github.com/aws/aws-sdk-go/service/s3"

type S3er interface {
	CreateBucket(*s3.CreateBucketInput) (*s3.CreateBucketOutput, error)
	DeleteBucket(*s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error)
	DeleteObject(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
	DeleteObjects(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error)
	GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
	PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error)
	ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error)
	ListObjects(*s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
}
