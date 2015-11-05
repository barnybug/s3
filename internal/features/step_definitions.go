package features

import (
	"strings"

	. "github.com/lsegal/gucumber"
)

var s3 S3er

func init() {
	Given(`^I have empty bucket "(.+?)"$`, func(s1 string) {
		s3 := MockS3{}
	})

	And(`^the bucket "(.+?)" has a key "(.+?)" with contents "(.+?)"$`, func(s1 string, s2 string, s3 string) {
	})

	When(`^I run "(.+?)"$`, func(s1 string) {
		parts := strings.Split(s1)

	})

	Then(`^local file "(.+?)" has contents "(.+?)"$`, func(s1 string, s2 string) {
	})

}
