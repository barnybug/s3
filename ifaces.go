package s3

import "github.com/aws/aws-sdk-go/service/s3"

type S3er interface {
	ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error)
	DeleteBucket(*s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error)
	PutBucketAcl(*s3.PutBucketAclInput) (*s3.PutBucketAclOutput, error)
	ListObjects(*s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
	GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error)
	DeleteObjects(*s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error)
	DeleteObject(*s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
}
