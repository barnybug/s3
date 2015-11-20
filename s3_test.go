package s3

import (
	"bytes"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/mitchellh/goamz/testutil"
	. "github.com/motain/gocheck"
)

func Test(t *testing.T) {
	TestingT(t)
}

type S struct {
	s3      *s3.S3
	out     bytes.Buffer
	tempdir string
}

var _ = Suite(&S{})

var testServer = testutil.NewHTTPServer()

func (s *S) SetUpSuite(c *C) {
	testServer.Start()
	auth := aws.Auth{"abc", "123", ""}
	s.s3 = s3.New(auth, aws.Region{Name: "faux-region-1", S3Endpoint: testServer.URL})
}

func (s *S) SetUpTest(c *C) {
	// catch 'stdout'
	s.out = bytes.Buffer{}
	out = &s.out
	// run in temporary directory
	s.tempdir, _ = ioutil.TempDir("", "test")
	syscall.Chdir(s.tempdir)
}

func (s *S) TearDownTest(c *C) {
	testServer.Flush()
	syscall.Unlink(s.tempdir)
}

var GetListBucketsDump = `
<?xml version="1.0" encoding="UTF-8"?>
<ListAllMyBucketsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Owner>
    <ID>bb5c0f63b0b25f2d0</ID>
    <DisplayName>joe</DisplayName>
  </Owner>
  <Buckets>
    <Bucket>
      <Name>bucket1</Name>
      <CreationDate>2012-01-01T02:03:04.000Z</CreationDate>
    </Bucket>
    <Bucket>
      <Name>bucket2</Name>
      <CreationDate>2014-01-11T02:03:04.000Z</CreationDate>
    </Bucket>
  </Buckets>
</ListAllMyBucketsResult>
`

func (s *S) TestLsBuckets(c *C) {
	testServer.Response(200, nil, GetListBucketsDump)

	run(s.s3, []string{"s3", "ls"})

	c.Assert(s.out.String(), Equals, "s3://bucket1/\ns3://bucket2/\n")
}

var GetListResultDump1 = `
<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01">
  <Name>quotes</Name>
  <Prefix>N</Prefix>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>Nelson</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>5</Size>
    <StorageClass>STANDARD</StorageClass>
    <Owner>
      <ID>bcaf161ca5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
     </Owner>
  </Contents>
  <Contents>
    <Key>Neo</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>4</Size>
    <StorageClass>STANDARD</StorageClass>
     <Owner>
      <ID>bcaf1ffd86a5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
    </Owner>
 </Contents>
</ListBucketResult>
`

var GetListResultDump2 = `
<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01">
  <Name>quotes</Name>
  <Prefix>N</Prefix>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>Leo</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>4</Size>
    <StorageClass>STANDARD</StorageClass>
     <Owner>
      <ID>bcaf1ffd86a5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
    </Owner>
 </Contents>
  <Contents>
    <Key>Nelson</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>5</Size>
    <StorageClass>STANDARD</StorageClass>
    <Owner>
      <ID>bcaf161ca5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
     </Owner>
  </Contents>
</ListBucketResult>
`

var GetListResultDump3 = `
<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01">
  <Name>quotes</Name>
  <Prefix>N</Prefix>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>test/Nelson</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>5</Size>
    <StorageClass>STANDARD</StorageClass>
    <Owner>
      <ID>bcaf161ca5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
     </Owner>
  </Contents>
  <Contents>
    <Key>test/Neo</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>4</Size>
    <StorageClass>STANDARD</StorageClass>
     <Owner>
      <ID>bcaf1ffd86a5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
    </Owner>
 </Contents>
</ListBucketResult>
`

var GetListResultDump4 = `
<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01">
  <Name>quotes</Name>
  <Prefix>N</Prefix>
  <IsTruncated>false</IsTruncated>
  <Contents>
    <Key>test/Nelson</Key>
    <LastModified>2006-01-01T12:00:00.000Z</LastModified>
    <ETag>&quot;828ef3fdfa96f00ad9f27c383fc9ac7f&quot;</ETag>
    <Size>5</Size>
    <StorageClass>STANDARD</StorageClass>
    <Owner>
      <ID>bcaf161ca5fb16fd081034f</ID>
      <DisplayName>webfile</DisplayName>
     </Owner>
  </Contents>
</ListBucketResult>
`

func (s *S) TestLsKeys(c *C) {
	testServer.Response(200, nil, GetListResultDump1)

	run(s.s3, []string{"s3", "ls", "s3://bucket/"})

	c.Assert(s.out.String(), Equals, "s3://bucket/Nelson\t5b\ns3://bucket/Neo\t4b\n\n2 files, 9 bytes\n")
}

func (s *S) TestCat(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "abcd")
	testServer.Response(200, nil, "efghi")

	run(s.s3, []string{"s3", "cat", "-p=1", "s3://bucket/"})

	c.Assert(s.out.String(), Equals, "abcdefghi")
}

func listFiles(folder string) []string {
	var files []string
	fis, _ := ioutil.ReadDir(folder)
	for _, fi := range fis {
		files = append(files, fi.Name())
	}
	return files
}

