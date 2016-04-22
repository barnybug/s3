package s3

import (
	"bytes"
	"errors"
	"io/ioutil"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

var (
	ErrNoSuchBucket  = errors.New("NoSuchBucket: The specified bucket does not exist")
	ErrBucketExists  = errors.New("Bucket already exists")
	ErrBucketHasKeys = errors.New("Bucket has keys so cannot be deleted")
)

type MockBucket map[string][]byte

type MockS3 struct {
	sync.RWMutex
	// bucket: {key: value}
	data map[string]MockBucket
}

func NewMockS3() *MockS3 {
	return &MockS3{
		data: map[string]MockBucket{},
	}
}

func (self *MockS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	self.RLock()
	defer self.RUnlock()
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

func (self *MockS3) DeleteBucket(input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	self.Lock()
	defer self.Unlock()
	if bucket, exists := self.data[*input.Bucket]; exists {
		if len(bucket) > 0 {
			return nil, ErrBucketHasKeys
		}
		delete(self.data, *input.Bucket)
		return &s3.DeleteBucketOutput{}, nil
	} else {
		return nil, ErrNoSuchBucket
	}
}

func (self *MockS3) CreateBucket(input *s3.CreateBucketInput) (*s3.CreateBucketOutput, error) {
	self.Lock()
	defer self.Unlock()
	if _, exists := self.data[*input.Bucket]; exists {
		return nil, ErrBucketExists
	}
	self.data[*input.Bucket] = MockBucket{}
	return &s3.CreateBucketOutput{}, nil
}

func (self *MockS3) ListObjects(input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	self.RLock()
	defer self.RUnlock()
	bucket, ok := self.data[*input.Bucket]
	if !ok {
		return nil, ErrNoSuchBucket
	}
	var keys []string
	for key := range bucket {
		if strings.HasPrefix(key, *input.Prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	contents := []*s3.Object{}
	for _, key := range keys {
		value := bucket[key]
		object := s3.Object{
			Key:  aws.String(key),
			Size: aws.Int64(int64(len(value))),
		}
		contents = append(contents, &object)
	}

	output := s3.ListObjectsOutput{
		Contents:    contents,
		IsTruncated: aws.Bool(false),
	}
	return &output, nil
}

func (self *MockS3) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	self.RLock()
	defer self.RUnlock()
	bucket := self.data[*input.Bucket]
	if object, ok := bucket[*input.Key]; ok {
		body := ioutil.NopCloser(bytes.NewReader(object))
		output := s3.GetObjectOutput{
			Body: body,
		}
		return &output, nil
	} else {
		return nil, errors.New("missing key")
	}
}

func (self *MockS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	self.Lock()
	defer self.Unlock()
	content, _ := ioutil.ReadAll(input.Body)
	if bucket, ok := self.data[*input.Bucket]; ok {
		bucket[*input.Key] = content
	} else {
		return nil, ErrNoSuchBucket
	}
	return &s3.PutObjectOutput{}, nil
}

func (self *MockS3) PutObjectRequest(input *s3.PutObjectInput) (*request.Request, *s3.PutObjectOutput) {
	self.Lock()
	defer self.Unlock()
	// required for s3manager.Upload
	// TODO: should only alter bucket on Send()
	content, _ := ioutil.ReadAll(input.Body)
	req := request.New(aws.Config{}, metadata.ClientInfo{}, request.Handlers{}, nil, &request.Operation{}, nil, nil)
	if bucket, ok := self.data[*input.Bucket]; ok {
		bucket[*input.Key] = content
	} else {
		// pre-set the error on the request
		req.Build()
		req.Error = ErrNoSuchBucket
	}
	return req, &s3.PutObjectOutput{}
}

func (self *MockS3) DeleteObjects(input *s3.DeleteObjectsInput) (*s3.DeleteObjectsOutput, error) {
	self.Lock()
	defer self.Unlock()
	bucket := self.data[*input.Bucket]
	for _, id := range input.Delete.Objects {
		delete(bucket, *id.Key)
	}
	return &s3.DeleteObjectsOutput{}, nil
}

func (self *MockS3) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	self.Lock()
	defer self.Unlock()
	bucket := self.data[*input.Bucket]
	delete(bucket, *input.Key)
	return &s3.DeleteObjectOutput{}, nil
}

// unimplemented

func (self *MockS3) AbortMultipartUploadRequest(*s3.AbortMultipartUploadInput) (*request.Request, *s3.AbortMultipartUploadOutput) {
	return nil, &s3.AbortMultipartUploadOutput{}
}
func (self *MockS3) AbortMultipartUpload(*s3.AbortMultipartUploadInput) (*s3.AbortMultipartUploadOutput, error) {
	return &s3.AbortMultipartUploadOutput{}, nil
}
func (self *MockS3) CompleteMultipartUploadRequest(*s3.CompleteMultipartUploadInput) (*request.Request, *s3.CompleteMultipartUploadOutput) {
	return nil, &s3.CompleteMultipartUploadOutput{}
}
func (self *MockS3) CompleteMultipartUpload(*s3.CompleteMultipartUploadInput) (*s3.CompleteMultipartUploadOutput, error) {
	return &s3.CompleteMultipartUploadOutput{}, nil
}
func (self *MockS3) CopyObjectRequest(*s3.CopyObjectInput) (*request.Request, *s3.CopyObjectOutput) {
	return nil, &s3.CopyObjectOutput{}
}
func (self *MockS3) CopyObject(*s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	return &s3.CopyObjectOutput{}, nil
}
func (self *MockS3) CreateBucketRequest(*s3.CreateBucketInput) (*request.Request, *s3.CreateBucketOutput) {
	return nil, &s3.CreateBucketOutput{}
}
func (self *MockS3) CreateMultipartUploadRequest(*s3.CreateMultipartUploadInput) (*request.Request, *s3.CreateMultipartUploadOutput) {
	return nil, &s3.CreateMultipartUploadOutput{}
}
func (self *MockS3) CreateMultipartUpload(*s3.CreateMultipartUploadInput) (*s3.CreateMultipartUploadOutput, error) {
	return &s3.CreateMultipartUploadOutput{}, nil
}
func (self *MockS3) DeleteBucketRequest(*s3.DeleteBucketInput) (*request.Request, *s3.DeleteBucketOutput) {
	return nil, &s3.DeleteBucketOutput{}
}
func (self *MockS3) DeleteBucketCorsRequest(*s3.DeleteBucketCorsInput) (*request.Request, *s3.DeleteBucketCorsOutput) {
	return nil, &s3.DeleteBucketCorsOutput{}
}
func (self *MockS3) DeleteBucketCors(*s3.DeleteBucketCorsInput) (*s3.DeleteBucketCorsOutput, error) {
	return &s3.DeleteBucketCorsOutput{}, nil
}
func (self *MockS3) DeleteBucketLifecycleRequest(*s3.DeleteBucketLifecycleInput) (*request.Request, *s3.DeleteBucketLifecycleOutput) {
	return nil, &s3.DeleteBucketLifecycleOutput{}
}
func (self *MockS3) DeleteBucketLifecycle(*s3.DeleteBucketLifecycleInput) (*s3.DeleteBucketLifecycleOutput, error) {
	return &s3.DeleteBucketLifecycleOutput{}, nil
}
func (self *MockS3) DeleteBucketPolicyRequest(*s3.DeleteBucketPolicyInput) (*request.Request, *s3.DeleteBucketPolicyOutput) {
	return nil, &s3.DeleteBucketPolicyOutput{}
}
func (self *MockS3) DeleteBucketPolicy(*s3.DeleteBucketPolicyInput) (*s3.DeleteBucketPolicyOutput, error) {
	return &s3.DeleteBucketPolicyOutput{}, nil
}
func (self *MockS3) DeleteBucketReplicationRequest(*s3.DeleteBucketReplicationInput) (*request.Request, *s3.DeleteBucketReplicationOutput) {
	return nil, &s3.DeleteBucketReplicationOutput{}
}
func (self *MockS3) DeleteBucketReplication(*s3.DeleteBucketReplicationInput) (*s3.DeleteBucketReplicationOutput, error) {
	return &s3.DeleteBucketReplicationOutput{}, nil
}
func (self *MockS3) DeleteBucketTaggingRequest(*s3.DeleteBucketTaggingInput) (*request.Request, *s3.DeleteBucketTaggingOutput) {
	return nil, &s3.DeleteBucketTaggingOutput{}
}
func (self *MockS3) DeleteBucketTagging(*s3.DeleteBucketTaggingInput) (*s3.DeleteBucketTaggingOutput, error) {
	return &s3.DeleteBucketTaggingOutput{}, nil
}
func (self *MockS3) DeleteBucketWebsiteRequest(*s3.DeleteBucketWebsiteInput) (*request.Request, *s3.DeleteBucketWebsiteOutput) {
	return nil, &s3.DeleteBucketWebsiteOutput{}
}
func (self *MockS3) DeleteBucketWebsite(*s3.DeleteBucketWebsiteInput) (*s3.DeleteBucketWebsiteOutput, error) {
	return &s3.DeleteBucketWebsiteOutput{}, nil
}
func (self *MockS3) DeleteObjectRequest(*s3.DeleteObjectInput) (*request.Request, *s3.DeleteObjectOutput) {
	return nil, &s3.DeleteObjectOutput{}
}
func (self *MockS3) DeleteObjectsRequest(*s3.DeleteObjectsInput) (*request.Request, *s3.DeleteObjectsOutput) {
	return nil, &s3.DeleteObjectsOutput{}
}

func (self *MockS3) GetBucketAccelerateConfiguration(input *GetBucketAccelerateConfigurationInput) (*GetBucketAccelerateConfigurationOutput, error) {
	req, out := self.GetBucketAccelerateConfigurationRequest(input)
	err := req.Send()
	return out, err
}

func (self *MockS3) GetBucketAclRequest(*s3.GetBucketAclInput) (*request.Request, *s3.GetBucketAclOutput) {
	return nil, &s3.GetBucketAclOutput{}
}
func (self *MockS3) GetBucketAcl(*s3.GetBucketAclInput) (*s3.GetBucketAclOutput, error) {
	return &s3.GetBucketAclOutput{}, nil
}
func (self *MockS3) GetBucketCorsRequest(*s3.GetBucketCorsInput) (*request.Request, *s3.GetBucketCorsOutput) {
	return nil, &s3.GetBucketCorsOutput{}
}
func (self *MockS3) GetBucketCors(*s3.GetBucketCorsInput) (*s3.GetBucketCorsOutput, error) {
	return &s3.GetBucketCorsOutput{}, nil
}
func (self *MockS3) GetBucketLifecycleRequest(*s3.GetBucketLifecycleInput) (*request.Request, *s3.GetBucketLifecycleOutput) {
	return nil, &s3.GetBucketLifecycleOutput{}
}
func (self *MockS3) GetBucketLifecycle(*s3.GetBucketLifecycleInput) (*s3.GetBucketLifecycleOutput, error) {
	return &s3.GetBucketLifecycleOutput{}, nil
}
func (self *MockS3) GetBucketLifecycleConfigurationRequest(*s3.GetBucketLifecycleConfigurationInput) (*request.Request, *s3.GetBucketLifecycleConfigurationOutput) {
	return nil, &s3.GetBucketLifecycleConfigurationOutput{}
}
func (self *MockS3) GetBucketLifecycleConfiguration(*s3.GetBucketLifecycleConfigurationInput) (*s3.GetBucketLifecycleConfigurationOutput, error) {
	return &s3.GetBucketLifecycleConfigurationOutput{}, nil
}
func (self *MockS3) GetBucketLocationRequest(*s3.GetBucketLocationInput) (*request.Request, *s3.GetBucketLocationOutput) {
	return nil, &s3.GetBucketLocationOutput{}
}
func (self *MockS3) GetBucketLocation(*s3.GetBucketLocationInput) (*s3.GetBucketLocationOutput, error) {
	return &s3.GetBucketLocationOutput{}, nil
}
func (self *MockS3) GetBucketLoggingRequest(*s3.GetBucketLoggingInput) (*request.Request, *s3.GetBucketLoggingOutput) {
	return nil, &s3.GetBucketLoggingOutput{}
}
func (self *MockS3) GetBucketLogging(*s3.GetBucketLoggingInput) (*s3.GetBucketLoggingOutput, error) {
	return &s3.GetBucketLoggingOutput{}, nil
}
func (self *MockS3) GetBucketNotificationRequest(*s3.GetBucketNotificationConfigurationRequest) (*request.Request, *s3.NotificationConfigurationDeprecated) {
	return nil, &s3.NotificationConfigurationDeprecated{}
}
func (self *MockS3) GetBucketNotification(*s3.GetBucketNotificationConfigurationRequest) (*s3.NotificationConfigurationDeprecated, error) {
	return &s3.NotificationConfigurationDeprecated{}, nil
}
func (self *MockS3) GetBucketNotificationConfigurationRequest(*s3.GetBucketNotificationConfigurationRequest) (*request.Request, *s3.NotificationConfiguration) {
	return nil, &s3.NotificationConfiguration{}
}
func (self *MockS3) GetBucketNotificationConfiguration(*s3.GetBucketNotificationConfigurationRequest) (*s3.NotificationConfiguration, error) {
	return &s3.NotificationConfiguration{}, nil
}
func (self *MockS3) GetBucketPolicyRequest(*s3.GetBucketPolicyInput) (*request.Request, *s3.GetBucketPolicyOutput) {
	return nil, &s3.GetBucketPolicyOutput{}
}
func (self *MockS3) GetBucketPolicy(*s3.GetBucketPolicyInput) (*s3.GetBucketPolicyOutput, error) {
	return &s3.GetBucketPolicyOutput{}, nil
}
func (self *MockS3) GetBucketReplicationRequest(*s3.GetBucketReplicationInput) (*request.Request, *s3.GetBucketReplicationOutput) {
	return nil, &s3.GetBucketReplicationOutput{}
}
func (self *MockS3) GetBucketReplication(*s3.GetBucketReplicationInput) (*s3.GetBucketReplicationOutput, error) {
	return &s3.GetBucketReplicationOutput{}, nil
}
func (self *MockS3) GetBucketRequestPaymentRequest(*s3.GetBucketRequestPaymentInput) (*request.Request, *s3.GetBucketRequestPaymentOutput) {
	return nil, &s3.GetBucketRequestPaymentOutput{}
}
func (self *MockS3) GetBucketRequestPayment(*s3.GetBucketRequestPaymentInput) (*s3.GetBucketRequestPaymentOutput, error) {
	return &s3.GetBucketRequestPaymentOutput{}, nil
}
func (self *MockS3) GetBucketTaggingRequest(*s3.GetBucketTaggingInput) (*request.Request, *s3.GetBucketTaggingOutput) {
	return nil, &s3.GetBucketTaggingOutput{}
}
func (self *MockS3) GetBucketTagging(*s3.GetBucketTaggingInput) (*s3.GetBucketTaggingOutput, error) {
	return &s3.GetBucketTaggingOutput{}, nil
}
func (self *MockS3) GetBucketVersioningRequest(*s3.GetBucketVersioningInput) (*request.Request, *s3.GetBucketVersioningOutput) {
	return nil, &s3.GetBucketVersioningOutput{}
}
func (self *MockS3) GetBucketVersioning(*s3.GetBucketVersioningInput) (*s3.GetBucketVersioningOutput, error) {
	return &s3.GetBucketVersioningOutput{}, nil
}
func (self *MockS3) GetBucketWebsiteRequest(*s3.GetBucketWebsiteInput) (*request.Request, *s3.GetBucketWebsiteOutput) {
	return nil, &s3.GetBucketWebsiteOutput{}
}
func (self *MockS3) GetBucketWebsite(*s3.GetBucketWebsiteInput) (*s3.GetBucketWebsiteOutput, error) {
	return &s3.GetBucketWebsiteOutput{}, nil
}
func (self *MockS3) GetObjectRequest(*s3.GetObjectInput) (*request.Request, *s3.GetObjectOutput) {
	return nil, &s3.GetObjectOutput{}
}
func (self *MockS3) GetObjectAclRequest(*s3.GetObjectAclInput) (*request.Request, *s3.GetObjectAclOutput) {
	return nil, &s3.GetObjectAclOutput{}
}
func (self *MockS3) GetObjectAcl(*s3.GetObjectAclInput) (*s3.GetObjectAclOutput, error) {
	return &s3.GetObjectAclOutput{}, nil
}
func (self *MockS3) GetObjectTorrentRequest(*s3.GetObjectTorrentInput) (*request.Request, *s3.GetObjectTorrentOutput) {
	return nil, &s3.GetObjectTorrentOutput{}
}
func (self *MockS3) GetObjectTorrent(*s3.GetObjectTorrentInput) (*s3.GetObjectTorrentOutput, error) {
	return &s3.GetObjectTorrentOutput{}, nil
}
func (self *MockS3) HeadBucketRequest(*s3.HeadBucketInput) (*request.Request, *s3.HeadBucketOutput) {
	return nil, &s3.HeadBucketOutput{}
}
func (self *MockS3) HeadBucket(*s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}
func (self *MockS3) HeadObjectRequest(*s3.HeadObjectInput) (*request.Request, *s3.HeadObjectOutput) {
	return nil, &s3.HeadObjectOutput{}
}
func (self *MockS3) HeadObject(*s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{}, nil
}
func (self *MockS3) ListBucketsRequest(*s3.ListBucketsInput) (*request.Request, *s3.ListBucketsOutput) {
	return nil, &s3.ListBucketsOutput{}
}
func (self *MockS3) ListMultipartUploadsRequest(*s3.ListMultipartUploadsInput) (*request.Request, *s3.ListMultipartUploadsOutput) {
	return nil, &s3.ListMultipartUploadsOutput{}
}
func (self *MockS3) ListMultipartUploads(*s3.ListMultipartUploadsInput) (*s3.ListMultipartUploadsOutput, error) {
	return &s3.ListMultipartUploadsOutput{}, nil
}
func (self *MockS3) ListMultipartUploadsPages(*s3.ListMultipartUploadsInput, func(*s3.ListMultipartUploadsOutput, bool) bool) error {
	return nil
}
func (self *MockS3) ListObjectVersionsRequest(*s3.ListObjectVersionsInput) (*request.Request, *s3.ListObjectVersionsOutput) {
	return nil, &s3.ListObjectVersionsOutput{}
}
func (self *MockS3) ListObjectVersions(*s3.ListObjectVersionsInput) (*s3.ListObjectVersionsOutput, error) {
	return &s3.ListObjectVersionsOutput{}, nil
}
func (self *MockS3) ListObjectVersionsPages(*s3.ListObjectVersionsInput, func(*s3.ListObjectVersionsOutput, bool) bool) error {
	return nil
}
func (self *MockS3) ListObjectsRequest(*s3.ListObjectsInput) (*request.Request, *s3.ListObjectsOutput) {
	return nil, &s3.ListObjectsOutput{}
}
func (self *MockS3) ListObjectsPages(*s3.ListObjectsInput, func(*s3.ListObjectsOutput, bool) bool) error {
	return nil
}
func (self *MockS3) ListPartsRequest(*s3.ListPartsInput) (*request.Request, *s3.ListPartsOutput) {
	return nil, &s3.ListPartsOutput{}
}
func (self *MockS3) ListParts(*s3.ListPartsInput) (*s3.ListPartsOutput, error) {
	return &s3.ListPartsOutput{}, nil
}
func (self *MockS3) ListPartsPages(*s3.ListPartsInput, func(*s3.ListPartsOutput, bool) bool) error {
	return nil
}
func (self *MockS3) PutBucketAclRequest(*s3.PutBucketAclInput) (*request.Request, *s3.PutBucketAclOutput) {
	return nil, &s3.PutBucketAclOutput{}
}
func (self *MockS3) PutBucketAcl(*s3.PutBucketAclInput) (*s3.PutBucketAclOutput, error) {
	return &s3.PutBucketAclOutput{}, nil
}
func (self *MockS3) PutBucketCorsRequest(*s3.PutBucketCorsInput) (*request.Request, *s3.PutBucketCorsOutput) {
	return nil, &s3.PutBucketCorsOutput{}
}
func (self *MockS3) PutBucketCors(*s3.PutBucketCorsInput) (*s3.PutBucketCorsOutput, error) {
	return &s3.PutBucketCorsOutput{}, nil
}
func (self *MockS3) PutBucketLifecycleRequest(*s3.PutBucketLifecycleInput) (*request.Request, *s3.PutBucketLifecycleOutput) {
	return nil, &s3.PutBucketLifecycleOutput{}
}
func (self *MockS3) PutBucketLifecycle(*s3.PutBucketLifecycleInput) (*s3.PutBucketLifecycleOutput, error) {
	return &s3.PutBucketLifecycleOutput{}, nil
}
func (self *MockS3) PutBucketLifecycleConfigurationRequest(*s3.PutBucketLifecycleConfigurationInput) (*request.Request, *s3.PutBucketLifecycleConfigurationOutput) {
	return nil, &s3.PutBucketLifecycleConfigurationOutput{}
}
func (self *MockS3) PutBucketLifecycleConfiguration(*s3.PutBucketLifecycleConfigurationInput) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	return &s3.PutBucketLifecycleConfigurationOutput{}, nil
}
func (self *MockS3) PutBucketLoggingRequest(*s3.PutBucketLoggingInput) (*request.Request, *s3.PutBucketLoggingOutput) {
	return nil, &s3.PutBucketLoggingOutput{}
}
func (self *MockS3) PutBucketLogging(*s3.PutBucketLoggingInput) (*s3.PutBucketLoggingOutput, error) {
	return &s3.PutBucketLoggingOutput{}, nil
}
func (self *MockS3) PutBucketNotificationRequest(*s3.PutBucketNotificationInput) (*request.Request, *s3.PutBucketNotificationOutput) {
	return nil, &s3.PutBucketNotificationOutput{}
}
func (self *MockS3) PutBucketNotification(*s3.PutBucketNotificationInput) (*s3.PutBucketNotificationOutput, error) {
	return &s3.PutBucketNotificationOutput{}, nil
}
func (self *MockS3) PutBucketNotificationConfigurationRequest(*s3.PutBucketNotificationConfigurationInput) (*request.Request, *s3.PutBucketNotificationConfigurationOutput) {
	return nil, &s3.PutBucketNotificationConfigurationOutput{}
}
func (self *MockS3) PutBucketNotificationConfiguration(*s3.PutBucketNotificationConfigurationInput) (*s3.PutBucketNotificationConfigurationOutput, error) {
	return &s3.PutBucketNotificationConfigurationOutput{}, nil
}
func (self *MockS3) PutBucketPolicyRequest(*s3.PutBucketPolicyInput) (*request.Request, *s3.PutBucketPolicyOutput) {
	return nil, &s3.PutBucketPolicyOutput{}
}
func (self *MockS3) PutBucketPolicy(*s3.PutBucketPolicyInput) (*s3.PutBucketPolicyOutput, error) {
	return &s3.PutBucketPolicyOutput{}, nil
}
func (self *MockS3) PutBucketReplicationRequest(*s3.PutBucketReplicationInput) (*request.Request, *s3.PutBucketReplicationOutput) {
	return nil, &s3.PutBucketReplicationOutput{}
}
func (self *MockS3) PutBucketReplication(*s3.PutBucketReplicationInput) (*s3.PutBucketReplicationOutput, error) {
	return &s3.PutBucketReplicationOutput{}, nil
}
func (self *MockS3) PutBucketRequestPaymentRequest(*s3.PutBucketRequestPaymentInput) (*request.Request, *s3.PutBucketRequestPaymentOutput) {
	return nil, &s3.PutBucketRequestPaymentOutput{}
}
func (self *MockS3) PutBucketRequestPayment(*s3.PutBucketRequestPaymentInput) (*s3.PutBucketRequestPaymentOutput, error) {
	return &s3.PutBucketRequestPaymentOutput{}, nil
}
func (self *MockS3) PutBucketTaggingRequest(*s3.PutBucketTaggingInput) (*request.Request, *s3.PutBucketTaggingOutput) {
	return nil, &s3.PutBucketTaggingOutput{}
}
func (self *MockS3) PutBucketTagging(*s3.PutBucketTaggingInput) (*s3.PutBucketTaggingOutput, error) {
	return &s3.PutBucketTaggingOutput{}, nil
}
func (self *MockS3) PutBucketVersioningRequest(*s3.PutBucketVersioningInput) (*request.Request, *s3.PutBucketVersioningOutput) {
	return nil, &s3.PutBucketVersioningOutput{}
}
func (self *MockS3) PutBucketVersioning(*s3.PutBucketVersioningInput) (*s3.PutBucketVersioningOutput, error) {
	return &s3.PutBucketVersioningOutput{}, nil
}
func (self *MockS3) PutBucketWebsiteRequest(*s3.PutBucketWebsiteInput) (*request.Request, *s3.PutBucketWebsiteOutput) {
	return nil, &s3.PutBucketWebsiteOutput{}
}
func (self *MockS3) PutBucketWebsite(*s3.PutBucketWebsiteInput) (*s3.PutBucketWebsiteOutput, error) {
	return &s3.PutBucketWebsiteOutput{}, nil
}
func (self *MockS3) PutObjectAclRequest(*s3.PutObjectAclInput) (*request.Request, *s3.PutObjectAclOutput) {
	return nil, &s3.PutObjectAclOutput{}
}
func (self *MockS3) PutObjectAcl(*s3.PutObjectAclInput) (*s3.PutObjectAclOutput, error) {
	return &s3.PutObjectAclOutput{}, nil
}
func (self *MockS3) RestoreObjectRequest(*s3.RestoreObjectInput) (*request.Request, *s3.RestoreObjectOutput) {
	return nil, &s3.RestoreObjectOutput{}
}
func (self *MockS3) RestoreObject(*s3.RestoreObjectInput) (*s3.RestoreObjectOutput, error) {
	return &s3.RestoreObjectOutput{}, nil
}
func (self *MockS3) UploadPartRequest(*s3.UploadPartInput) (*request.Request, *s3.UploadPartOutput) {
	return nil, &s3.UploadPartOutput{}
}
func (self *MockS3) UploadPart(*s3.UploadPartInput) (*s3.UploadPartOutput, error) {
	return &s3.UploadPartOutput{}, nil
}
func (self *MockS3) UploadPartCopyRequest(*s3.UploadPartCopyInput) (*request.Request, *s3.UploadPartCopyOutput) {
	return nil, &s3.UploadPartCopyOutput{}
}
func (self *MockS3) UploadPartCopy(*s3.UploadPartCopyInput) (*s3.UploadPartCopyOutput, error) {
	return &s3.UploadPartCopyOutput{}, nil
}

var _ s3iface.S3API = (*MockS3)(nil)
