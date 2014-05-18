//
// s3 - Swiss army pen-knife for Amazon S3.
//
//   https://github.com/barnybug/s3
//
// Copyright (c) 2014 Barnaby Gray

package main

import (
	"bytes"
	"compress/gzip"
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

func iterateKeys(conn *s3.S3, urls []string, callback func(file File)) {
	for _, url := range urls {
		fs := getFilesystem(conn, url)
		ch := fs.Files()
		for file := range ch {
			callback(file)
		}
	}
}

func iterateKeysParallel(conn *s3.S3, urls []string, callback func(file File)) {
	// create pool for processing
	wg := sync.WaitGroup{}
	q := make(chan File, 1000)
	for i := 0; i < parallel; i += 1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range q {
				callback(key)
			}
		}()
	}

	iterateKeys(conn, urls, func(file File) {
		q <- file
	})

	close(q)
	wg.Wait()
}

func listKeys(conn *s3.S3, urls []string) {
	var count, totalSize int64
	iterateKeys(conn, urls, func(file File) {
		if quiet {
			fmt.Println(file)
		} else {
			fmt.Printf("%s\t%db\n", file, file.Size())
		}
		count += 1
		totalSize += file.Size()
	})
	if !quiet {
		fmt.Printf("\n%d files, %d bytes\n", count, totalSize)
	}
}

func getKeys(conn *s3.S3, urls []string) {
	iterateKeysParallel(conn, urls, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		// write files under relative path to the source path
		fpath := file.Relative()
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
		if !quiet {
			fmt.Printf("%s -> %s (%d bytes)\n", file, fpath, nbytes)
		}
	})
}

func catKeys(conn *s3.S3, urls []string) {
	iterateKeysParallel(conn, urls, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		if strings.HasSuffix(file.String(), ".gz") {
			reader, err = gzip.NewReader(reader)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			log.Fatal(err.Error())
		}
	})
}

func rmKeys(conn *s3.S3, urls []string) {
	batch := make([]string, 0, 1000)
	var bucket *s3.Bucket
	start := time.Now()
	var deleted int
	iterateKeys(conn, urls, func(file File) {
		deleted += 1
		if !quiet {
			fmt.Printf("D %s\n", file)
		}
		switch t := file.(type) {
		case *S3File:
			// optimize as a batch delete
			if bucket != nil && t.bucket != bucket && len(batch) > 0 {
				if !dryRun {
					bucket.MultiDel(batch)
				}
				batch = batch[:0]
			}
			bucket = t.bucket
			batch = append(batch, t.key.Key)
			if len(batch) == 1000 {
				if !dryRun {
					// send batch delete
					bucket.MultiDel(batch)
				}
				batch = batch[:0]
			}

		default:
			if !dryRun {
				file.Delete()
			}
		}
	})

	// final batch
	if len(batch) > 0 {
		if !dryRun {
			bucket.MultiDel(batch)
		}
	}
	end := time.Now()
	took := end.Sub(start)
	summary(0, deleted, 0, 0, took)
}

func summary(added, deleted, updated, unchanged int, took time.Duration) {
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

func putKeys(conn *s3.S3, urls []string) {
	sources := urls[:len(urls)-1]
	destination := urls[len(urls)-1]
	start := time.Now()
	dfs := getFilesystem(conn, destination)
	var added int
	iterateKeysParallel(conn, sources, func(file File) {
		reader, err := file.Reader()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer reader.Close()

		if !quiet {
			fmt.Printf("A %s\n", file)
		}
		err = dfs.Create(file)
		if err != nil {
			log.Fatal(err.Error())
		}
		added += 1
	})
	end := time.Now()
	took := end.Sub(start)
	summary(added, 0, 0, 0, took)
}

type File interface {
	Relative() string
	Size() int64
	MD5() []byte
	Reader() (io.ReadCloser, error)
	Delete() error
	String() string
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
		if !quiet {
			fmt.Printf("A %s\n", action.File.Relative())
		}
		if dryRun {
			return
		}
		err := fs2.Create(action.File)
		if err != nil {
			log.Fatal(err)
		}
	case "delete":
		if !quiet {
			fmt.Printf("D %s\n", action.File.Relative())
		}
		if dryRun {
			return
		}
		err := fs2.Delete(action.File.Relative())
		if err != nil {
			log.Fatal(err)
		}
	case "update":
		if !quiet {
			fmt.Printf("U %s\n", action.File.Relative())
		}
		if dryRun {
			return
		}
		err := fs2.Create(action.File)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func syncFiles(conn *s3.S3, urls []string) {
	if len(urls) != 2 {
		// TODO: support multiple sources -> single destination
		panic("Unsupported")
	}
	url1 := urls[0]
	url2 := urls[1]
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
		} else if f2 == nil || (f1 != nil && f1.Relative() < f2.Relative()) {
			q <- Action{"create", f1}
			added += 1
			f1 = <-ch1
		} else if f1 == nil || (f2 != nil && f1.Relative() > f2.Relative()) {
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
	summary(added, deleted, updated, unchanged, took)
}

var (
	parallel int
	dryRun   bool
	delete   bool
	public   bool
	quiet    bool
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

var minArgs = map[string]int{
	"cat":  1,
	"get":  1,
	"ls":   0,
	"put":  2,
	"rm":   1,
	"sync": 2,
}

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: s3 COMMAND [source...] [destination]

Commands:
	cat	Cat key contents
	get	Download keys
	ls	List buckets or keys
	put 	Upload files
	rm	Delete keys
	sync	Synchronise local to s3, s3 to s3 or s3 to local

Options:
`)
		flag.PrintDefaults()
	}
	flag.IntVar(&parallel, "p", 32, "number of parallel operations to run")
	flag.BoolVar(&dryRun, "n", false, "dry-run, no actions taken")
	flag.BoolVar(&delete, "delete", false, "delete extraneous files from destination (sync)")
	flag.BoolVar(&public, "P", false, "shortcut to set acl to public-read")
	flag.BoolVar(&quiet, "q", false, "quieter (less verbose) output")
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

	if _, ok := minArgs[command]; !ok {
		flag.Usage()
		os.Exit(-1)
	}

	if len(flag.Args()) < minArgs[command] {
		fmt.Println("More arguments required\n")
		flag.Usage()
		os.Exit(-1)
	}

	switch command {
	case "cat":
		catKeys(conn, flag.Args())
	case "get":
		getKeys(conn, flag.Args())
	case "ls":
		if len(flag.Args()) < 1 {
			listBuckets(conn)
		} else {
			listKeys(conn, flag.Args())
		}
	case "put":
		putKeys(conn, flag.Args())
	case "sync":
		syncFiles(conn, flag.Args())
	case "rm":
		rmKeys(conn, flag.Args())
	}
}