func (s *S) TestGetWholeBucket(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "abcd")
	testServer.Response(200, nil, "efghi")

	run(s.s3, []string{"s3", "get", "-p", "1", "s3://bucket/"})

	files := listFiles(".")
	c.Assert(files, DeepEquals, []string{"Nelson", "Neo"})
	c.Assert(s.out.String(), Equals, "s3://bucket/Nelson -> Nelson (4 bytes)\ns3://bucket/Neo -> Neo (5 bytes)\n")
}

func (s *S) TestGetSubkeys(c *C) {
	testServer.Response(200, nil, GetListResultDump3)
	testServer.Response(200, nil, "abcd")
	testServer.Response(200, nil, "efghi")

	run(s.s3, []string{"s3", "get", "-p", "1", "s3://bucket/test/"})

	files := listFiles(".")
	c.Assert(files, DeepEquals, []string{"Nelson", "Neo"})
	c.Assert(s.out.String(), Equals, "s3://bucket/test/Nelson -> Nelson (4 bytes)\ns3://bucket/test/Neo -> Neo (5 bytes)\n")
}

func (s *S) TestGetDir(c *C) {
	testServer.Response(200, nil, GetListResultDump3)
	testServer.Response(200, nil, "abcd")
	testServer.Response(200, nil, "efghi")

	run(s.s3, []string{"s3", "get", "-p", "1", "s3://bucket/test"})

	files := listFiles(".")
	c.Assert(files, DeepEquals, []string{"test"})
	c.Assert(s.out.String(), Equals, "s3://bucket/test/Nelson -> test/Nelson (4 bytes)\ns3://bucket/test/Neo -> test/Neo (5 bytes)\n")
}

func (s *S) TestGetSingleKey(c *C) {
	testServer.Response(200, nil, GetListResultDump4)
	testServer.Response(200, nil, "abcd")

	run(s.s3, []string{"s3", "get", "-p", "1", "s3://bucket/test/Nelson"})

	files := listFiles(".")
	c.Assert(files, DeepEquals, []string{"Nelson"})
	c.Assert(s.out.String(), Equals, "s3://bucket/test/Nelson -> Nelson (4 bytes)\n")
}

func (s *S) TestPut(c *C) {
	testServer.Response(200, nil, "")

	f, _ := os.Create("xyz")
	f.WriteString("1")
	f.Close()

	run(s.s3, []string{"s3", "put", "xyz", "s3://bucket/"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "PUT")
	c.Assert(req.URL.Path, Equals, "/bucket/xyz")
}

func (s *S) TestRm(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "")

	run(s.s3, []string{"s3", "rm", "s3://bucket/"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")

	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "POST")
	c.Assert(req.RequestURI, Equals, "/bucket/?delete=")
	data, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	c.Assert(err, IsNil)
	c.Assert(string(data), Equals, "<Delete><Quiet>false</Quiet><Object><Key>Nelson</Key></Object><Object><Key>Neo</Key></Object></Delete>")
}

func setupFiles1() {
	os.Mkdir("folder1", 0700)
	f, _ := os.Create("folder1/xyz")
	f.WriteString("1")
	f.Close()
}

func (s *S) TestSyncLocalToS3(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")

	setupFiles1()

	run(s.s3, []string{"s3", "sync", "-p=1", "folder1", "s3://bucket/"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "PUT")
}

func (s *S) TestSyncLocalToS3Deletes(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")

	setupFiles1()

	run(s.s3, []string{"s3", "sync", "-p=1", "-delete", "folder1", "s3://bucket/"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "DELETE")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "DELETE")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "PUT")
	c.Assert(req.URL.Path, Equals, "/bucket/xyz")
}

func (s *S) TestSyncS3ToLocal(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")

	setupFiles1()

	run(s.s3, []string{"s3", "sync", "-p", "1", "-delete", "s3://bucket/", "folder1"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket/Nelson")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket/Neo")

	files := listFiles("folder1")
	c.Assert(files, DeepEquals, []string{"Nelson", "Neo"})
}

func (s *S) TestSyncS3ToLocalEmpty(c *C) {
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")

	run(s.s3, []string{"s3", "sync", "-p", "1", "s3://bucket/", "folder1"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket/Nelson")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket/Neo")

	files := listFiles("folder1")
	c.Assert(files, DeepEquals, []string{"Nelson", "Neo"})
}

func (s *S) TestSyncS3ToS3(c *C) {
	testServer.Response(200, nil, GetListResultDump2)
	testServer.Response(200, nil, GetListResultDump1)
	testServer.Response(200, nil, "abcd")
	testServer.Response(200, nil, "")
	testServer.Response(200, nil, "")

	run(s.s3, []string{"s3", "sync", "-p=1", "-delete", "s3://bucket/", "s3://bucket2/"})

	req := testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket/")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket2/")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "GET")
	c.Assert(req.URL.Path, Equals, "/bucket/Leo")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "PUT")
	c.Assert(req.URL.Path, Equals, "/bucket2/Leo")
	req = testServer.WaitRequest()
	c.Assert(req.Method, Equals, "DELETE")
	c.Assert(req.URL.Path, Equals, "/bucket2/Neo")
}
