//
// s3 - Swiss army pen-knife for Amazon S3.
//
//   https://github.com/barnybug/s3
//
// Copyright (c) 2014 Barnaby Gray

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
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
		if dryRun {
			return
		}
		err := fs2.Create(action.File)
		if err != nil {
			log.Fatal(err)
		}
	case "delete":
		fmt.Printf("D %s\n", action.File.Name())
		if dryRun {
			return
		}
		err := fs2.Delete(action.File.Name())
		if err != nil {
			log.Fatal(err)
		}
	case "update":
		fmt.Printf("U %s\n", action.File.Name())
		if dryRun {
			return
		}
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
			if delete {
				q <- Action{"delete", f2}
				deleted += 1
			}
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

	if dryRun {
		fmt.Println("-- summary (dry-run) --")
	} else {
		fmt.Println("-- summary --")
	}
	fmt.Printf(`%d added %d deleted %d updated %d unchanged
took: %s (%.1f ops/s)

`, added, deleted, updated, unchanged, took, rate)
}

var (
	parallel int
	dryRun   bool
	delete   bool
	public   bool
	acl      string
)

var ValidACLs = map[string]bool{
	"private":                   true,
	"public-read":               true,
	"public-read-write":         true,
	"authenticated-read":        true,
	"bucket-owner-read":         true,
	"bucket-owner-full-control": true,
	"log-delivery-write":        true,
}

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
	flag.IntVar(&parallel, "p", 32, "number of parallel operations to run")
	flag.BoolVar(&dryRun, "n", false, "dry-run, no actions taken")
	flag.BoolVar(&delete, "delete", false, "delete extraneous files from destination")
	flag.BoolVar(&public, "P", false, "shortcut to set acl to public-read")
	flag.StringVar(&acl, "acl", "", "set acl to one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write")

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(-1)
	}

	command := os.Args[1]
	// pop out the command argument
	os.Args = append(os.Args[0:1], os.Args[2:]...)
	flag.Parse()
	if public {
		acl = "public-read"
	}
	if acl != "" && !ValidACLs[acl] {
		fmt.Println("-acl should be one of: private, public-read, public-read-write, authenticated-read, bucket-owner-read, bucket-owner-full-control, log-delivery-write")
		os.Exit(-1)
	}

	auth, err := aws.EnvAuth()
	if err != nil {
		panic(err.Error())
	}
	conn := s3.New(auth, aws.EUWest)

	switch command {
	case "ls":
		if len(flag.Args()) < 1 {
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
