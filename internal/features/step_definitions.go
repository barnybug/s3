package features

import (
	"bytes"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awss3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/barnybug/s3"
	. "github.com/lsegal/gucumber"
)

var conn s3.S3er
var out bytes.Buffer

var replacer = strings.NewReplacer(`\n`, "\n", `\t`, "\t")

func init() {
	Before("", func() {
		conn = s3.NewMockS3()
		out = bytes.Buffer{}
	})

	Given(`^I have bucket "(.+?)"$`, func(bucket string) {
		input := awss3.CreateBucketInput{
			Bucket: aws.String(bucket),
		}
		conn.CreateBucket(&input)
	})

	Given(`^bucket "(.+?)" has key "(.+?)" containing "(.+?)"$`, func(bucket string, key string, content string) {
		body := bytes.NewReader([]byte(content))
		input := awss3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   body,
		}
		conn.PutObject(&input)
	})

	When(`^I run "(.+?)"$`, func(s1 string) {
		args := strings.Split(s1, " ")
		s3.Main(conn, args, &out)
	})

	Then(`^local file "(.+?)" has contents "(.+?)"$`, func(s1 string, s2 string) {
	})

	Then(`^the output is "(.*?)"$`, func(exp string) {
		// replace newlines
		exp = replacer.Replace(exp)
		act := string(out.Bytes())
		if act != exp {
			T.Errorf("Output expected:\n%s\ngot:\n%s", exp, act)
		}
	})
}
