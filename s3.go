//
// s3 - Swiss army pen-knife for Amazon S3.
//
//   https://github.com/barnybug/s3
//
// Copyright (c) 2014 Barnaby Gray

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

var reBucketPath = regexp.MustCompile("^(?:s3://)?([^/]+)/?(.*)$")

func extractBucketPath(conn *s3.S3, url string) (*s3.Bucket, string) {
	parts := reBucketPath.FindStringSubmatch(url)
	b := conn.Bucket(parts[1])
	return b, parts[2]
}

func listBuckets(conn *s3.S3) {
	data, err := conn.ListBuckets()
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, b := range data.Buckets {
		fmt.Println(b.Name)
	}
}

func iterateKeys(conn *s3.S3, bucket *s3.Bucket, prefix string, callback func(key *s3.Key)) {
	truncated := true
	marker := ""
	for truncated {
		data, err := bucket.List(prefix, "", marker, 0)
		if err != nil {
			log.Fatal(err.Error())
		}
		last_key := ""
		for _, c := range data.Contents {
			k := c
			callback(&k)
			last_key = c.Key
		}
		truncated = data.IsTruncated
		marker = data.NextMarker
		if marker == "" {
			// Response may not include NextMarker.
			marker = last_key
		}
	}
}

func iterateKeysParallel(conn *s3.S3, bucket *s3.Bucket, prefix string, callback func(key *s3.Key)) {
	// create pool for processing
	wg := sync.WaitGroup{}
	q := make(chan *s3.Key, 1000)
	for i := 0; i < parallel; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range q {
				callback(key)
			}
		}()
	}

	iterateKeys(conn, bucket, prefix, func(key *s3.Key) {
		q <- key
	})

	close(q)
	wg.Wait()
}

func listKeys(conn *s3.S3, url string) {
	bucket, prefix := extractBucketPath(conn, url)
	iterateKeys(conn, bucket, prefix, func(key *s3.Key) {
		fmt.Printf("s3://%s/%s\n", bucket.Name, key.Key)
	})
}

func getKeys(conn *s3.S3, url string) {
	bucket, prefix := extractBucketPath(conn, url)
	prefixPath := path.Dir(prefix)
	if prefixPath == "." {
		prefixPath = ""
	} else {
		prefixPath = prefixPath + "/"
	}

	iterateKeysParallel(conn, bucket, prefix, func(key *s3.Key) {
		reader, err := bucket.GetReader(key.Key)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		// write files under relative path to the source path
		fpath := key.Key[len(prefixPath):]
		dirpath := path.Dir(fpath)
		if dirpath != "." {
			err = os.MkdirAll(dirpath, 0777)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		writer, err := os.Create(fpath)
		if err != nil {
			log.Fatal(err.Error())
		}
		nbytes, err := io.Copy(writer, reader)
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Printf("s3://%s/%s -> %s (%d bytes)\n", bucket.Name, key.Key, fpath, nbytes)
	})
}

func catKeys(conn *s3.S3, url string) {
	bucket, prefix := extractBucketPath(conn, url)
	iterateKeys(conn, bucket, prefix, func(key *s3.Key) {
		reader, err := bucket.GetReader(key.Key)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			log.Fatal(err.Error())
		}
	})
}

type File interface {
	Name() string
	Size() int64
	MD5() []byte
	Reader() (io.ReadCloser, error)
}

type Filesystem interface {
	Files() <-chan File
	Create(src File) error
	Delete(path string) error
}

type LocalFilesystem struct {
	path string
}

type LocalFile struct {
	info     os.FileInfo
	fullpath string
	relpath  string
	md5      []byte
}

func (self *LocalFile) Name() string {
	return self.relpath
}

func (self *LocalFile) Size() int64 {
	return self.info.Size()
}

func (self *LocalFile) MD5() []byte {
	if self.md5 == nil {
		// cache md5
		h := md5.New()
		reader, err := os.Open(self.fullpath)
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(h, reader)
		if err != nil {
			log.Fatal(err)
		}
		self.md5 = h.Sum(nil)
	}
	return self.md5
}

func (self *LocalFile) Reader() (io.ReadCloser, error) {
	return os.Open(self.fullpath)
}

func (self *LocalFile) String() string {
	return self.relpath
}

func scanFiles(ch chan<- File, fullpath string, relpath string) {
	entries, err := ioutil.ReadDir(fullpath)
	if os.IsNotExist(err) {
		// this is fine - indicates no files are there
		return
	}
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, entry := range entries {
		f := filepath.Join(fullpath, entry.Name())
		r := filepath.Join(relpath, entry.Name())
		if entry.IsDir() {
			// recurse
			scanFiles(ch, f, r)
		} else {
			ch <- &LocalFile{entry, f, r, nil}
		}
	}
}

func (self *LocalFilesystem) Files() <-chan File {
	ch := make(chan File, 1000)
	go func() {
		defer close(ch)
		scanFiles(ch, self.path, "")
	}()
	return ch
}

func (self *LocalFilesystem) Create(src File) error {
	reader, err := src.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()
	fullpath := filepath.Join(self.path, src.Name())
	dirpath := filepath.Dir(fullpath)
	err = os.MkdirAll(dirpath, 0777)
	if err != nil {
		return err
	}
	writer, err := os.Create(fullpath)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	return err
}

func (self *LocalFilesystem) Delete(path string) error {
	fullpath := filepath.Join(self.path, path)
	return os.Remove(fullpath)
}

type S3Filesystem struct {
	conn   *s3.S3
	bucket *s3.Bucket
	path   string
}

type S3File struct {
	bucket *s3.Bucket
	key    *s3.Key
	path   string
	md5    []byte
}

func (self *S3File) Name() string {
	return self.path
}

func (self *S3File) Size() int64 {
	return self.key.Size
}

