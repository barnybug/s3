package features

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/barnybug/s3"
	. "github.com/lsegal/gucumber"
)

var conn s3iface.S3API
var testBuckets []string
var out bytes.Buffer
var tempDir string

var replacer = strings.NewReplacer(`\n`, "\n", `\t`, "\t")

func deleteAllKeys(bucket string) {
	truncated := true
	marker := ""
	for truncated {
		input := awss3.ListObjectsInput{
			Bucket: aws.String(bucket),
			Marker: aws.String(marker),
		}
		output, err := conn.ListObjects(&input)
		if err != nil {
			log.Fatal(err.Error())
		}
		last_key := ""
		var objects []*awss3.ObjectIdentifier
		for _, object := range output.Contents {
			id := awss3.ObjectIdentifier{
				Key: object.Key,
			}
			objects = append(objects, &id)
			last_key = *object.Key
		}
		deleteObjectsInput := awss3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &awss3.Delete{
				Objects: objects,
			},
		}
		conn.DeleteObjects(&deleteObjectsInput)

		truncated = *output.IsTruncated
		if output.NextMarker != nil {
			marker = *output.NextMarker
		}
		if marker == "" {
			// Response may not include NextMarker.
			marker = last_key
		}
	}
}

func cleanupBucket(bucket string) {
	deleteAllKeys(bucket)
	input := awss3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	}
	conn.DeleteBucket(&input)
}

func init() {
	Before("", func() {
		conn = s3.NewMockS3()
		out = bytes.Buffer{}
		tempDir, _ = ioutil.TempDir("", "")
		os.Chdir(tempDir)
	})
	After("", func() {
		// Integration tests are mostly run against mock S3, but can be run
		// against a real account, so we should cleanup.
		for _, bucket := range testBuckets {
			cleanupBucket(bucket)
		}
		// Cleanup temp dir
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	Given(`^I have bucket "(.+?)"$`, func(bucket string) {
		input := awss3.CreateBucketInput{
			Bucket: aws.String(bucket),
		}
		conn.CreateBucket(&input)
	})

	Given(`^bucket "(.+?)" key "(.+?)" contains "(.+?)"$`, func(bucket string, key string, content string) {
		body := bytes.NewReader([]byte(content))
		input := awss3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   body,
		}
		conn.PutObject(&input)
	})

	Given(`^local file "(.+?)" contains "(.+?)"$`, func(filename string, content string) {
		file, err := os.Create(filename)
		if err != nil {
			T.Errorf("Couldn't create file: %s\n%s", filename, err)
			return
		}
		defer file.Close()
		file.WriteString(content)
	})

	When(`^I run "(.+?)"$`, func(s1 string) {
		args := strings.Split(s1, " ")
		s3.Main(conn, args, &out)
	})

	Then(`^local file "(.+?)" has contents "(.+?)"$`, func(filename string, exp string) {
		file, err := os.Open(filename)
		if err != nil {
			T.Errorf("Local file error:\n%s", err)
			return
		}
		defer file.Close()
		content, err := ioutil.ReadAll(file)
		if err != nil {
			T.Errorf("Local file error:\n%s", err)
			return
		}
		act := string(content)
		if act != exp {
			T.Errorf("%s contents expected:\n%s\ngot:\n%s", filename, exp, act)
		}
	})

	Then(`^the output is "(.*?)"$`, func(exp string) {
		// replace newlines
		exp = replacer.Replace(exp)
		act := string(out.Bytes())
		if act != exp {
			T.Errorf("Output expected:\n%s\ngot:\n%s", exp, act)
		}
	})

	Then(`^bucket "(.+?)" has key "(.+?)" with contents "(.+?)"$`, func(bucket string, key string, exp string) {
		fmt.Println(bucket, key, exp)
		input := awss3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}
		output, err := conn.GetObject(&input)
		if err != nil {
			T.Errorf("Bucket %s Key %s error:\n%s", bucket, key, err)
			return
		}
		content, err := ioutil.ReadAll(output.Body)
		if err != nil {
			T.Errorf("Bucket %s Key %s error:\n%s", bucket, key, err)
			return
		}
		act := string(content)
		if act != exp {
			T.Errorf("%s Key %s contents expected:\n%s\ngot:\n%s", bucket, key, exp, act)
			return
		}
	})
}