func (self *S3File) MD5() []byte {
	if self.md5 == nil {
		v := self.key.ETag[1 : len(self.key.ETag)-1]
		self.md5, _ = hex.DecodeString(v)
	}
	return self.md5
}

func (self *S3File) Reader() (io.ReadCloser, error) {
	resp, err := self.bucket.GetResponse(self.key.Key)
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}

func (self *S3File) String() string {
	return fmt.Sprintf("s3://%s/%s", self.bucket.Name, self.key.Key)
}

func (self *S3Filesystem) Files() <-chan File {
	ch := make(chan File, 1000)
	go func() {
		defer close(ch)
		iterateKeys(self.conn, self.bucket, self.path, func(key *s3.Key) {
			relpath := key.Key[len(self.path):]
			ch <- &S3File{self.bucket, key, relpath, nil}
		})
	}()
	return ch
}

func (self *S3Filesystem) Create(src File) error {
	var reader io.ReadCloser
	var contType string
	switch t := src.(type) {
	case *S3File:
		// special case for S3File to preserve header information
		resp, err := t.bucket.GetResponse(t.key.Key)
		if err != nil {
			return err
		}
		reader = resp.Body
		defer reader.Close()
		contType = resp.Header.Get("Content-Type")
	default:
		var err error
		reader, err = src.Reader()
		if err != nil {
			return err
		}
		defer reader.Close()
		// TODO: content-type
		contType = "application/binary"
		// TODO: acl
	}

	fullpath := filepath.Join(self.path, src.Name())
	err := self.bucket.PutReader(fullpath, reader, src.Size(), contType, s3.PublicRead)
	return err
}

func (self *S3Filesystem) Delete(path string) error {
	fullpath := filepath.Join(self.path, path)
	return self.bucket.Del(fullpath)
}

func getFilesystem(conn *s3.S3, url string) Filesystem {
	if strings.HasPrefix(url, "s3:") {
		bucket, prefix := extractBucketPath(conn, url)
		return &S3Filesystem{conn: conn, bucket: bucket, path: prefix}
	} else {
		return &LocalFilesystem{path: url}
	}
}

type Action struct {
	Action string
	File   File
}

func processAction(action Action, fs2 Filesystem) {
	switch action.Action {
	case "create":
		fmt.Printf("A %s\n", action.File.Name())
		err := fs2.Create(action.File)
		if err != nil {
			log.Fatal(err)
		}
	case "delete":
		fmt.Printf("D %s\n", action.File.Name())
		err := fs2.Delete(action.File.Name())
		if err != nil {
			log.Fatal(err)
		}
	case "update":
		fmt.Printf("U %s\n", action.File.Name())
		err := fs2.Create(action.File)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func syncFiles(conn *s3.S3, url1, url2 string) {
	start := time.Now()
	fs1 := getFilesystem(conn, url1)
	fs2 := getFilesystem(conn, url2)
	ch1 := fs1.Files()
	ch2 := fs2.Files()

	// create pool for processing
	wg := sync.WaitGroup{}
	q := make(chan Action, 1000)
	for i := 0; i < parallel; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for action := range q {
				processAction(action, fs2)
			}
		}()
	}

	f1 := <-ch1
	f2 := <-ch2
	var added, deleted, updated, unchanged int
	for {
		// iterate files in fs1 and fs2
		// if f1 is nil and f2 is nil, we're done
		// if f1 is nil or f1 < f2, create f1
		// if f2 is nil or f1 > f2, delete f2
		// if f1 = f2, check size, md5
		if f1 == nil && f2 == nil {
			break
		} else if f2 == nil || (f1 != nil && f1.Name() < f2.Name()) {
			q <- Action{"create", f1}
			added += 1
			f1 = <-ch1
		} else if f1 == nil || (f2 != nil && f1.Name() > f2.Name()) {
			q <- Action{"delete", f2}
			deleted += 1
			f2 = <-ch2
		} else if f1.Size() != f2.Size() || bytes.Compare(f1.MD5(), f2.MD5()) != 0 {
			// fmt.Println(f1.Name(), f2.Name(), f1.Size(), f2.Size(), f1.MD5(), f2.MD5())
			q <- Action{"update", f1}
			updated += 1
			f1 = <-ch1
			f2 = <-ch2
		} else {
			unchanged += 1
			f1 = <-ch1
			f2 = <-ch2
		}
	}

	close(q)
	wg.Wait()

	end := time.Now()
	took := end.Sub(start)
	rate := float64(added+deleted+updated) / took.Seconds()

	fmt.Printf(`-- summary --
%d added %d deleted %d updated %d unchanged
took: %s (%.1f ops/s)

`, added, deleted, updated, unchanged, took, rate)
}

var (
	parallel int
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: s3 COMMAND [arg...]

Commands:
	ls		List buckets or keys
	get 	Download keys
	cat		Cat keys
	sync	Synchronise local to s3, s3 to s3 or s3 to local
`)
		flag.PrintDefaults()
	}
	flag.IntVar(&parallel, "p", 32, "Number of parallel operations to run")

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(-1)
	}

	command := os.Args[1]
	// pop out the command argument
	os.Args = append(os.Args[0:1], os.Args[2:]...)
	flag.Parse()

	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err.Error())
	}
	conn := s3.New(auth, aws.EUWest)

	switch command {
	case "ls":
		if len(flag.Args()) < 2 {
			listBuckets(conn)
		} else {
			listKeys(conn, flag.Arg(0))
		}
	case "get":
		getKeys(conn, flag.Arg(0))
	case "cat":
		catKeys(conn, flag.Arg(0))
	case "sync":
		syncFiles(conn, flag.Arg(0), flag.Arg(1))
	default:
		flag.Usage()
		os.Exit(-1)
	}
}
